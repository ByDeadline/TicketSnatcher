package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gocql/gocql"
)

var session *gocql.Session

func connectToCassandra() {
	cassandraHost := os.Getenv("CASSANDRA_HOST")
	if cassandraHost == "" {
		cassandraHost = "127.0.0.1"
	}

	cluster := gocql.NewCluster(cassandraHost)
	cluster.Port = 9042
	cluster.Keyspace = "ticketsnatcher"
	cluster.Consistency = gocql.Quorum

	var err error
	for i := 0; i < 15; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			fmt.Println("Connected to Cassandra at", cassandraHost)
			return
		}
		fmt.Println("Waiting for Cassandra...", err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("Failed to connect to Cassandra:", err)
}

func AttemptBooking(req CreateRequest) (*Reservation, error) {
	queryUpdate := `UPDATE seats SET status = 'SOLD', user_id = ?, last_update = toTimestamp(now()) 
                    WHERE event_id = ? AND seat_number = ?`

	if err := session.Query(queryUpdate, req.UserID, req.EventID, req.SeatNumber).Exec(); err != nil {
		return nil, err
	}

	time.Sleep(200 * time.Millisecond)

	var winnerID string
	var status string
	queryCheck := `SELECT user_id, status FROM seats WHERE event_id = ? AND seat_number = ?`

	if err := session.Query(queryCheck, req.EventID, req.SeatNumber).Scan(&winnerID, &status); err != nil {
		return nil, err
	}

	if winnerID == req.UserID && status == "SOLD" {
		return createReservationLog(req)
	} else {
		return nil, fmt.Errorf("conflict: seat taken by %s", winnerID)
	}
}

func createReservationLog(req CreateRequest) (*Reservation, error) {
	id := gocql.TimeUUID().String()
	now := time.Now()
	eventName := "Unknown Event"

	query := `INSERT INTO reservations (id, event_id, event_name, user_id, user_name, res_timestamp) VALUES (?, ?, ?, ?, ?, ?)`

	if err := session.Query(query, id, req.EventID, eventName, req.UserID, req.UserName, now).Exec(); err != nil {
		return nil, err
	}

	return &Reservation{
		ID:        id,
		EventID:   req.EventID,
		EventName: eventName,
		UserID:    req.UserID,
		UserName:  req.UserName,
		Timestamp: now,
	}, nil
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
		return nil, err
	}
	return reservations, nil
}
