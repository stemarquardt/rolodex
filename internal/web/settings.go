package web

import (
	"net/http"
	"strings"

	"rolodex/internal/web/templates"
)

func (h *Handlers) SettingsPage(w http.ResponseWriter, r *http.Request) {
	options, err := h.store.ListAllOptionValues(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.SettingsPage(options).Render(r.Context(), w)
}

func (h *Handlers) SettingsOptionCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category := strings.TrimSpace(r.FormValue("category"))
	value := strings.TrimSpace(r.FormValue("value"))
	if category == "" || value == "" {
		http.Error(w, "category and value are required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateOptionValue(r.Context(), category, value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderSettingsOptionsBody(w, r)
}

func (h *Handlers) SettingsOptionDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteOptionValue(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderSettingsOptionsBody(w, r)
}

func (h *Handlers) renderSettingsOptionsBody(w http.ResponseWriter, r *http.Request) {
	options, err := h.store.ListAllOptionValues(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.SettingsOptionsBody(options).Render(r.Context(), w)
}
