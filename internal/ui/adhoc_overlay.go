package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/core"
)

// AdHocRunMsg is sent when the user confirms an ad-hoc command.
type AdHocRunMsg struct{ Opts core.AdHocOptions }

// AdHocOverlay provides a form for running ansible ad-hoc commands.
type AdHocOverlay struct {
	moduleInput textinput.Model
	argsInput   textinput.Model
	focusIdx    int // 0 = module, 1 = args, 2 = become toggle
	become      bool
	target      string
	inventory   string
	width       int
	height      int
}

func newAdHocOverlay(width, height int) *AdHocOverlay {
	mod := textinput.New()
	mod.Placeholder = "ping"
	mod.Width = 40
	mod.CharLimit = 128
	mod.Focus()

	args := textinput.New()
	args.Placeholder = "msg='hello world'"
	args.Width = 40
	args.CharLimit = 256

	return &AdHocOverlay{
		moduleInput: mod,
		argsInput:   args,
		width:       width,
		height:      height,
	}
}

func (a *AdHocOverlay) SetTarget(target, inventory string) {
	a.target = target
	a.inventory = inventory
}

func (a *AdHocOverlay) switchFocus(dir int) {
	a.focusIdx = (a.focusIdx + dir + 3) % 3
	if a.focusIdx == 0 {
		a.moduleInput.Focus()
		a.argsInput.Blur()
	} else if a.focusIdx == 1 {
		a.argsInput.Focus()
		a.moduleInput.Blur()
	} else {
		a.moduleInput.Blur()
		a.argsInput.Blur()
	}
}

func (a *AdHocOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		if a.focusIdx == 0 {
			a.moduleInput, cmd = a.moduleInput.Update(msg)
		} else {
			a.argsInput, cmd = a.argsInput.Update(msg)
		}
		return cmd
	}

	switch key.String() {
	case "tab":
		a.switchFocus(1)
		return nil
	case "shift+tab":
		a.switchFocus(-1)
		return nil
	case " ":
		// Space on the become row toggles it.
		if a.focusIdx == 2 {
			a.become = !a.become
		}
		return nil
	case "enter":
		if a.focusIdx == 2 {
			// On the become row, Enter also toggles.
			a.become = !a.become
			return nil
		}
		mod := strings.TrimSpace(a.moduleInput.Value())
		if mod == "" {
			mod = "ping"
		}
		opts := core.AdHocOptions{
			Hosts:     a.target,
			Inventory: a.inventory,
			Module:    mod,
			Args:      strings.TrimSpace(a.argsInput.Value()),
			Become:    a.become,
		}
		return func() tea.Msg { return AdHocRunMsg{Opts: opts} }
	default:
		var cmd tea.Cmd
		if a.focusIdx == 0 {
			a.moduleInput, cmd = a.moduleInput.Update(msg)
		} else if a.focusIdx == 1 {
			a.argsInput, cmd = a.argsInput.Update(msg)
		}
		return cmd
	}
}

func (a *AdHocOverlay) View() string {
	boxW := min(a.width-8, 60)

	target := a.target
	if target == "" {
		target = "all"
	}

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Ad-hoc Command") + "\n\n")

	// Target display.
	sb.WriteString(overlayLabelStyle.Render("Target:  ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true).Render(target) + "\n\n")

	// Module input.
	modLabel := overlayLabelStyle.Render("Module:  ")
	if a.focusIdx == 0 {
		modLabel = overlayActiveInputStyle.Render("Module:  ")
	}
	sb.WriteString(modLabel + a.moduleInput.View() + "\n\n")

	// Args input.
	argsLabel := overlayLabelStyle.Render("Args:    ")
	if a.focusIdx == 1 {
		argsLabel = overlayActiveInputStyle.Render("Args:    ")
	}
	sb.WriteString(argsLabel + a.argsInput.View() + "\n\n")

	// Example hint based on current module.
	mod := strings.TrimSpace(a.moduleInput.Value())
	if ex := moduleExample(mod); ex != "" {
		sb.WriteString(overlayMutedStyle.Render("  e.g. "+ex) + "\n\n")
	}

	// Become toggle row.
	becomeLabel := overlayLabelStyle.Render("Become:  ")
	if a.focusIdx == 2 {
		becomeLabel = overlayActiveInputStyle.Render("Become:  ")
	}
	becomeVal := "[ ] --become"
	if a.become {
		becomeVal = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true).Render("[✓] --become")
	}
	sb.WriteString(becomeLabel + becomeVal + "\n\n")

	sb.WriteString(overlayHintStyle.Render("[tab] switch  [space/enter] toggle become  [enter on run fields] run  [esc] cancel"))

	return overlayBoxStyle.
		Width(boxW).
		Height(16).
		Render(sb.String())
}

// moduleExample returns a usage hint for common modules.
func moduleExample(mod string) string {
	examples := map[string]string{
		"ping":    "(no args needed)",
		"command": "cmd='uptime'",
		"shell":   "cmd='ps aux | grep nginx'",
		"copy":    "src=/tmp/file.txt dest=/tmp/file.txt",
		"file":    "path=/tmp/test state=touch",
		"service": "name=nginx state=restarted",
		"yum":     "name=htop state=present",
		"apt":     "name=htop state=present",
		"setup":   "(no args needed – gathers facts)",
		"debug":   "msg='hello'",
		"uri":     "url=http://localhost status_code=200",
	}
	return examples[mod]
}
