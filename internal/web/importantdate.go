package web

import (
	"net/http"
	"strconv"
	"strings"

	"rolodex/internal/web/templates"
)

func (h *Handlers) ImportantDateCreate(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	typ := strings.TrimSpace(r.FormValue("type"))
	label := strings.TrimSpace(r.FormValue("label"))
	month, err := strconv.Atoi(r.FormValue("month"))
	if err != nil || month < 1 || month > 12 {
		http.Error(w, "month must be between 1 and 12", http.StatusBadRequest)
		return
	}
	day, err := strconv.Atoi(r.FormValue("day"))
	if err != nil || day < 1 || day > 31 {
		http.Error(w, "day must be between 1 and 31", http.StatusBadRequest)
		return
	}
	if typ == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}
	// Label is only really needed to distinguish multiple dates of the same
	// type (e.g. two "Anniversary" entries) or to say what a "Custom" date
	// actually is — for the common case (a single Birthday) it's just
	// redundant with the type, so default to that instead of forcing the
	// user to retype it.
	if label == "" {
		label = typ
	}

	var year *int
	if raw := strings.TrimSpace(r.FormValue("year")); raw != "" {
		y, err := strconv.Atoi(raw)
		if err != nil {
			http.Error(w, "year must be a number", http.StatusBadRequest)
			return
		}
		year = &y
	}

	if _, err := h.store.CreateImportantDate(r.Context(), personID, typ, label, month, day, year); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListImportantDates(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ImportantDatesList(personID, items).Render(r.Context(), w)
}

func (h *Handlers) ImportantDateDelete(w http.ResponseWriter, r *http.Request) {
	personID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	dateID, ok := pathID(w, r, "dateID")
	if !ok {
		return
	}

	if err := h.store.DeleteImportantDate(r.Context(), dateID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items, err := h.store.ListImportantDates(r.Context(), personID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.ImportantDatesList(personID, items).Render(r.Context(), w)
}
