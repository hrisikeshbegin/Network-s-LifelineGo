package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hrisikeshbegin/go-network-lifeline/internal/config"
	"github.com/hrisikeshbegin/go-network-lifeline/internal/discovery"
	"github.com/hrisikeshbegin/go-network-lifeline/internal/store"
)

const (
	devicesFile          = "./devices.json"
	pingSweepConcurrency = 50
)

func main() {
	cfg := config.Load()
	log.Printf("starting agent %s (heartbeat every %dms)", cfg.AgentID, cfg.HeartbeatIntervalMS)

	st, err := store.NewJSONStore(devicesFile)
	if err != nil {
		log.Fatalf("opening store: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	netInfo, err := discovery.LocalNetworkInfo()
	if err != nil {
		log.Fatalf("detecting local network: %v", err)
	}
	log.Printf("scanning subnet %s (local IP %s)", netInfo.Subnet, netInfo.IP)

	hosts, err := discovery.NmapScan(netInfo.Subnet)
	if err != nil {
		log.Printf("nmap unavailable (%v); falling back to ping sweep", err)
		hosts, err = discovery.PingSweep(netInfo.Subnet, pingSweepConcurrency)
		if err != nil {
			log.Fatalf("ping sweep failed: %v", err)
		}
	}
	log.Printf("discovered %d device(s)", len(hosts))

	now := time.Now()
	knownIPs := make([]string, 0, len(hosts))
	for _, h := range hosts {
		device := store.Device{
			ID:        h.IP,
			IP:        h.IP,
			Hostname:  h.Hostname,
			Status:    store.StatusOnline,
			FirstSeen: now,
			LastSeen:  now,
		}
		if err := st.UpsertDevice(ctx, device); err != nil {
			log.Printf("upsert device %s: %v", h.IP, err)
			continue
		}
		knownIPs = append(knownIPs, h.IP)
	}

	ticker := time.NewTicker(time.Duration(cfg.HeartbeatIntervalMS) * time.Millisecond)
	defer ticker.Stop()

	log.Println("entering heartbeat loop (Ctrl+C to stop)")
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			heartbeat(ctx, st, knownIPs, pingSweepConcurrency)
		}
	}
}

// heartbeat re-pings each known device IP, bounded by maxConcurrency
// simultaneous pings, and records the result in st.
func heartbeat(ctx context.Context, st store.Store, ips []string, maxConcurrency int) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	for _, ip := range ips {
		ip := ip
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			status := store.StatusOffline
			if hosts, err := discovery.PingSweep(ip+"/32", 1); err == nil && len(hosts) > 0 {
				status = store.StatusOnline
			}

			if err := st.UpdateHeartbeat(ctx, ip, status, time.Now()); err != nil {
				log.Printf("update heartbeat for %s: %v", ip, err)
			}
		}()
	}

	wg.Wait()
}
