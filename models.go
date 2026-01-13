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
	ID          string    `json:"id" cql:"id"`
	EventID     string    `json:"event_id" cql:"event_id"`
	SectionID   string    `json:"section_id" cql:"section_id"`     
	SeatNumbers []int     `json:"seat_numbers" cql:"seat_numbers"` 
	UserID      string    `json:"user_id" cql:"user_id"`
	UserName    string    `json:"user_name" cql:"user_name"`
	Timestamp   time.Time `json:"timestamp" cql:"res_timestamp"`
}

type CreateRequest struct {
	EventID     string `json:"event_id"`
	SectionID   string `json:"section_id"`   
	SeatNumbers []int  `json:"seat_numbers"` 
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}