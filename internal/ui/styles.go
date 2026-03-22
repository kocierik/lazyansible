package ui

import "github.com/charmbracelet/lipgloss"

var (
	// ── Colors used by panel styles ───────────────────────────────────────────
	colorWhite       = lipgloss.Color("#F9FAFB")
	colorBorder      = lipgloss.Color("#334155")
	colorBorderFocus = lipgloss.Color("#818CF8")
	colorHeaderBg    = lipgloss.Color("#0F172A")

	// ── Panel border styles ───────────────────────────────────────────────────
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Background(lipgloss.Color("#0D1117")).
			Padding(0, 1)

	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorderFocus).
				Background(lipgloss.Color("#0D1117")).
				Padding(0, 1)

	// ── Header bar ────────────────────────────────────────────────────────────
	headerStyle = lipgloss.NewStyle().
			Background(colorHeaderBg).
			Foreground(colorWhite).
			Bold(true)

	// ── Shared overlay styles ─────────────────────────────────────────────────
	overlayBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#111827")).
			Padding(1, 2)

	overlayTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true)

	overlaySelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F9FAFB")).
				Background(lipgloss.Color("#374151")).
				Bold(true)

	overlayItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D1D5DB"))

	overlayMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4B5563")).
				Italic(true)

	overlayHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4B5563"))

	overlayLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF"))

	overlayActiveInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true)
)
