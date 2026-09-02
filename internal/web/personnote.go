package web

import (
	"net/http"
	"strings"

	"rolodex/internal/web/templates"
)

func (h *Handlers) PersonNoteCreate(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
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

	if _, err := h.store.CreateNote(r.Context(), personID, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListNotes(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.NotesList(personID, items).Render(r.Context(), w)
}

// PersonCheckIn logs a "Checked in" note with no user input required — a
// low-friction way to reset the staleness clock (which is driven purely by
// Note.created_at, see internal/model/person.go's ListStalePeople) for
// contact you had but don't have anything specific to write down about.
// Deliberately its own note rather than a separate "last contacted" field —
// it shows up honestly in the person's note history like anything else, and
// needs no schema change.
func (h *Handlers) PersonCheckIn(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	if _, err := h.store.CreateNote(r.Context(), personID, "Checked in"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListNotes(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.NotesList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) PersonNoteDelete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	noteID, ok := pathID(w, r, "noteID")
	if !ok {
		return
	}

	if err := h.store.DeleteNote(r.Context(), noteID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListNotes(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.NotesList(personID, items).Render(r.Context(), w)
}
