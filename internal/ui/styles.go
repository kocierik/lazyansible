package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette.
	colorPrimary    = lipgloss.Color("#7C3AED") // violet
	colorAccent     = lipgloss.Color("#06B6D4") // cyan
	colorOK         = lipgloss.Color("#22C55E") // green
	colorChanged    = lipgloss.Color("#F59E0B") // amber
	colorFailed     = lipgloss.Color("#EF4444") // red
	colorSkipped    = lipgloss.Color("#6B7280") // gray
	colorUnreachble = lipgloss.Color("#F97316") // orange
	colorMuted      = lipgloss.Color("#4B5563") // dim gray
	colorWhite      = lipgloss.Color("#F9FAFB")
	colorBorder     = lipgloss.Color("#374151")
	colorBorderFocus = lipgloss.Color("#7C3AED")

	// Panel border styles.
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorderFocus).
				Padding(0, 1)

	// Header bar.
	headerStyle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			MarginBottom(1)

	// List item styles.
	itemStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Status styles.
	okStyle          = lipgloss.NewStyle().Foreground(colorOK).Bold(true)
	changedStyle     = lipgloss.NewStyle().Foreground(colorChanged).Bold(true)
	failedStyle      = lipgloss.NewStyle().Foreground(colorFailed).Bold(true)
	skippedStyle     = lipgloss.NewStyle().Foreground(colorSkipped)
	unreachableStyle = lipgloss.NewStyle().Foreground(colorUnreachble).Bold(true)

	// Help bar.
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// Key hint style.
	keyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)
)
