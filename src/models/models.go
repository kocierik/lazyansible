package models

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
)

type SessionState uint

type MainModel struct {
	State      SessionState
	ListModel  ListModel
	PagerModel PagerModel
}

type ListModel struct {
	List     list.Model
	Choice   string
	Quitting bool
}

type PagerModel struct {
	Content  string
	Ready    bool
	Viewport viewport.Model
}

// type PagerModel struct {
// 	Content  string
// 	Ready    bool
// 	Viewport viewport.Model
// }
