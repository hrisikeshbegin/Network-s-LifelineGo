package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultHeartbeatIntervalMS    = 60000
	defaultOfflineThresholdChecks = 3
)

// Config holds runtime settings for the agent, sourced from environment
// variables with sensible defaults applied when a variable is unset or
// cannot be parsed.
type Config struct {
	AgentID                string
	HeartbeatIntervalMS    int
	OfflineThresholdChecks int
}

// Load reads AGENT_ID, HEARTBEAT_INTERVAL_MS, and OFFLINE_THRESHOLD_CHECKS
// from the environment, falling back to defaults for any that are unset or
// invalid.
func Load() Config {
	return Config{
		AgentID:                loadAgentID(),
		HeartbeatIntervalMS:    loadInt("HEARTBEAT_INTERVAL_MS", defaultHeartbeatIntervalMS),
		OfflineThresholdChecks: loadInt("OFFLINE_THRESHOLD_CHECKS", defaultOfflineThresholdChecks),
	}
}

func loadAgentID() string {
	if v := strings.TrimSpace(os.Getenv("AGENT_ID")); v != "" {
		return v
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "unknown-agent"
	}
	return sanitizeHostname(hostname)
}

func loadInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// sanitizeHostname lowercases the hostname and replaces any character that
// isn't alphanumeric, '-', or '_' with '-', so it's safe to use as an
// identifier (e.g. in metric labels, file names, or topic names).
func sanitizeHostname(hostname string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(hostname) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
