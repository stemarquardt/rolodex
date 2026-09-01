package web

import (
	"net/http"
	"strings"

	"rolodex/internal/web/templates"
)

func (h *Handlers) FactCreate(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	label := strings.TrimSpace(r.FormValue("label"))
	value := strings.TrimSpace(r.FormValue("value"))
	if label == "" || value == "" {
		http.Error(w, "label and value are required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateFact(r.Context(), personID, label, value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListFacts(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.FactsList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) FactDelete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	factID, ok := pathID(w, r, "factID")
	if !ok {
		return
	}

	if err := h.store.DeleteFact(r.Context(), factID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListFacts(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.FactsList(personID, items).Render(r.Context(), w)
}
