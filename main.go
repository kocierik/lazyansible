package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"os"

	// "github.com/charmbracelet/lipgloss"

	// "github.com/kocierik/lazyansible/src/handlers"
	"github.com/kocierik/lazyansible/src/handlers"
	"github.com/kocierik/lazyansible/src/models"
	// "github.com/kocierik/lazyansible/src/utils"
	// "github.com/kocierik/lazyansible/src/utils"
)

func main() {
	h := handlers.New(&models.Model{Content: "ciao"})
	m := h.InitialModel()
	model := handlers.New(&m)
	p := tea.NewProgram(model, tea.WithAltScreen(),
		tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting program: %v", err)
		os.Exit(1)
	}
}
