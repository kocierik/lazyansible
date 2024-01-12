package models

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
)

type SessionState uint

type MainModel struct {
	ListModel  ListModel
	PagerModel PageModel
	State      SessionState
}

type ListModel struct {
	List     list.Model
	Choice   string
	Quitting bool
}

type PageModel struct {
	Content  string
	Ready    bool
	Viewport viewport.Model
}
