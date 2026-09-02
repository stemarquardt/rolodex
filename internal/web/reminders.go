package web

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"rolodex/internal/model"
	"rolodex/internal/web/templates"
)

// reminderFromForm builds a Reminder from shared fields used by both the
// standalone Reminders page and the person-profile reminders section. An
// optional personID (0 = none) fixes the person link for the latter case.
func reminderFromForm(r *http.Request, fixedPersonID int64) model.Reminder {
	rem := model.Reminder{
		DueDate: strings.TrimSpace(r.FormValue("due_date")),
		Note:    strings.TrimSpace(r.FormValue("note")),
	}

	if fixedPersonID != 0 {
		rem.PersonID = sql.NullInt64{Int64: fixedPersonID, Valid: true}
	} else if v := strings.TrimSpace(r.FormValue("person_id")); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			rem.PersonID = sql.NullInt64{Int64: id, Valid: true}
		}
	}

	if v := strings.TrimSpace(r.FormValue("event_id")); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			rem.EventID = sql.NullInt64{Int64: id, Valid: true}
		}
	}

	interval := strings.TrimSpace(r.FormValue("recurrence_interval"))
	unit := strings.TrimSpace(r.FormValue("recurrence_unit"))
	if interval != "" && unit != "" {
		if n, err := strconv.ParseInt(interval, 10, 64); err == nil && n > 0 {
			rem.RecurrenceInterval = sql.NullInt64{Int64: n, Valid: true}
			rem.RecurrenceUnit = sql.NullString{String: unit, Valid: true}
		}
	}

	return rem
}

func (h *Handlers) RemindersList(w http.ResponseWriter, r *http.Request) {
	reminders, err := h.store.ListReminders(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	people, err := h.store.ListPeople(r.Context(), "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	events, err := h.store.ListEvents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.RemindersList(reminders, people, events).Render(r.Context(), w)
}

func (h *Handlers) ReminderCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rem := reminderFromForm(r, 0)
	if rem.DueDate == "" || rem.Note == "" {
		http.Error(w, "due_date and note are required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateReminder(r.Context(), rem); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reminders, err := h.store.ListReminders(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ReminderSections(reminders).Render(r.Context(), w)
}

func (h *Handlers) ReminderDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteReminder(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reminders, err := h.store.ListReminders(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ReminderSections(reminders).Render(r.Context(), w)
}

func (h *Handlers) ReminderComplete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.store.CompleteReminder(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reminders, err := h.store.ListReminders(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ReminderSections(reminders).Render(r.Context(), w)
}
