package web

import (
	"net/http"

	"rolodex/internal/web/templates"
)

func (h *Handlers) Notes(w http.ResponseWriter, r *http.Request) {
	templates.ComingSoon("notes", "Notes").Render(r.Context(), w)
}
