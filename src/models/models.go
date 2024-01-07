package models

import (
	"github.com/charmbracelet/bubbles/viewport"
)

type Model struct {
	Choices    []string         // items on the to-do list
	Cursor     int              // which to-do list item our cursor is pointing at
	Selected   map[int]struct{} // which to-do items are selected
	Content    string
	FileCursor string
	Ready      bool
	Viewport   viewport.Model
}
