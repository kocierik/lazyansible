package models

import (
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
)

type SessionState uint

type MainModel struct {
	ListModel       ListModel
	FilePickerModel ModelFilePicker
	PagerModel      ModelPager
	State           SessionState
}

type ListModel struct {
	List     list.Model
	Choice   string
	Quitting bool
}

type ModelFilePicker struct {
	Filepicker   filepicker.Model
	SelectedFile string
	Quitting     bool
	Err          error
}

type ModelPager struct {
	Content  string
	Ready    bool
	Viewport viewport.Model
}
