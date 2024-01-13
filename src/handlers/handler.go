package handlers

import (
	"github.com/kocierik/lazyansible/src/models"
)

type MainModel struct {
	ListView       ListModel
	FilePickerView FilePickerModel
	PagerView      PagerModel
	ListHostView   ListHostModel
	State          models.SessionState
	Height         int
	Width          int
}
