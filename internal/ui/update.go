// Package ui implements warp-speed's terminal dashboard using Bubble Tea.
package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"warp-speed/internal/config"
	"warp-speed/internal/network"
)

// Model is warp-speed's Bubble Tea application state.
type Model struct {
	cfg   config.Config
	iface string

	prevSample network.Sample
	haveSample bool
	downMbps   float64
	upMbps     float64

	bgPingMs float64
	bgPingOK bool

	input textinput.Model

	testing    bool
	testTarget string
	testResult *network.PingResult
	inputErr   string

	fatalErr error

	width  int
	height int
}

// --- Messages ---

type tickMsg time.Time

type statsMsg struct {
	sample network.Sample
	rates  network.Rates
}

type bgPingMsg struct {
	ms float64
	ok bool
}

type pingDoneMsg network.PingResult

type fatalErrMsg struct{ err error }

// New builds the initial Model for the given configuration.
func New(cfg config.Config, iface string) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter domain or IP address"
	ti.Focus()
	ti.CharLimit = 253
	ti.Width = 40

	return Model{
		cfg:      cfg,
		iface:    iface,
		bgPingOK: false,
		input:    ti,
	}
}

// Init kicks off the periodic refresh loop.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(m.cfg.RefreshInterval),
		refreshStatsCmd(m.iface, network.Sample{}, false),
		bgPingCmd(m.cfg.PingHost, m.cfg.PingTimeout),
		textinput.Blink,
	)
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshStatsCmd(iface string, prev network.Sample, havePrev bool) tea.Cmd {
	return func() tea.Msg {
		cur, err := network.ReadStats(iface)
		if err != nil {
			return fatalErrMsg{err: err}
		}
		var rates network.Rates
		if havePrev {
			rates = network.ComputeRates(prev, cur)
		}
		return statsMsg{sample: cur, rates: rates}
	}
}

func bgPingCmd(host string, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		ms := network.QuickPing(host, timeout)
		return bgPingMsg{ms: ms, ok: ms >= 0}
	}
}

func runPingCmd(target string, count int, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		res := network.Ping(target, count, timeout)
		return pingDoneMsg(res)
	}
}

// Update handles all incoming messages and advances the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			target := m.input.Value()
			if target == "q" || target == "quit" {
				return m, tea.Quit
			}
			if !network.IsValidHost(target) {
				m.inputErr = "That doesn't look like a valid domain or IP address."
				return m, nil
			}
			m.inputErr = ""
			m.testing = true
			m.testTarget = target
			m.testResult = nil
			m.input.SetValue("")
			return m, runPingCmd(target, m.cfg.TestCount, m.cfg.PingTimeout)
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case tickMsg:
		return m, tea.Batch(
			tickCmd(m.cfg.RefreshInterval),
			refreshStatsCmd(m.iface, m.prevSample, m.haveSample),
			bgPingCmd(m.cfg.PingHost, m.cfg.PingTimeout),
		)

	case statsMsg:
		m.downMbps = network.Mbps(msg.rates.DownBytesPerSec)
		m.upMbps = network.Mbps(msg.rates.UpBytesPerSec)
		m.prevSample = msg.sample
		m.haveSample = true
		return m, nil

	case bgPingMsg:
		m.bgPingMs = msg.ms
		m.bgPingOK = msg.ok
		return m, nil

	case pingDoneMsg:
		res := network.PingResult(msg)
		m.testResult = &res
		m.testing = false
		return m, nil

	case fatalErrMsg:
		m.fatalErr = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
