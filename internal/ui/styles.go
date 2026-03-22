package ui

import "github.com/charmbracelet/lipgloss"

var (
	// ── Color palette ────────────────────────────────────────────────────────
	colorPrimary     = lipgloss.Color("#7C3AED") // violet
	colorAccent      = lipgloss.Color("#06B6D4") // cyan
	colorOK          = lipgloss.Color("#22C55E") // green
	colorChanged     = lipgloss.Color("#F59E0B") // amber
	colorFailed      = lipgloss.Color("#EF4444") // red
	colorSkipped     = lipgloss.Color("#6B7280") // gray
	colorUnreachble  = lipgloss.Color("#F97316") // orange
	colorMuted       = lipgloss.Color("#4B5563") // dim gray
	colorWhite       = lipgloss.Color("#F9FAFB")
	colorBorder      = lipgloss.Color("#334155") // slate-700 — visible but not overpowering
	colorBorderFocus = lipgloss.Color("#818CF8") // indigo-400 — bright focused border
	colorHeaderBg    = lipgloss.Color("#0F172A") // near-black header bg
	colorHeaderBg2   = lipgloss.Color("#1E293B") // slightly lighter bg for sections
	colorSurface     = lipgloss.Color("#0F172A") // panel background (subtle dark blue)

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

	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			MarginBottom(1)

	// ── List item styles ──────────────────────────────────────────────────────
	itemStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// ── Status styles ─────────────────────────────────────────────────────────
	okStyle          = lipgloss.NewStyle().Foreground(colorOK).Bold(true)
	changedStyle     = lipgloss.NewStyle().Foreground(colorChanged).Bold(true)
	failedStyle      = lipgloss.NewStyle().Foreground(colorFailed).Bold(true)
	skippedStyle     = lipgloss.NewStyle().Foreground(colorSkipped)
	unreachableStyle = lipgloss.NewStyle().Foreground(colorUnreachble).Bold(true)

	// ── Help / hint bar ───────────────────────────────────────────────────────
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// ── Key hint style ────────────────────────────────────────────────────────
	keyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
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
