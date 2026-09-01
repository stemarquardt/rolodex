package web

import (
	"net/http"
	"strings"

	"rolodex/internal/web/templates"
)

func (h *Handlers) PersonCircleCreate(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	circleName := strings.TrimSpace(r.FormValue("circle_name"))
	note := strings.TrimSpace(r.FormValue("note"))
	if circleName == "" {
		http.Error(w, "circle_name is required", http.StatusBadRequest)
		return
	}

	circleID, err := h.store.GetOrCreateCircleByName(r.Context(), circleName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.AddPersonToCircle(r.Context(), personID, circleID, note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListCircleMemberships(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.CirclesList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) PersonCircleDelete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	circleID, ok := pathID(w, r, "circleID")
	if !ok {
		return
	}

	if err := h.store.RemovePersonFromCircle(r.Context(), personID, circleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListCircleMemberships(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.CirclesList(personID, items).Render(r.Context(), w)
}
