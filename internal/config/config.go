// Package config holds warp-speed's runtime configuration: which network
// interface to monitor, which host to continuously ping in the background,
// and how often to refresh the dashboard.
package config

import (
	"flag"
	"time"
)

// Config is the fully resolved runtime configuration for the app.
type Config struct {
	// Interface is the network interface to monitor for bandwidth.
	// Empty string means "auto-detect the default route interface".
	Interface string

	// PingHost is the host continuously pinged in the background to
	// show a live latency reading on the dashboard.
	PingHost string

	// RefreshInterval controls how often bandwidth and the live ping
	// sample are refreshed.
	RefreshInterval time.Duration

	// PingTimeout bounds how long a single ping probe may wait for a reply.
	PingTimeout time.Duration

	// TestCount is how many probes an on-demand ping test sends.
	TestCount int
}

// Default returns warp-speed's built-in default configuration.
func Default() Config {
	return Config{
		Interface:       "",
		PingHost:        "8.8.8.8",
		RefreshInterval: time.Second,
		PingTimeout:     1500 * time.Millisecond,
		TestCount:       5,
	}
}

// FromFlags parses command-line flags into a Config, starting from
// Default() so any flag the user omits keeps its sensible default.
func FromFlags() Config {
	cfg := Default()

	iface := flag.String("iface", cfg.Interface, "network interface to monitor (default: auto-detect)")
	host := flag.String("host", cfg.PingHost, "host to continuously ping in the background")
	interval := flag.Duration("interval", cfg.RefreshInterval, "dashboard refresh interval, e.g. 500ms, 1s")
	timeout := flag.Duration("timeout", cfg.PingTimeout, "timeout for a single ping probe")
	count := flag.Int("count", cfg.TestCount, "number of probes sent for an on-demand ping test")
	flag.Parse()

	cfg.Interface = *iface
	cfg.PingHost = *host
	cfg.RefreshInterval = *interval
	cfg.PingTimeout = *timeout
	cfg.TestCount = *count

	return cfg
}
