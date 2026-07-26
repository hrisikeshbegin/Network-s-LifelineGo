package discovery

import (
	"fmt"
	"net"
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
