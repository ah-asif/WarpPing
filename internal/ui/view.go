package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorAccent = lipgloss.Color("39")  // blue
	colorGood   = lipgloss.Color("42")  // green
	colorBad    = lipgloss.Color("203") // red
	colorMuted  = lipgloss.Color("245") // gray
	colorText   = lipgloss.Color("255")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)

	goodStyle = lipgloss.NewStyle().Bold(true).Foreground(colorGood)
	badStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorBad)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	errStyle = lipgloss.NewStyle().Foreground(colorBad)
)

// View renders the full dashboard.
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("warp-speed — Linux network speed & ping monitor"))
	b.WriteString("\n\n")

	if m.fatalErr != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.fatalErr)))
		b.WriteString("\n\n")
	}

	b.WriteString(m.renderStatsPanel())
	b.WriteString("\n\n")
	b.WriteString(m.renderInputBox())
	b.WriteString("\n")

	if m.inputErr != "" {
		b.WriteString(errStyle.Render(m.inputErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.renderTestResult())

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Type a domain or IP and press Enter to test its ping · Esc / Ctrl+C to quit"))

	return b.String()
}

func (m Model) renderStatsPanel() string {
	down := fmt.Sprintf("%.2f Mbps", m.downMbps)
	up := fmt.Sprintf("%.2f Mbps", m.upMbps)

	var ping string
	if m.bgPingOK {
		ping = valueStyle.Render(fmt.Sprintf("%.0f ms", m.bgPingMs))
	} else {
		ping = badStyle.Render("timeout")
	}

	rows := []string{
		fmt.Sprintf("%s  %s", labelStyle.Render("Interface     :"), valueStyle.Render(m.iface)),
		"",
		fmt.Sprintf("%s  %s", labelStyle.Render("↓ Download    :"), valueStyle.Render(down)),
		fmt.Sprintf("%s  %s", labelStyle.Render("↑ Upload      :"), valueStyle.Render(up)),
		fmt.Sprintf("%s  %s", labelStyle.Render(fmt.Sprintf("⏱ Ping (%s):", m.cfg.PingHost)), ping),
	}

	return panelStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) renderInputBox() string {
	return inputBoxStyle.Render(m.input.View())
}

func (m Model) renderTestResult() string {
	if m.testing {
		return labelStyle.Render(fmt.Sprintf("Pinging %s ...", m.testTarget))
	}
	if m.testResult == nil {
		return ""
	}

	r := m.testResult
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", valueStyle.Render(fmt.Sprintf("Ping results for %s", r.Target)))

	if r.Received == 0 {
		fmt.Fprintf(&b, "  %s\n", badStyle.Render(fmt.Sprintf(
			"Unreachable — 0/%d packets received", r.Sent)))
		if r.Err != nil {
			fmt.Fprintf(&b, "  %s\n", errStyle.Render(r.Err.Error()))
		}
		return b.String()
	}

	lossStyle := goodStyle
	if r.PacketLossPercent() > 0 {
		lossStyle = badStyle
	}

	fmt.Fprintf(&b, "  Sent: %d  Received: %d  Loss: %s\n",
		r.Sent, r.Received, lossStyle.Render(fmt.Sprintf("%.0f%%", r.PacketLossPercent())))
	fmt.Fprintf(&b, "  min/avg/max: %.1f / %.1f / %.1f ms\n", r.MinMs, r.AvgMs, r.MaxMs)

	return b.String()
}
