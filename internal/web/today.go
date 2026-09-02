package web

import (
	"net/http"

	"rolodex/internal/web/templates"
)

// Lookahead/staleness windows for the Today page's aggregation queries. See
// PLANNING.md's "Today pull page" section for the rationale behind each.
const (
	upcomingDatesWindowDays = 14
	eventsLookaheadDays     = 14
	staleThresholdDays      = 60
)

func (h *Handlers) Today(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dates, err := h.store.ListUpcomingImportantDates(ctx, upcomingDatesWindowDays)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	events, err := h.store.ListEventsNeedingAttention(ctx, eventsLookaheadDays)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reminders, err := h.store.ListDueReminders(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stale, err := h.store.ListStalePeople(ctx, staleThresholdDays)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.Today(dates, events, reminders, stale).Render(ctx, w)
}
