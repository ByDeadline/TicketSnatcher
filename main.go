package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	connectToCassandra()
	defer session.Close()

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/reservations", CreateReservationHandler)
	r.Get("/reservations", GetReservationsHandler)
	r.Get("/", HealthCheckHandler)

	log.Println("Server running on 127.0.0.1:1234...")
	log.Fatal(http.ListenAndServe("127.0.0.1:1234", r))
}
