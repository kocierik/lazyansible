package handlers

// tea "github.com/charmbracelet/bubbletea"

type SessionState uint

type MainModel struct {
	// MainModel models.MainModel
	ListView  ListModel
	PagerView PagerModel
	State     SessionState
}

// func New(model models.MainModel, list models.ListModel, pager models.PagerModel, state SessionState) handler {
// 	return handler{model, list, pager, 0}
// }
