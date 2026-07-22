package network

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// PingResult is the aggregate outcome of sending one or more ICMP echo
// requests to a single target.
type PingResult struct {
	Target   string
	Sent     int
	Received int
	MinMs    float64
	AvgMs    float64
	MaxMs    float64
	LastMs   float64
	LastOK   bool
	Err      error
}

// PacketLossPercent returns the percentage of probes that got no reply.
func (r PingResult) PacketLossPercent() float64 {
	if r.Sent == 0 {
		return 0
	}
	return 100 * float64(r.Sent-r.Received) / float64(r.Sent)
}

// Ping sends `count` ICMP echo requests to target (an IPv4 address or a
// resolvable hostname) and returns aggregate round-trip statistics.
//
// It uses an unprivileged ICMP datagram socket (SOCK_DGRAM), which on Linux
// works without root as long as net.ipv4.ping_group_range permits the
// running process's group — true out of the box on most modern
// distributions. If it isn't, Err explains how to fix it.
func Ping(target string, count int, timeout time.Duration) PingResult {
	res := PingResult{Target: target}
	if count < 1 {
		count = 1
	}

	dst, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		res.Err = fmt.Errorf("could not resolve %q: %w", target, err)
		return res
	}

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		res.Err = fmt.Errorf(
			"could not open an ICMP socket (%v).\n"+
				"On Linux, enable unprivileged ping with:\n"+
				`  sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"`+"\n"+
				"or run warp-speed with sudo / grant it CAP_NET_RAW", err)
		return res
	}
	defer conn.Close()

	var rtts []float64
	pid := os.Getpid() & 0xffff

	for i := 0; i < count; i++ {
		seq := i + 1

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   pid,
				Seq:  seq,
				Data: []byte("warp-speed-probe"),
			},
		}
		wb, err := msg.Marshal(nil)
		if err != nil {
			continue
		}

		res.Sent++
		start := time.Now()

		if _, err := conn.WriteTo(wb, &net.UDPAddr{IP: dst.IP}); err != nil {
			if i < count-1 {
				time.Sleep(200 * time.Millisecond)
			}
			continue
		}

		if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			continue
		}

		rb := make([]byte, 1500)
		n, _, err := conn.ReadFrom(rb)
		if err != nil {
			// Timed out or errored: counted as a lost packet.
			if i < count-1 {
				time.Sleep(200 * time.Millisecond)
			}
			continue
		}
		rtt := time.Since(start)

		rm, err := icmp.ParseMessage(1, rb[:n]) // protocol 1 = ICMPv4
		if err != nil {
			continue
		}

		if rm.Type == ipv4.ICMPTypeEchoReply {
			ms := float64(rtt.Microseconds()) / 1000.0
			res.Received++
			res.LastMs = ms
			res.LastOK = true
			rtts = append(rtts, ms)
		}

		if i < count-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if len(rtts) == 0 {
		if res.Err == nil {
			res.Err = fmt.Errorf("no reply from %s (%d/%d packets received)", target, res.Received, res.Sent)
		}
		res.LastOK = false
		return res
	}

	res.MinMs, res.MaxMs = rtts[0], rtts[0]
	sum := 0.0
	for _, v := range rtts {
		if v < res.MinMs {
			res.MinMs = v
		}
		if v > res.MaxMs {
			res.MaxMs = v
		}
		sum += v
	}
	res.AvgMs = sum / float64(len(rtts))

	return res
}

// QuickPing sends a single echo request and returns the round-trip time in
// milliseconds, or -1 if there was no reply within timeout.
func QuickPing(target string, timeout time.Duration) float64 {
	res := Ping(target, 1, timeout)
	if res.Received == 0 {
		return -1
	}
	return res.LastMs
}
