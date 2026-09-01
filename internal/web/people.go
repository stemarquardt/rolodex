package web

import (
	"net/http"
	"strconv"
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
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

	templates.PersonDetail(detail).Render(r.Context(), w)
}
