package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONStore is a Store implementation backed by a single local JSON file.
// It keeps all devices in memory and, on every update, rewrites the whole
// file atomically (temp file + rename) under a mutex.
type JSONStore struct {
	mu      sync.Mutex
	path    string
	devices map[string]Device
}

var _ Store = (*JSONStore)(nil)

// NewJSONStore creates a JSONStore backed by path, loading any existing
// data if the file is already present.
func NewJSONStore(path string) (*JSONStore, error) {
	s := &JSONStore{
		path:    path,
		devices: make(map[string]Device),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if len(data) == 0 {
		return s, nil
	}

	if err := json.Unmarshal(data, &s.devices); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return s, nil
}

// UpsertDevice inserts a new device or updates an existing one matched by
// device.ID, preserving its original FirstSeen if it already had one.
func (s *JSONStore) UpsertDevice(_ context.Context, device Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.devices[device.ID]; ok && device.FirstSeen.IsZero() {
		device.FirstSeen = existing.FirstSeen
	}

	s.devices[device.ID] = device
	return s.saveLocked()
}

// UpdateHeartbeat records a heartbeat for the device with the given ID,
// setting its status and last-seen timestamp. It returns an error if the
// device hasn't been upserted before.
func (s *JSONStore) UpdateHeartbeat(_ context.Context, id string, status Status, seenAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device %q not found", id)
	}

	device.Status = status
	device.LastSeen = seenAt
	s.devices[id] = device

	return s.saveLocked()
}

// ListDevices returns all known devices.
func (s *JSONStore) ListDevices(_ context.Context) ([]Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	devices := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}
	return devices, nil
}

// saveLocked writes the current device map to disk atomically: it writes
// to a temp file in the same directory and renames it into place. Callers
// must hold s.mu.
func (s *JSONStore) saveLocked() error {
	data, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling devices: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".jsonstore-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file to %s: %w", s.path, err)
	}

	return nil
}
