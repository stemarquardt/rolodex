package web

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"rolodex/internal/model"
	"rolodex/internal/web/templates"
)

// nullableFormValue returns a valid sql.NullString for a non-empty trimmed
// form field, or an invalid (NULL) one if it's blank — used for the
// optional start_date/end_date fields.
func nullableFormValue(r *http.Request, name string) sql.NullString {
	v := strings.TrimSpace(r.FormValue(name))
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}

func eventFromForm(r *http.Request) model.Event {
	return model.Event{
		Kind:          strings.TrimSpace(r.FormValue("kind")),
		Status:        strings.TrimSpace(r.FormValue("status")),
		Title:         strings.TrimSpace(r.FormValue("title")),
		StartDate:     nullableFormValue(r, "start_date"),
		EndDate:       nullableFormValue(r, "end_date"),
		TimeframeNote: strings.TrimSpace(r.FormValue("timeframe_note")),
		Notes:         strings.TrimSpace(r.FormValue("notes")),
	}
}

func (h *Handlers) EventsList(w http.ResponseWriter, r *http.Request) {
	events, err := h.store.ListEvents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	kindOptions, statusOptions, err := h.eventOptionValues(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.EventsList(events, kindOptions, statusOptions).Render(r.Context(), w)
}

// eventOptionValues loads the managed Event kind/status <select> option
// lists.
func (h *Handlers) eventOptionValues(ctx context.Context) (kinds, statuses []model.OptionValue, err error) {
	kinds, err = h.store.ListOptionValues(ctx, model.CategoryEventKind)
	if err != nil {
		return nil, nil, err
	}
	statuses, err = h.store.ListOptionValues(ctx, model.CategoryEventStatus)
	if err != nil {
		return nil, nil, err
	}
	return kinds, statuses, nil
}

func (h *Handlers) EventCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	e := eventFromForm(r)
	if e.Kind == "" {
		http.Error(w, "kind is required", http.StatusBadRequest)
		return
	}
	if e.Status == "" {
		e.Status = "idea"
	}

	if _, err := h.store.CreateEvent(r.Context(), e); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	events, err := h.store.ListEvents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.EventGroups(events).Render(r.Context(), w)
}

func (h *Handlers) EventDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	event, err := h.store.GetEventDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if event == nil {
		http.NotFound(w, r)
		return
	}

	allPeople, err := h.store.ListPeople(r.Context(), "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	participantIDs := make(map[int64]bool, len(event.People))
	for _, p := range event.People {
		participantIDs[p.ID] = true
	}
	otherPeople := make([]model.Person, 0, len(allPeople))
	for _, p := range allPeople {
		if !participantIDs[p.ID] {
			otherPeople = append(otherPeople, p)
		}
	}

	notes, err := h.store.ListEventNotes(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.EventDetail(event, otherPeople, notes).Render(r.Context(), w)
}

func (h *Handlers) EventHeaderView(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	e, err := h.store.GetEvent(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if e == nil {
		http.NotFound(w, r)
		return
	}
	templates.EventHeader(*e).Render(r.Context(), w)
}

func (h *Handlers) EventEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	e, err := h.store.GetEvent(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if e == nil {
		http.NotFound(w, r)
		return
	}
	kindOptions, statusOptions, err := h.eventOptionValues(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.EventHeaderEdit(*e, kindOptions, statusOptions).Render(r.Context(), w)
}

func (h *Handlers) EventUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	e := eventFromForm(r)
	e.ID = id
	if e.Kind == "" {
		http.Error(w, "kind is required", http.StatusBadRequest)
		return
	}
	if e.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateEvent(r.Context(), e); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.EventHeader(e).Render(r.Context(), w)
}

func (h *Handlers) EventDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteEvent(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", "/events")
}

func (h *Handlers) EventParticipantCreate(w http.ResponseWriter, r *http.Request) {
	eventID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	personID, err := parseFormID(r, "person_id")
	if err != nil {
		http.Error(w, "invalid person_id", http.StatusBadRequest)
		return
	}

	if err := h.store.AddPersonToEvent(r.Context(), eventID, personID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	people, err := h.store.ListEventPeople(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.EventParticipantsList(eventID, people).Render(r.Context(), w)
}

func (h *Handlers) EventParticipantDelete(w http.ResponseWriter, r *http.Request) {
	eventID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	personID, ok := pathID(w, r, "personID")
	if !ok {
		return
	}

	if err := h.store.RemovePersonFromEvent(r.Context(), eventID, personID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	people, err := h.store.ListEventPeople(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.EventParticipantsList(eventID, people).Render(r.Context(), w)
}
