package store

import (
	"context"
	"time"
)

// Status is a device's last-known reachability.
type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
)

// Device is a host discovered on the local network.
type Device struct {
	ID        string // stable identifier, e.g. MAC address or IP
	IP        string
	Hostname  string
	Status    Status
	FirstSeen time.Time
	LastSeen  time.Time
}

// Store persists discovered devices. Implementations include a local JSON
// file store and, later, a Firestore-backed store — so methods take a
// context and report errors rather than assuming in-process, synchronous
// storage.
type Store interface {
	// UpsertDevice inserts a new device or updates an existing one matched
	// by device.ID.
	UpsertDevice(ctx context.Context, device Device) error

	// UpdateHeartbeat records a heartbeat for the device with the given ID,
	// setting its status and last-seen timestamp.
	UpdateHeartbeat(ctx context.Context, id string, status Status, seenAt time.Time) error

	// ListDevices returns all known devices.
	ListDevices(ctx context.Context) ([]Device, error)
}
