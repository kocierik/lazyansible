package models

import (
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
)

type SessionState uint

type MainModel struct {
	ListModel       ListModel
	FilePickerModel FilePickerModel
	PagerModel      PageModel
	State           SessionState
}

type ListModel struct {
	List     list.Model
	Choice   string
	Quitting bool
}

type FilePickerModel struct {
	Filepicker   filepicker.Model
	SelectedFile string
	Quitting     bool
	Err          error
}

type PageModel struct {
	Content  string
	Ready    bool
	Viewport viewport.Model
}
