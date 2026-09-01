package web

import (
	"net/http"
	"strconv"

	"rolodex/internal/model"
)

type Handlers struct {
	store *model.Store
}

func NewHandlers(store *model.Store) *Handlers {
	return &Handlers{store: store}
}

// pathID parses a path parameter as an int64, writing a 400 response and
// returning ok=false if it's missing or malformed.
func pathID(w http.ResponseWriter, r *http.Request, name string) (id int64, ok bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		http.Error(w, "invalid "+name, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// parseFormID parses a required int64 form field (e.g. a foreign key chosen
// from a <select>).
func parseFormID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.FormValue(name), 10, 64)
}
