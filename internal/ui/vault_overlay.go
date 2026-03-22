package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/vault"
)

// VaultPasswordMsg is sent when the user confirms the vault password.
type VaultPasswordMsg struct{ Password string }

// VaultOverlay prompts for an Ansible Vault password.
type VaultOverlay struct {
	input          textinput.Model
	encryptedFiles []string
	width          int
	height         int
}

func newVaultOverlay(width, height int) *VaultOverlay {
	ti := textinput.New()
	ti.Placeholder = "vault password"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Width = 40
	ti.CharLimit = 256
	ti.Focus()

	return &VaultOverlay{input: ti, width: width, height: height}
}

// Scan detects vault-encrypted files in workDir and returns true if any exist.
func (v *VaultOverlay) Scan(workDir string) bool {
	v.encryptedFiles = vault.FindEncryptedFiles(workDir)
	return len(v.encryptedFiles) > 0
}

// HasEncryptedFiles returns true if the last Scan found vault files.
func (v *VaultOverlay) HasEncryptedFiles() bool {
	return len(v.encryptedFiles) > 0
}

func (v *VaultOverlay) Reset() {
	v.input.SetValue("")
	v.input.Focus()
}

func (v *VaultOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return cmd
	}
	switch key.String() {
	case "enter":
		pw := strings.TrimSpace(v.input.Value())
		return func() tea.Msg { return VaultPasswordMsg{Password: pw} }
	default:
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return cmd
	}
}

func (v *VaultOverlay) View() string {
	boxW := min(v.width-8, 60)

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Ansible Vault") + "\n\n")

	// Show detected vault files.
	if len(v.encryptedFiles) > 0 {
		sb.WriteString(overlayLabelStyle.Render("Encrypted files detected:") + "\n")
		maxShow := 4
		for i, f := range v.encryptedFiles {
			if i >= maxShow {
				sb.WriteString(overlayMutedStyle.Render(
					"  … and more\n",
				))
				break
			}
			sb.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B")).
				Render("  ⚠ "+f) + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(overlayLabelStyle.Render("Password: ") + v.input.View() + "\n")
	sb.WriteString("\n" + overlayMutedStyle.Render("Password is written to a temp file and deleted after the run.") + "\n")
	sb.WriteString("\n" + overlayHintStyle.Render("[enter] confirm  [esc] skip / cancel"))

	return overlayBoxStyle.
		Width(boxW).
		Height(14).
		Render(sb.String())
}
