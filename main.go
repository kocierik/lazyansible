package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kocierik/lazyansible/src/handlers"
)

func main() {

	// h := handlers.New(models.MainModel{}, models.ListModel{}, models.PagerModel{}, 0)
	listModel := handlers.ListModel{}
	pagerModel := handlers.PagerModel{Content: ""}
	listModel = listModel.InitializeListModel()
	pagerModel = pagerModel.InitializePagerModel()

	mainModel := handlers.MainModel{
		ListView:  listModel,
		PagerView: pagerModel,
		State:     0,
	}

	if _, err := tea.NewProgram(mainModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
