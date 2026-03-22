package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/ssh"
)

// SSHProfileAppliedMsg is sent when the user applies a profile.
type SSHProfileAppliedMsg struct{ ExtraVars string }

type sshFormMode int

const (
	sshFormList sshFormMode = iota
	sshFormAdd
)

// SSHProfileOverlay manages SSH profiles (list / add / delete / apply).
type SSHProfileOverlay struct {
	profiles []*ssh.Profile
	cursor   int
	mode     sshFormMode
	width    int
	height   int

	// Form fields for adding a new profile.
	formFields []*textinput.Model
	formIdx    int
	formLabels []string
}

func newSSHProfileOverlay(width, height int) *SSHProfileOverlay {
	o := &SSHProfileOverlay{width: width, height: height}
	o.loadProfiles()
	o.buildForm()
	return o
}

func (o *SSHProfileOverlay) loadProfiles() {
	profiles, _ := ssh.Load()
	o.profiles = profiles
}

func (o *SSHProfileOverlay) buildForm() {
	labels := []string{"Name", "User", "Key file", "Port", "Bastion host", "Extra SSH args"}
	placeholders := []string{"prod-ops", "ubuntu", "~/.ssh/id_rsa", "22", "", "-o StrictHostKeyChecking=no"}
	o.formLabels = labels

	var fields []*textinput.Model
	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.Width = 36
		ti.CharLimit = 128
		_ = label
		fields = append(fields, &ti)
	}
	fields[0].Focus()
	o.formFields = fields
	o.formIdx = 0
}

func (o *SSHProfileOverlay) focusField(idx int) {
	for i, f := range o.formFields {
		if i == idx {
			f.Focus()
		} else {
			f.Blur()
		}
	}
	o.formIdx = idx
}

func (o *SSHProfileOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		if o.mode == sshFormAdd {
			var cmd tea.Cmd
			*o.formFields[o.formIdx], cmd = o.formFields[o.formIdx].Update(msg)
			return cmd
		}
		return nil
	}

	if o.mode == sshFormAdd {
		return o.updateForm(key)
	}
	return o.updateList(key)
}

func (o *SSHProfileOverlay) updateList(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "j", "down":
		if o.cursor < len(o.profiles)-1 {
			o.cursor++
		}
	case "k", "up":
		if o.cursor > 0 {
			o.cursor--
		}
	case "a":
		o.mode = sshFormAdd
		o.buildForm()
	case "d":
		if o.cursor < len(o.profiles) {
			o.profiles = append(o.profiles[:o.cursor], o.profiles[o.cursor+1:]...)
			if o.cursor > 0 {
				o.cursor--
			}
			_ = ssh.Save(o.profiles)
		}
	case "enter":
		if o.cursor < len(o.profiles) {
			extra := o.profiles[o.cursor].ToExtraVarsRaw()
			return func() tea.Msg { return SSHProfileAppliedMsg{ExtraVars: extra} }
		}
	}
	return nil
}

func (o *SSHProfileOverlay) updateForm(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc":
		o.mode = sshFormList
		return nil
	case "tab", "down":
		o.focusField((o.formIdx + 1) % len(o.formFields))
	case "shift+tab", "up":
		o.focusField((o.formIdx - 1 + len(o.formFields)) % len(o.formFields))
	case "enter":
		if o.formIdx < len(o.formFields)-1 {
			o.focusField(o.formIdx + 1)
			return nil
		}
		// Save the new profile.
		p := o.buildProfile()
		if p.Name != "" {
			o.profiles = append(o.profiles, p)
			_ = ssh.Save(o.profiles)
		}
		o.mode = sshFormList
		o.cursor = len(o.profiles) - 1
		return nil
	default:
		var cmd tea.Cmd
		*o.formFields[o.formIdx], cmd = o.formFields[o.formIdx].Update(key)
		return cmd
	}
	return nil
}

func (o *SSHProfileOverlay) buildProfile() *ssh.Profile {
	p := &ssh.Profile{
		Name:    strings.TrimSpace(o.formFields[0].Value()),
		User:    strings.TrimSpace(o.formFields[1].Value()),
		KeyFile: strings.TrimSpace(o.formFields[2].Value()),
	}
	if port, err := strconv.Atoi(strings.TrimSpace(o.formFields[3].Value())); err == nil {
		p.Port = port
	}
	p.BastionHost = strings.TrimSpace(o.formFields[4].Value())
	p.ExtraArgs = strings.TrimSpace(o.formFields[5].Value())
	return p
}

func (o *SSHProfileOverlay) View() string {
	boxW := min(o.width-8, 68)
	boxH := min(o.height-4, 26)

	if o.mode == sshFormAdd {
		return o.viewForm(boxW, boxH)
	}
	return o.viewList(boxW, boxH)
}

func (o *SSHProfileOverlay) viewList(boxW, boxH int) string {
	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("SSH Profiles") + "\n\n")

	if len(o.profiles) == 0 {
		sb.WriteString(overlayMutedStyle.Render("No profiles yet.  Press [a] to add one.") + "\n")
	} else {
		contentH := boxH - 8
		start := 0
		if o.cursor >= contentH {
			start = o.cursor - contentH + 1
		}
		end := start + contentH
		if end > len(o.profiles) {
			end = len(o.profiles)
		}

		for i := start; i < end; i++ {
			p := o.profiles[i]
			line := fmt.Sprintf("%-16s  %s", truncateStr(p.Name, 16), p.Summary())
			if i == o.cursor {
				sb.WriteString(overlaySelectedStyle.Render(line) + "\n")
			} else {
				sb.WriteString(overlayItemStyle.Render(line) + "\n")
			}
		}
	}

	// Preview of apply vars.
	if o.cursor < len(o.profiles) {
		ev := o.profiles[o.cursor].ToExtraVarsRaw()
		if ev != "" {
			sb.WriteString("\n" + overlayLabelStyle.Render("Applies: ") +
				overlayMutedStyle.Render(truncateStr(ev, boxW-12)) + "\n")
		}
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[a]dd  [d]elete  [enter] apply to run  [esc] close"))
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}

func (o *SSHProfileOverlay) viewForm(boxW, boxH int) string {
	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Add SSH Profile") + "\n\n")

	for i, label := range o.formLabels {
		l := overlayLabelStyle.Render(fmt.Sprintf("%-14s", label+":"))
		if i == o.formIdx {
			l = overlayActiveInputStyle.Render(fmt.Sprintf("%-14s", label+":"))
		}
		sb.WriteString(l + o.formFields[i].View() + "\n")
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[tab] next field  [enter] on last = save  [esc] cancel"))
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}

var sshAppliedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#22C55E")).
	Bold(true)
