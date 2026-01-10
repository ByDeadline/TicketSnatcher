package main

import "time"

type event struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Date int64  `json:"date"`
}

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Reservation struct {
	ID        string    `json:"id" cql:"id"`
	EventID   string    `json:"event_id" cql:"event_id"`
	EventName string    `json:"event_name" cql:"event_name"`
	UserID    string    `json:"user_id" cql:"user_id"`
	UserName  string    `json:"user_name" cql:"user_name"`
	Timestamp time.Time `json:"timestamp" cql:"res_timestamp"`
}

type CreateRequest struct {
	EventID    string `json:"event_id"`
	SeatNumber int    `json:"seat_number"`
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}
