package discovery

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LocalNetwork describes the host's address on its primary local network.
type LocalNetwork struct {
	IP     net.IP
	Subnet string // CIDR notation, e.g. "192.168.1.0/24"
}

// LocalNetworkInfo returns the IP and CIDR subnet of the first non-loopback,
// active IPv4 network interface it finds.
func LocalNetworkInfo() (LocalNetwork, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return LocalNetwork{}, fmt.Errorf("listing network interfaces: %w", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip4 := ipNet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}

			ones, _ := ipNet.Mask.Size()
			network := ip4.Mask(ipNet.Mask)

			return LocalNetwork{
				IP:     ip4,
				Subnet: fmt.Sprintf("%s/%d", network.String(), ones),
			}, nil
		}
	}

	return LocalNetwork{}, fmt.Errorf("no non-loopback IPv4 network interface found")
}

// Host is a device discovered on the local network.
type Host struct {
	IP       string
	Hostname string        // populated by NmapScan when nmap resolves a name; empty otherwise
	RTT      time.Duration // populated by PingSweep; zero for NmapScan
}

var nmapHostLineRe = regexp.MustCompile(`^Host: (\S+) \(([^)]*)\)\s+Status: Up`)

// NmapScan runs `nmap -sn` (a ping scan) against cidr and returns the hosts
// found up in its greppable output. It returns an error if nmap isn't
// installed, so callers can fall back to PingSweep.
func NmapScan(cidr string) ([]Host, error) {
	nmapPath, err := exec.LookPath("nmap")
	if err != nil {
		return nil, fmt.Errorf("nmap not available: %w", err)
	}

	out, err := exec.Command(nmapPath, "-sn", "-oG", "-", cidr).Output()
	if err != nil {
		return nil, fmt.Errorf("running nmap: %w", err)
	}

	var hosts []Host
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		m := nmapHostLineRe.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		hosts = append(hosts, Host{IP: m[1], Hostname: m[2]})
	}

	return hosts, nil
}

// PingSweep is the fallback discovery method for when nmap isn't installed.
// It concurrently pings every host address in cidr, using at most
// maxConcurrency workers at once, and returns the hosts that responded
// along with their round-trip time. It shells out to the OS ping utility so
// it works without elevated privileges.
func PingSweep(cidr string, maxConcurrency int) ([]Host, error) {
	ips, err := hostsInCIDR(cidr)
	if err != nil {
		return nil, err
	}
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		hosts []Host
		sem   = make(chan struct{}, maxConcurrency)
	)

	for _, ip := range ips {
		ip := ip
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			rtt, up := pingHost(ip.String())
			if !up {
				return
			}

			mu.Lock()
			hosts = append(hosts, Host{IP: ip.String(), RTT: rtt})
			mu.Unlock()
		}()
	}

	wg.Wait()
	return hosts, nil
}

// hostsInCIDR enumerates every address in cidr, excluding the network and
// broadcast addresses when the subnet is large enough to have distinct ones.
func hostsInCIDR(cidr string) ([]net.IP, error) {
	base, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("parsing CIDR %q: %w", cidr, err)
	}

	var ips []net.IP
	for ip := base.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, append(net.IP(nil), ip...))
	}

	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

var pingRTTRe = regexp.MustCompile(`time[=<]([\d.]+)\s*ms`)

// pingHost sends a single ICMP echo via the OS ping command and reports
// whether the host responded and its round-trip time.
func pingHost(ip string) (time.Duration, bool) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", "1000", ip)
	} else {
		cmd = exec.Command("ping", "-c", "1", "-W", "1", ip)
	}

	start := time.Now()
	out, err := cmd.Output()
	if err != nil {
		return 0, false
	}

	if m := pingRTTRe.FindSubmatch(out); m != nil {
		if ms, err := strconv.ParseFloat(string(m[1]), 64); err == nil {
			return time.Duration(ms * float64(time.Millisecond)), true
		}
	}

	return time.Since(start), true
}
