package web

import (
	"net/http"

	"rolodex/internal/web/templates"
)

func (h *Handlers) RelationshipCreate(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	relatedPersonID, err := parseFormID(r, "related_person_id")
	if err != nil {
		http.Error(w, "related_person_id is required", http.StatusBadRequest)
		return
	}
	relationshipTypeID, err := parseFormID(r, "relationship_type_id")
	if err != nil {
		http.Error(w, "relationship_type_id is required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateRelationship(r.Context(), personID, relatedPersonID, relationshipTypeID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListRelationships(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.RelationshipsList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) RelationshipDelete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	relID, ok := pathID(w, r, "relID")
	if !ok {
		return
	}

	if err := h.store.DeleteRelationship(r.Context(), relID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListRelationships(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.RelationshipsList(personID, items).Render(r.Context(), w)
}
