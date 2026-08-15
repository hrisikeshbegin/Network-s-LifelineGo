package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestUpsertDevice_PreservesFirstSeenAcrossRescan(t *testing.T) {
	s, err := NewJSONStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatalf("NewJSONStore returned error: %v", err)
	}

	ctx := context.Background()
	firstSeen := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := s.UpsertDevice(ctx, Device{ID: "1.2.3.4", IP: "1.2.3.4", Status: StatusOnline, FirstSeen: firstSeen, LastSeen: firstSeen}); err != nil {
		t.Fatalf("initial UpsertDevice returned error: %v", err)
	}

	// A later discovery pass re-upserts the same device with a fresh,
	// non-zero FirstSeen (mirroring main.go, which always stamps the
	// current scan time) — the store must keep the original FirstSeen.
	rescanTime := firstSeen.Add(24 * time.Hour)
	if err := s.UpsertDevice(ctx, Device{ID: "1.2.3.4", IP: "1.2.3.4", Status: StatusOnline, FirstSeen: rescanTime, LastSeen: rescanTime}); err != nil {
		t.Fatalf("rescan UpsertDevice returned error: %v", err)
	}

	devices, err := s.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices returned error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(devices))
	}

	got := devices[0]
	if !got.FirstSeen.Equal(firstSeen) {
		t.Errorf("FirstSeen = %v, want original %v (rescan overwrote it)", got.FirstSeen, firstSeen)
	}
	if !got.LastSeen.Equal(rescanTime) {
		t.Errorf("LastSeen = %v, want updated %v", got.LastSeen, rescanTime)
	}
}
