# warpping

A terminal network speed meter and ping tester for Linux, written in Go.

`warpping` shows your live download/upload throughput and a background ping
latency reading, and lets you run an on-demand ping test against **any**
domain or IP address without leaving the dashboard — just type it and press
Enter.

```
┌───────────────────────────────────────────────────┐
│ Interface     : eth0                               │
│                                                     │
│ ↓ Download    : 85.40 Mbps                         │
│ ↑ Upload      : 12.10 Mbps                          │
│ ⏱ Ping (8.8.8.8): 14 ms                             │
└───────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────┐
│ Enter domain or IP address                         │
└───────────────────────────────────────────────────┘

Type a domain or IP and press Enter to test its ping · Esc / Ctrl+C to quit
```

## Features

- **Live throughput** — reads `/proc/net/dev` on an interval and reports
  real download/upload speed in Mbps.
- **Auto-detected interface** — finds the interface tied to your default
  route, or pin one manually with `-iface`.
- **Background ping** — continuously pings a target host (default
  `8.8.8.8`) so latency is always visible.
- **On-demand ping test** — type any domain or IP into the input box to
  send a short burst of pings and see sent/received/loss and
  min/avg/max round-trip time.
- **Native ICMP** — pings are sent directly via `golang.org/x/net/icmp`
  (no shelling out to the system `ping` binary).

## Requirements

- Linux (reads `/proc/net/dev` and `/proc/net/route`)
- Go 1.22+ to build from source

### Unprivileged ping

`warpping` uses an unprivileged ICMP datagram socket, which works without
root on most modern distributions out of the box. If pings fail with a
permission error, enable it with:

```bash
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

or run `warpping` with `sudo` / grant it `CAP_NET_RAW`:

```bash
sudo setcap cap_net_raw+ep ./bin/warpping
```

## Quick install (prebuilt binary)

Once you've published a release on GitHub (see the Makefile/CI setup below),
Linux users can install the latest binary with:

```bash
curl -fsSL https://raw.githubusercontent.com/YOUR_GITHUB_USERNAME/warpping/main/install.sh | bash
```

## Build & run

```bash
make build      # -> ./bin/warpping
make run        # build and run with defaults
make install    # copy to /usr/local/bin (may need sudo)
```

Or directly with `go build`:

```bash
go build -o warpping ./cmd/warpping
./warpping
```

## Usage

```bash
./warpping [flags]

Flags:
  -iface string      network interface to monitor (default: auto-detect)
  -host string        host to continuously ping in the background (default "8.8.8.8")
  -interval duration   dashboard refresh interval, e.g. 500ms, 1s (default 1s)
  -timeout duration    timeout for a single ping probe (default 1.5s)
  -count int           number of probes sent for an on-demand ping test (default 5)
```

Example:

```bash
./warpping -iface wlan0 -host 1.1.1.1 -interval 500ms
```

While it's running, type any domain or IP address and press Enter to test
its ping. Press `Esc` or `Ctrl+C` to quit.

## Project layout

```
warpping/
├── cmd/warpping/main.go        # Application entry point
├── internal/
│   ├── config/config.go        # App configuration & flag parsing
│   ├── network/
│   │   ├── bandwidth.go        # /proc/net/dev parsing, throughput calc
│   │   ├── ping.go             # ICMP ping via golang.org/x/net
│   │   └── validator.go        # Domain / IP validation
│   └── ui/
│       ├── update.go           # Bubble Tea model & message handling
│       └── view.go             # Dashboard rendering (lipgloss)
├── go.mod / go.sum
├── Makefile
├── LICENSE
└── README.md
```

## Tech stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — the text input component
- `golang.org/x/net/icmp` — raw ICMP echo requests

## License

MIT — see [LICENSE](LICENSE).
