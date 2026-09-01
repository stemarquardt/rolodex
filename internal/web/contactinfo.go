package web

import (
	"net/http"
	"strings"

	"rolodex/internal/web/templates"
)

func (h *Handlers) ContactInfoCreate(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	typ := strings.TrimSpace(r.FormValue("type"))
	value := strings.TrimSpace(r.FormValue("value"))
	if typ == "" || value == "" {
		http.Error(w, "type and value are required", http.StatusBadRequest)
		return
	}

	if _, err := h.store.CreateContactInfo(r.Context(), personID, typ, value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListContactInfo(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ContactInfoList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) ContactInfoDelete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	ciID, ok := pathID(w, r, "ciID")
	if !ok {
		return
	}

	if err := h.store.DeleteContactInfo(r.Context(), ciID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListContactInfo(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ContactInfoList(personID, items).Render(r.Context(), w)
}
