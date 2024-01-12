package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kocierik/lazyansible/src/handlers"
	"github.com/kocierik/lazyansible/src/utils"
)

func main() {
	listModel := handlers.ListModel{}
	pagerModel := handlers.PagerModel{}
	listModel = listModel.InitializeListModel()
	pagerModel = pagerModel.InitializePagerModel()

	mainModel := handlers.MainModel{
		ListView:  listModel,
		PagerView: pagerModel,
		State:     utils.ListModelState,
	}
	if _, err := tea.NewProgram(
		mainModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
