package web

import (
	"net/http"
	"strings"

	"rolodex/internal/model"
	"rolodex/internal/web/templates"
)

func (h *Handlers) PeopleList(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	people, err := h.store.ListPeople(r.Context(), search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.PeopleList(people, search).Render(r.Context(), w)
}

func (h *Handlers) PeopleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := model.Person{
		FirstName: strings.TrimSpace(r.FormValue("first_name")),
		LastName:  strings.TrimSpace(r.FormValue("last_name")),
		Nickname:  strings.TrimSpace(r.FormValue("nickname")),
		Location:  strings.TrimSpace(r.FormValue("location")),
	}
	if p.FirstName == "" {
		http.Error(w, "first name is required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreatePerson(r.Context(), p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	people, err := h.store.ListPeople(r.Context(), "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.PeopleTable(people).Render(r.Context(), w)
}

func (h *Handlers) PersonDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	detail, err := h.store.GetPersonDetail(r.Context(), id)
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
	otherPeople := make([]model.Person, 0, len(allPeople))
	for _, p := range allPeople {
		if p.ID != id {
			otherPeople = append(otherPeople, p)
		}
	}

	circles, err := h.store.ListCircles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	relTypes, err := h.store.ListRelationshipTypes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.PersonDetail(detail, otherPeople, circles, relTypes).Render(r.Context(), w)
}

func (h *Handlers) PersonHeaderView(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p == nil {
		http.NotFound(w, r)
		return
	}
	templates.PersonHeader(*p).Render(r.Context(), w)
}

func (h *Handlers) PersonEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p == nil {
		http.NotFound(w, r)
		return
	}
	templates.PersonHeaderEdit(*p).Render(r.Context(), w)
}

func (h *Handlers) PersonUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := model.Person{
		ID:           id,
		FirstName:    strings.TrimSpace(r.FormValue("first_name")),
		LastName:     strings.TrimSpace(r.FormValue("last_name")),
		Nickname:     strings.TrimSpace(r.FormValue("nickname")),
		Location:     strings.TrimSpace(r.FormValue("location")),
		NudgeEnabled: r.FormValue("nudge_enabled") != "",
	}
	if p.FirstName == "" {
		http.Error(w, "first name is required", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdatePerson(r.Context(), p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.PersonHeader(p).Render(r.Context(), w)
}

func (h *Handlers) PersonDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.store.DeletePerson(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", "/people")
}
