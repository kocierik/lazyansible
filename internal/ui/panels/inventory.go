// Package panels contains the individual TUI panel models.
package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/core"
)

// InventoryNode is a flattened row in the inventory tree.
type InventoryNode struct {
	Kind     string // "group" | "host"
	Name     string
	Indent   int
	Expanded bool
	Parent   string
}

// InventoryPanel renders the inventory tree.
type InventoryPanel struct {
	inventory *core.Inventory
	nodes     []InventoryNode
	cursor    int
	// Track which groups are collapsed.
	collapsed map[string]bool
	width     int
	height    int
	focused   bool
}

func NewInventoryPanel(inv *core.Inventory, width, height int) *InventoryPanel {
	p := &InventoryPanel{
		inventory: inv,
		collapsed: make(map[string]bool),
		width:     width,
		height:    height,
	}
	p.buildNodes()
	return p
}

func (p *InventoryPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *InventoryPanel) SetFocused(f bool) { p.focused = f }

func (p *InventoryPanel) SetInventory(inv *core.Inventory) {
	p.inventory = inv
	p.cursor = 0
	p.buildNodes()
}

func (p *InventoryPanel) SelectedHost() string {
	if p.cursor < len(p.nodes) && p.nodes[p.cursor].Kind == "host" {
		return p.nodes[p.cursor].Name
	}
	return ""
}

func (p *InventoryPanel) SelectedGroup() string {
	if p.cursor < len(p.nodes) && p.nodes[p.cursor].Kind == "group" {
		return p.nodes[p.cursor].Name
	}
	return ""
}

// buildNodes flattens the inventory tree into a list of renderable nodes.
func (p *InventoryPanel) buildNodes() {
	p.nodes = nil
	if p.inventory == nil {
		return
	}
	for _, groupName := range p.inventory.OrderedGroups {
		g, ok := p.inventory.Groups[groupName]
		if !ok {
			continue
		}
		collapsed := p.collapsed[groupName]
		hostCount := len(g.Hosts)
		label := fmt.Sprintf("%s (%d)", groupName, hostCount)

		var prefix string
		if collapsed {
			prefix = "▶ "
		} else {
			prefix = "▼ "
		}

		p.nodes = append(p.nodes, InventoryNode{
			Kind:     "group",
			Name:     groupName,
			Indent:   0,
			Expanded: !collapsed,
			Parent:   "",
		})
		_ = label
		_ = prefix

		if collapsed {
			continue
		}

		for _, hostName := range g.Hosts {
			p.nodes = append(p.nodes, InventoryNode{
				Kind:   "host",
				Name:   hostName,
				Indent: 1,
				Parent: groupName,
			})
		}
	}
}

// Update handles keyboard input for the inventory panel.
func (p *InventoryPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.cursor < len(p.nodes)-1 {
				p.cursor++
			}
		case "k", "up":
			if p.cursor > 0 {
				p.cursor--
			}
		case "enter", " ":
			if p.cursor < len(p.nodes) && p.nodes[p.cursor].Kind == "group" {
				name := p.nodes[p.cursor].Name
				p.collapsed[name] = !p.collapsed[name]
				p.buildNodes()
			}
		case "g":
			p.cursor = 0
		case "G":
			p.cursor = len(p.nodes) - 1
		}
	}
	return nil
}

// View renders the inventory panel.
func (p *InventoryPanel) View() string {
	if p.inventory == nil {
		return mutedText("No inventory loaded.\nUse -i flag or place inventory in current dir.")
	}

	var sb strings.Builder
	// title is shown in the panel border; no need to repeat it here

	// Determine visible slice.
	contentH := p.height - 4 // border + title
	if contentH < 1 {
		contentH = 1
	}
	start := 0
	if p.cursor >= contentH {
		start = p.cursor - contentH + 1
	}
	end := start + contentH
	if end > len(p.nodes) {
		end = len(p.nodes)
	}

	for i := start; i < end; i++ {
		node := p.nodes[i]
		selected := i == p.cursor

		var line string
		switch node.Kind {
		case "group":
			g := p.inventory.Groups[node.Name]
			collapsed := p.collapsed[node.Name]
			arrow := "▼"
			if collapsed {
				arrow = "▶"
			}
			count := len(g.Hosts)
			text := fmt.Sprintf("%s %s (%d)", arrow, node.Name, count)
			if selected && p.focused {
				line = selectedGroupStyle.Render(text)
			} else {
				line = groupStyle.Render(text)
			}
		case "host":
			indent := strings.Repeat("  ", node.Indent)
			text := indent + "• " + node.Name
			if selected && p.focused {
				line = selectedHostStyle.Render(text)
			} else {
				line = hostStyle.Render(text)
			}
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

var (
	groupStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	selectedGroupStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true).
				Background(lipgloss.Color("#1F2937"))

	hostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	selectedHostStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F9FAFB")).
				Bold(true).
				Background(lipgloss.Color("#1F2937"))
)

func mutedText(s string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")).
		Italic(true).
		Render(s)
}
