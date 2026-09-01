package web

import (
	"net/http"

	"rolodex/internal/web/templates"
)

func (h *Handlers) Today(w http.ResponseWriter, r *http.Request) {
	templates.ComingSoon("today", "Today").Render(r.Context(), w)
}

func (h *Handlers) Circles(w http.ResponseWriter, r *http.Request) {
	templates.ComingSoon("circles", "Circles").Render(r.Context(), w)
}

func (h *Handlers) Events(w http.ResponseWriter, r *http.Request) {
	templates.ComingSoon("events", "Visits & events").Render(r.Context(), w)
}

func (h *Handlers) Reminders(w http.ResponseWriter, r *http.Request) {
	templates.ComingSoon("reminders", "Reminders").Render(r.Context(), w)
}

func (h *Handlers) Notes(w http.ResponseWriter, r *http.Request) {
	templates.ComingSoon("notes", "Notes").Render(r.Context(), w)
}
