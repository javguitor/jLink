package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Jabra-inspired color palette
var (
	colorPrimary = lipgloss.Color("#FFD001")
	colorAccent  = lipgloss.Color("#FFD700")
	colorDanger  = lipgloss.Color("#FF4444")
	colorWarning = lipgloss.Color("#FFAA00")
	colorSuccess = lipgloss.Color("#00CC66")
	colorMuted   = lipgloss.Color("#666666")
	colorWhite   = lipgloss.Color("#FFFFFF")
	colorBlack   = lipgloss.Color("#000000")
)

// Reusable styles
var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorBlack).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	buttonStyle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorBlack).
			Padding(0, 1)

	buttonActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#4488FF")).
				Foreground(colorWhite).
				Padding(0, 1)

	accentTextStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlack).
			Background(colorPrimary).
			Padding(0, 3)
)

var primaryStyle = lipgloss.NewStyle().Foreground(colorPrimary)

func renderBatteryBar(percent uint8, charging, batteryLow bool) string {
	const barWidth = 10
	filled := int(percent) * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	var color lipgloss.Style
	switch {
	case batteryLow:
		color = dangerStyle
	case percent <= 30:
		color = warningStyle
	default:
		color = primaryStyle
	}

	bar := color.Render(strings.Repeat("◼", filled)) + mutedStyle.Render(strings.Repeat("◻", empty))

	chargingIcon := ""
	if charging {
		chargingIcon = "⚡"
	}

	return fmt.Sprintf("[%s] %d%%%s", bar, percent, chargingIcon)
}

func renderProgressBar(current, max, width int) string {
	if max <= 0 || width <= 0 {
		return ""
	}
	filled := current * width / max
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled
	return fmt.Sprintf("[%s%s]", strings.Repeat("█", filled), strings.Repeat("░", empty))
}

func renderHeader(m model) string {
	var parts []string

	if m.dongle != nil {
		parts = append(parts, headerStyle.Render(m.dongle.deviceName))
	} else {
		parts = append(parts, headerStyle.Render("Looking for dongle..."))
	}

	if m.headset != nil && m.headset.batteryStatus != nil {
		battery := renderBatteryBar(
			m.headset.batteryStatus.levelInPercent,
			m.headset.batteryStatus.charging,
			m.headset.batteryStatus.batteryLow,
		)

		btIndicator := ""
		if linkQualitySet {
			switch linkQualityStatus {
			case 2:
				btIndicator = successStyle.Render(" [BT:High]")
			case 1:
				btIndicator = warningStyle.Render(" [BT:Low]")
			default:
				btIndicator = dangerStyle.Render(" [BT:Off]")
			}
		}

		headIndicator := ""
		if m.headset.featureFlags != nil && m.headset.featureFlags.onHeadDetection && headDetectionSet {
			switch {
			case headDetectionLeft && headDetectionRight:
				headIndicator = successStyle.Render(" [Wearing]")
			case headDetectionLeft || headDetectionRight:
				l, r := "off", "off"
				if headDetectionLeft {
					l = "on"
				}
				if headDetectionRight {
					r = "on"
				}
				headIndicator = warningStyle.Render(fmt.Sprintf(" [L:%s R:%s]", l, r))
			default:
				headIndicator = mutedStyle.Render(" [Off-head]")
			}
		}

		headsetInfo := fmt.Sprintf("%s  %s%s%s",
			m.headset.deviceName, battery, btIndicator, headIndicator)
		parts = append(parts, headsetInfo)
	} else if m.headset != nil {
		parts = append(parts, m.headset.deviceName)
	}

	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + "    " + parts[1]
}

func renderFooter(hints []string) string {
	var rendered []string
	for _, h := range hints {
		rendered = append(rendered, buttonStyle.Render(h))
	}
	return strings.Join(rendered, " ")
}

func renderLogo() string {
	logoLines := []string{
		"    ██ ██      ██ ██   ██ ██  ██",
		"    ██ ██      ██ ███  ██ ██ ██ ",
		"    ██ ██      ██ ████ ██ ████  ",
		"██  ██ ██      ██ ██ ████ ██ ██ ",
		" ████  ██████  ██ ██  ███ ██  ██",
	}

	// Warm gradient (Jabra yellow/gold palette)
	gradientColors := []lipgloss.Color{
		lipgloss.Color("#FFE44D"),
		lipgloss.Color("#FFD001"),
		lipgloss.Color("#FFAA00"),
		lipgloss.Color("#FFD001"),
		lipgloss.Color("#FFE44D"),
	}

	var b strings.Builder
	for i, line := range logoLines {
		color := gradientColors[i%len(gradientColors)]
		b.WriteString(lipgloss.NewStyle().Foreground(color).Bold(true).Render(line))
		if i < len(logoLines)-1 {
			b.WriteString("\n")
		}
	}

	// Subtitle
	subtitle := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render("Jabra Direct for Linux")
	credit := mutedStyle.Render("by javguitor")

	// Frame with double border like engram
	logoFrame := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1).
		Render(b.String())

	return logoFrame + "\n" + subtitle + "  " + credit
}
