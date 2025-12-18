package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
)

var session *gocql.Session

func connectToCassandra() {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Port = 9042
	cluster.Keyspace = "ticketsnatcher"
	cluster.Consistency = gocql.Quorum

	var err error
	session, err = cluster.CreateSession()
	if err != nil {
		log.Fatal("Failed to connect to Cassandra:", err)
	}
	fmt.Println("Connected to Cassandra")
}

func GetReservations() ([]Reservation, error) {
	var reservations []Reservation

	query := `SELECT id, event_id, event_name, user_id, user_name, res_timestamp FROM reservations`

	iter := session.Query(query).Iter()

	var id, eventID, eventName, userID, userName string
	var rawTime time.Time

	for iter.Scan(&id, &eventID, &eventName, &userID, &userName, &rawTime) {
		reservations = append(reservations, Reservation{
			ID:        id,
			EventID:   eventID,
			EventName: eventName,
			UserID:    userID,
			UserName:  userName,
			Timestamp: rawTime,
		})
	}
	if err := iter.Close(); err != nil {
		log.Println("[E] Error closing iterator:", err)
		return nil, err
	}

	fmt.Println("Fetched reservations:", reservations)

	return reservations, nil
}

func CreateReservation(req CreateRequest) (*Reservation, error) {
	id := gocql.TimeUUID().String()
	now := time.Now()

	query := `INSERT INTO reservations (id, event_id, user_id, user_name, res_timestamp) VALUES (?, ?, ?, ?, ?)`

	if err := session.Query(query, id, req.EventID, req.UserID, req.UserName, now).Exec(); err != nil {
		return nil, err
	}

	querySelect := `SELECT id, event_id, user_id, user_name, res_timestamp FROM reservations WHERE id = ? LIMIT 1`
	var reservation Reservation
	if err := session.Query(querySelect, id).Scan(&reservation.ID, &reservation.EventID, &reservation.UserID, &reservation.UserName, &reservation.Timestamp); err != nil {
		return nil, err
	}
	return &reservation, nil
}
