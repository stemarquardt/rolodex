package web

import (
	"net/http"
	"strings"

	"rolodex/internal/model"
	"rolodex/internal/web/templates"
)

func (h *Handlers) CirclesList(w http.ResponseWriter, r *http.Request) {
	circles, err := h.store.ListCircles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.CircleIndex(circles).Render(r.Context(), w)
}

func (h *Handlers) CircleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateCircle(r.Context(), name, description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	circles, err := h.store.ListCircles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.CircleIndexTable(circles).Render(r.Context(), w)
}

func (h *Handlers) CircleDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	detail, err := h.store.GetCircleDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if detail == nil {
		http.NotFound(w, r)
		return
	}

	allPeople, err := h.store.ListPeople(r.Context(), "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	memberIDs := make(map[int64]bool, len(detail.Members))
	for _, m := range detail.Members {
		memberIDs[m.PersonID] = true
	}
	otherPeople := make([]model.Person, 0, len(allPeople))
	for _, p := range allPeople {
		if !memberIDs[p.ID] {
			otherPeople = append(otherPeople, p)
		}
	}

	templates.CircleDetail(detail, otherPeople).Render(r.Context(), w)
}

func (h *Handlers) CircleHeaderView(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	c, err := h.store.GetCircle(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if c == nil {
		http.NotFound(w, r)
		return
	}
	templates.CircleHeader(*c).Render(r.Context(), w)
}

func (h *Handlers) CircleEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	c, err := h.store.GetCircle(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if c == nil {
		http.NotFound(w, r)
		return
	}
	templates.CircleHeaderEdit(*c).Render(r.Context(), w)
}

func (h *Handlers) CircleUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c := model.Circle{
		ID:          id,
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}
	if c.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateCircle(r.Context(), c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.CircleHeader(c).Render(r.Context(), w)
}

func (h *Handlers) CircleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteCircle(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", "/circles")
}

func (h *Handlers) CircleMemberCreate(w http.ResponseWriter, r *http.Request) {
	circleID, ok := pathID(w, r, "id")
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
	note := strings.TrimSpace(r.FormValue("note"))

	if err := h.store.AddPersonToCircle(r.Context(), personID, circleID, note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	members, err := h.store.ListCircleMembers(r.Context(), circleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.CircleMembersList(circleID, members).Render(r.Context(), w)
}

func (h *Handlers) CircleMemberDelete(w http.ResponseWriter, r *http.Request) {
	circleID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	personID, ok := pathID(w, r, "personID")
	if !ok {
		return
	}

	if err := h.store.RemovePersonFromCircle(r.Context(), personID, circleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	members, err := h.store.ListCircleMembers(r.Context(), circleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.CircleMembersList(circleID, members).Render(r.Context(), w)
}
