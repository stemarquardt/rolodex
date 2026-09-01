package web

import "net/http"

func NewRouter(h *Handlers, staticDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", h.Today)
	mux.HandleFunc("GET /people", h.PeopleList)
	mux.HandleFunc("POST /people", h.PeopleCreate)
	mux.HandleFunc("GET /people/{id}", h.PersonDetail)
	mux.HandleFunc("GET /circles", h.Circles)
	mux.HandleFunc("GET /events", h.Events)
	mux.HandleFunc("GET /reminders", h.Reminders)
	mux.HandleFunc("GET /notes", h.Notes)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	return mux
}
