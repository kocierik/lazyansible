package handlers

import (
	// tea "github.com/charmbracelet/bubbletea"
	"github.com/kocierik/lazyansible/src/models"
)

type handler struct {
	Model *models.Model
}

func New(model *models.Model) handler {
	return handler{model}
}
