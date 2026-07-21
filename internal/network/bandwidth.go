// Package network provides bandwidth measurement, ICMP ping, and host
// validation used by warpping.
package network

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Sample is a snapshot of an interface's cumulative RX/TX byte counters.
type Sample struct {
	RxBytes   uint64
	TxBytes   uint64
	Timestamp time.Time
}

// ReadStats reads the cumulative RX/TX byte counters for iface from
// /proc/net/dev.
func ReadStats(iface string) (Sample, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return Sample{}, fmt.Errorf("reading /proc/net/dev: %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, ":") {
			continue // header lines
		}
		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		if name != iface {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		// Column layout after the interface name:
		// 0:rx_bytes 1:rx_packets ... 8:tx_bytes ...
		rx, err1 := strconv.ParseUint(fields[0], 10, 64)
		tx, err2 := strconv.ParseUint(fields[8], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		return Sample{RxBytes: rx, TxBytes: tx, Timestamp: time.Now()}, nil
	}

	return Sample{}, fmt.Errorf("interface %q not found in /proc/net/dev", iface)
}

// DefaultInterface returns the network interface used for the system's
// default route, read from /proc/net/route (destination 00000000).
func DefaultInterface() (string, error) {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return "", fmt.Errorf("reading /proc/net/route: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		iface, dest := fields[0], fields[1]
		if dest == "00000000" {
			return iface, nil
		}
	}

	return "", fmt.Errorf("could not determine the default route interface; pass -iface explicitly")
}

// Rates holds computed throughput between two samples, in bytes/sec.
type Rates struct {
	DownBytesPerSec float64
	UpBytesPerSec   float64
}

// ComputeRates derives throughput from two samples of the same interface.
// It guards against counter resets (e.g. interface bounce) by returning
// zero rates instead of a bogus negative/huge value.
func ComputeRates(prev, cur Sample) Rates {
	secs := cur.Timestamp.Sub(prev.Timestamp).Seconds()
	if secs <= 0 {
		return Rates{}
	}
	if cur.RxBytes < prev.RxBytes || cur.TxBytes < prev.TxBytes {
		return Rates{}
	}

	down := float64(cur.RxBytes-prev.RxBytes) / secs
	up := float64(cur.TxBytes-prev.TxBytes) / secs
	return Rates{DownBytesPerSec: down, UpBytesPerSec: up}
}

// Mbps converts a bytes/sec throughput value into megabits/sec.
func Mbps(bytesPerSec float64) float64 {
	return (bytesPerSec * 8) / 1_000_000
}
