package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ExtraVarsConfirmedMsg is sent when the user confirms extra vars.
type ExtraVarsConfirmedMsg struct{ Raw string }

// ExtraVarsOverlay provides a text-input prompt for --extra-vars.
type ExtraVarsOverlay struct {
	input   textinput.Model
	current string // currently active value (displayed as hint)
	width   int
	height  int
}

func newExtraVarsOverlay(width, height int) *ExtraVarsOverlay {
	ti := textinput.New()
	ti.Placeholder = "key=value key2=value2 …"
	ti.Width = 52
	ti.CharLimit = 512
	ti.Focus()

	return &ExtraVarsOverlay{input: ti, width: width, height: height}
}

func (e *ExtraVarsOverlay) SetCurrent(raw string) {
	e.current = raw
	e.input.SetValue(raw)
}

func (e *ExtraVarsOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		e.input, cmd = e.input.Update(msg)
		return cmd
	}

	switch key.String() {
	case "enter":
		val := strings.TrimSpace(e.input.Value())
		return func() tea.Msg { return ExtraVarsConfirmedMsg{Raw: val} }
	default:
		var cmd tea.Cmd
		e.input, cmd = e.input.Update(msg)
		return cmd
	}
}

func (e *ExtraVarsOverlay) View() string {
	boxW := min(e.width-8, 66)

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Extra Variables") + "\n\n")

	sb.WriteString(overlayLabelStyle.Render("Format: ") +
		overlayMutedStyle.Render("key=value key2=value2") + "\n\n")

	sb.WriteString(overlayLabelStyle.Render("Value:  ") + e.input.View() + "\n")

	if e.current != "" {
		sb.WriteString("\n" + overlayMutedStyle.Render("Current: "+e.current) + "\n")
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[enter] confirm  [esc] cancel  [ctrl+u] clear"))

	return overlayBoxStyle.
		Width(boxW).
		Height(12).
		Render(sb.String())
}
