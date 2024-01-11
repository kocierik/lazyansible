package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kocierik/lazyansible/src/handlers"
)

func main() {
	listModel := handlers.ListModel{}
	pagerModel := handlers.Model{}
	listModel = listModel.InitializeListModel()
	pagerModel = pagerModel.InitializePagerModel()

	mainModel := handlers.MainModel{
		ListView:  listModel,
		PagerView: pagerModel,
		State:     0,
	}

	if _, err := tea.NewProgram(
		mainModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
