package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func CreateReservationHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	res, err := AttemptBooking(req)
	if err != nil {
		//walka o miejsce
		if strings.Contains(err.Error(), "conflict") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) //409
			json.NewEncoder(w).Encode(map[string]string{
				"error":  "Seat already taken",
				"detail": err.Error(),
			})
			return
		}
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func GetReservationsHandler(w http.ResponseWriter, r *http.Request) {
	reservations, err := GetReservations()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reservations)
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
