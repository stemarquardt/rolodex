package web

import "rolodex/internal/model"

type Handlers struct {
	store *model.Store
}

func NewHandlers(store *model.Store) *Handlers {
	return &Handlers{store: store}
}
