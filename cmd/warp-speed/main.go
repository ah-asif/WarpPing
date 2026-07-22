// Command warp-speed is a terminal network speed meter and ping tester for
// Linux: it shows live download/upload throughput and background ping
// latency, and lets you run an on-demand ping test against any domain or
// IP address without leaving the dashboard.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"warp-speed/internal/config"
	"warp-speed/internal/network"
	"warp-speed/internal/ui"
)

func main() {
	cfg := config.FromFlags()

	iface := cfg.Interface
	if iface == "" {
		detected, err := network.DefaultInterface()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error detecting default network interface:", err)
			fmt.Fprintln(os.Stderr, "Specify one manually, e.g. -iface eth0")
			os.Exit(1)
		}
		iface = detected
	}

	model := ui.New(cfg, iface)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running warp-speed:", err)
		os.Exit(1)
	}
}
