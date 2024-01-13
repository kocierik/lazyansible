package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kocierik/lazyansible/src/handlers"
	"github.com/kocierik/lazyansible/src/utils"
)

func main() {

	pagerModel := handlers.PagerModel{}
	filePickerModel := handlers.FilePickerModel{}
	listHostModel := handlers.ListHostModel{}

	filePickerModel = filePickerModel.InitializeFilePicker()
	pagerModel = pagerModel.InitializePagerModel()
	listHostModel = listHostModel.InitialModel()

	mainModel := handlers.MainModel{
		FilePickerView: filePickerModel,
		PagerView:      pagerModel,
		ListHostView:   listHostModel,
		State:          utils.FilePickerState,
	}

	if _, err := tea.NewProgram(
		mainModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), tea.WithOutput(os.Stderr)).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
