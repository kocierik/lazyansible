package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kocierik/lazyansible/internal/runprofiles"
)

// RunProfileLoadMsg is sent when the user selects a run profile to apply.
type RunProfileLoadMsg struct{ Profile runprofiles.Profile }

type rpMode int

const (
	rpModeList rpMode = iota
	rpModeSave
)

// RunProfilesOverlay allows saving and loading named run configurations.
type RunProfilesOverlay struct {
	profiles []runprofiles.Profile
	cursor   int
	mode     rpMode
	width    int
	height   int
	err      string

	// Save form
	nameInput textinput.Model

	// Snapshot of current run state to save.
	snapPlaybook  string
	snapLimit     string
	snapTags      []string
	snapExtraVars string
	snapCheck     bool
	snapDiff      bool
	snapInventory string
}

func newRunProfilesOverlay(width, height int) *RunProfilesOverlay {
	ti := textinput.New()
	ti.Placeholder = "profile name"
	ti.CharLimit = 64
	ti.Width = 36

	o := &RunProfilesOverlay{width: width, height: height, nameInput: ti}
	o.reload()
	return o
}

func (o *RunProfilesOverlay) reload() {
	profs, err := runprofiles.Load()
	if err != nil {
		o.err = err.Error()
	} else {
		o.err = ""
	}
	o.profiles = profs
}

// SetSnapshot captures current run state so the user can save it as a profile.
func (o *RunProfilesOverlay) SetSnapshot(playbook, limit string, tags []string,
	extraVars string, check, diff bool, inventory string) {
	o.snapPlaybook = playbook
	o.snapLimit = limit
	o.snapTags = tags
	o.snapExtraVars = extraVars
	o.snapCheck = check
	o.snapDiff = diff
	o.snapInventory = inventory
}

func (o *RunProfilesOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		if o.mode == rpModeSave {
			var cmd tea.Cmd
			o.nameInput, cmd = o.nameInput.Update(msg)
			return cmd
		}
		return nil
	}

	if o.mode == rpModeSave {
		return o.updateSave(key)
	}
	return o.updateList(key)
}

func (o *RunProfilesOverlay) updateList(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "j", "down":
		if o.cursor < len(o.profiles)-1 {
			o.cursor++
		}
	case "k", "up":
		if o.cursor > 0 {
			o.cursor--
		}
	case "g":
		o.cursor = 0
	case "G":
		if len(o.profiles) > 0 {
			o.cursor = len(o.profiles) - 1
		}
	case "s":
		o.mode = rpModeSave
		o.nameInput.SetValue("")
		o.nameInput.Focus()
	case "d":
		if o.cursor < len(o.profiles) {
			updated := runprofiles.Delete(o.profiles, o.profiles[o.cursor].Name)
			_ = runprofiles.Save(updated)
			o.reload()
			if o.cursor >= len(o.profiles) && o.cursor > 0 {
				o.cursor--
			}
		}
	case "enter":
		if o.cursor < len(o.profiles) {
			p := o.profiles[o.cursor]
			return func() tea.Msg { return RunProfileLoadMsg{Profile: p} }
		}
	}
	return nil
}

func (o *RunProfilesOverlay) updateSave(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc":
		o.mode = rpModeList
	case "enter":
		name := strings.TrimSpace(o.nameInput.Value())
		if name != "" {
			p := runprofiles.Profile{
				Name:      name,
				Playbook:  o.snapPlaybook,
				Limit:     o.snapLimit,
				Tags:      o.snapTags,
				ExtraVars: o.snapExtraVars,
				CheckMode: o.snapCheck,
				DiffMode:  o.snapDiff,
				Inventory: o.snapInventory,
			}
			updated := runprofiles.Upsert(o.profiles, p)
			_ = runprofiles.Save(updated)
			o.reload()
		}
		o.mode = rpModeList
	default:
		var cmd tea.Cmd
		o.nameInput, cmd = o.nameInput.Update(key)
		return cmd
	}
	return nil
}

func (o *RunProfilesOverlay) View() string {
	boxW := min(o.width-8, 72)
	boxH := min(o.height-4, 28)

	if o.mode == rpModeSave {
		return o.viewSave(boxW, boxH)
	}
	return o.viewList(boxW, boxH)
}

func (o *RunProfilesOverlay) viewList(boxW, boxH int) string {
	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Run Profiles") + "\n\n")

	if o.err != "" {
		sb.WriteString(overlayMutedStyle.Render("Error: "+o.err) + "\n\n")
	}

	if len(o.profiles) == 0 {
		sb.WriteString(overlayMutedStyle.Render("No profiles saved yet.\nPress [s] to save current run configuration.") + "\n")
	} else {
		contentH := boxH - 9
		if contentH < 1 {
			contentH = 1
		}
		start := 0
		if o.cursor >= contentH {
			start = o.cursor - contentH + 1
		}
		end := start + contentH
		if end > len(o.profiles) {
			end = len(o.profiles)
		}

		header := fmt.Sprintf("  %-18s  %-20s  %-14s  %s", "Name", "Playbook", "Limit", "Flags")
		sb.WriteString(overlayLabelStyle.Render(header) + "\n")
		sb.WriteString(overlayMutedStyle.Render(strings.Repeat("─", min(boxW-6, 70))) + "\n")

		for i := start; i < end; i++ {
			p := o.profiles[i]
			flags := ""
			if p.CheckMode {
				flags += "✓check "
			}
			if p.DiffMode {
				flags += "diff "
			}
			if len(p.Tags) > 0 {
				flags += "#" + strings.Join(p.Tags, ",")
			}
			line := fmt.Sprintf("  %-18s  %-20s  %-14s  %s",
				truncateStr(p.Name, 18),
				truncateStr(p.Playbook, 20),
				truncateStr(p.Limit, 14),
				truncateStr(flags, 16),
			)
			if i == o.cursor {
				sb.WriteString(overlaySelectedStyle.Render(line) + "\n")
			} else {
				sb.WriteString(overlayItemStyle.Render(line) + "\n")
			}
		}

		// Detail for selected profile.
		if o.cursor < len(o.profiles) {
			p := o.profiles[o.cursor]
			if p.ExtraVars != "" {
				sb.WriteString("\n" + overlayLabelStyle.Render("Extra vars: ") +
					overlayMutedStyle.Render(truncateStr(p.ExtraVars, boxW-14)) + "\n")
			}
			if p.Inventory != "" {
				sb.WriteString(overlayLabelStyle.Render("Inventory:  ") +
					overlayMutedStyle.Render(truncateStr(p.Inventory, boxW-14)) + "\n")
			}
		}
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[s] save current  [enter] load  [d] delete  [esc] close"))
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}

func (o *RunProfilesOverlay) viewSave(boxW, boxH int) string {
	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Save Run Profile") + "\n\n")

	sb.WriteString(overlayLabelStyle.Render("Profile name: ") + o.nameInput.View() + "\n\n")

	sb.WriteString(overlayLabelStyle.Render("Will save:\n"))
	sb.WriteString(overlayMutedStyle.Render(fmt.Sprintf("  Playbook:   %s\n", truncateStr(o.snapPlaybook, 40))))
	sb.WriteString(overlayMutedStyle.Render(fmt.Sprintf("  Limit:      %s\n", truncateStr(o.snapLimit, 40))))
	if len(o.snapTags) > 0 {
		sb.WriteString(overlayMutedStyle.Render(fmt.Sprintf("  Tags:       %s\n", strings.Join(o.snapTags, ", "))))
	}
	if o.snapExtraVars != "" {
		sb.WriteString(overlayMutedStyle.Render(fmt.Sprintf("  Extra vars: %s\n", truncateStr(o.snapExtraVars, 40))))
	}
	flags := ""
	if o.snapCheck {
		flags += "--check "
	}
	if o.snapDiff {
		flags += "--diff"
	}
	if flags != "" {
		sb.WriteString(overlayMutedStyle.Render(fmt.Sprintf("  Flags:      %s\n", strings.TrimSpace(flags))))
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[enter] save  [esc] cancel"))
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}
