package web

import "net/http"

func NewRouter(h *Handlers, staticDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", h.Today)
	mux.HandleFunc("GET /people", h.PeopleList)
	mux.HandleFunc("POST /people", h.PeopleCreate)
	mux.HandleFunc("GET /people/{id}", h.PersonDetail)
	mux.HandleFunc("PUT /people/{id}", h.PersonUpdate)
	mux.HandleFunc("DELETE /people/{id}", h.PersonDelete)
	mux.HandleFunc("GET /people/{id}/edit", h.PersonEdit)
	mux.HandleFunc("GET /people/{id}/header", h.PersonHeaderView)

	mux.HandleFunc("POST /people/{id}/contact-info", h.ContactInfoCreate)
	mux.HandleFunc("DELETE /people/{id}/contact-info/{ciID}", h.ContactInfoDelete)

	mux.HandleFunc("POST /people/{id}/important-dates", h.ImportantDateCreate)
	mux.HandleFunc("DELETE /people/{id}/important-dates/{dateID}", h.ImportantDateDelete)

	mux.HandleFunc("POST /people/{id}/relationships", h.RelationshipCreate)
	mux.HandleFunc("DELETE /people/{id}/relationships/{relID}", h.RelationshipDelete)

	mux.HandleFunc("POST /people/{id}/circles", h.PersonCircleCreate)
	mux.HandleFunc("DELETE /people/{id}/circles/{circleID}", h.PersonCircleDelete)

	mux.HandleFunc("POST /people/{id}/pets", h.PetCreate)
	mux.HandleFunc("DELETE /people/{id}/pets/{petID}", h.PetDelete)

	mux.HandleFunc("POST /people/{id}/facts", h.FactCreate)
	mux.HandleFunc("DELETE /people/{id}/facts/{factID}", h.FactDelete)

	mux.HandleFunc("POST /people/{id}/notes", h.PersonNoteCreate)
	mux.HandleFunc("DELETE /people/{id}/notes/{noteID}", h.PersonNoteDelete)

	mux.HandleFunc("POST /people/{id}/reminders", h.PersonReminderCreate)
	mux.HandleFunc("DELETE /people/{id}/reminders/{remID}", h.PersonReminderDelete)
	mux.HandleFunc("POST /people/{id}/reminders/{remID}/complete", h.PersonReminderComplete)

	mux.HandleFunc("GET /circles", h.CirclesList)
	mux.HandleFunc("POST /circles", h.CircleCreate)
	mux.HandleFunc("GET /circles/{id}", h.CircleDetail)
	mux.HandleFunc("PUT /circles/{id}", h.CircleUpdate)
	mux.HandleFunc("DELETE /circles/{id}", h.CircleDelete)
	mux.HandleFunc("GET /circles/{id}/edit", h.CircleEdit)
	mux.HandleFunc("GET /circles/{id}/header", h.CircleHeaderView)

	mux.HandleFunc("POST /circles/{id}/members", h.CircleMemberCreate)
	mux.HandleFunc("DELETE /circles/{id}/members/{personID}", h.CircleMemberDelete)

	mux.HandleFunc("GET /events", h.EventsList)
	mux.HandleFunc("POST /events", h.EventCreate)
	mux.HandleFunc("GET /events/{id}", h.EventDetail)
	mux.HandleFunc("PUT /events/{id}", h.EventUpdate)
	mux.HandleFunc("DELETE /events/{id}", h.EventDelete)
	mux.HandleFunc("GET /events/{id}/edit", h.EventEdit)
	mux.HandleFunc("GET /events/{id}/header", h.EventHeaderView)

	mux.HandleFunc("POST /events/{id}/participants", h.EventParticipantCreate)
	mux.HandleFunc("DELETE /events/{id}/participants/{personID}", h.EventParticipantDelete)

	mux.HandleFunc("GET /reminders", h.RemindersList)
	mux.HandleFunc("POST /reminders", h.ReminderCreate)
	mux.HandleFunc("DELETE /reminders/{id}", h.ReminderDelete)
	mux.HandleFunc("POST /reminders/{id}/complete", h.ReminderComplete)

	mux.HandleFunc("GET /notes", h.Notes)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	return mux
}
