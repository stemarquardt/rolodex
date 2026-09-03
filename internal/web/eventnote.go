package web

import (
	"net/http"
	"strings"

	"rolodex/internal/web/templates"
)

func (h *Handlers) EventNoteCreate(w http.ResponseWriter, r *http.Request) {
	eventID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateEventNote(r.Context(), eventID, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListEventNotes(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.EventNotesList(eventID, items).Render(r.Context(), w)
}

func (h *Handlers) EventNoteDelete(w http.ResponseWriter, r *http.Request) {
	eventID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	noteID, ok := pathID(w, r, "noteID")
	if !ok {
		return
	}

	if err := h.store.DeleteEventNote(r.Context(), noteID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListEventNotes(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.EventNotesList(eventID, items).Render(r.Context(), w)
}
