package handlers

import (
	"github.com/kocierik/lazyansible/src/models"
)

type MainModel struct {
	// MainModel models.MainModel
	ListView  ListModel
	PagerView Model
	State     models.SessionState
}

// func New(model models.MainModel, list models.ListModel, pager models.PagerModel, state SessionState) handler {
// 	return handler{model, list, pager, 0}
// }
