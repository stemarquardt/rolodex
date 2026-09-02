package web

import (
	"net/http"

	"rolodex/internal/web/templates"
)

func (h *Handlers) PersonReminderCreate(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rem := reminderFromForm(r, personID)
	if rem.DueDate == "" || rem.Note == "" {
		http.Error(w, "due_date and note are required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateReminder(r.Context(), rem); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListRemindersForPerson(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.PersonRemindersList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) PersonReminderDelete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	remID, ok := pathID(w, r, "remID")
	if !ok {
		return
	}

	if err := h.store.DeleteReminder(r.Context(), remID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListRemindersForPerson(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.PersonRemindersList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) PersonReminderComplete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	remID, ok := pathID(w, r, "remID")
	if !ok {
		return
	}

	if err := h.store.CompleteReminder(r.Context(), remID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListRemindersForPerson(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.PersonRemindersList(personID, items).Render(r.Context(), w)
}
