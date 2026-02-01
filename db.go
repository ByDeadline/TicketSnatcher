package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"strconv"
	"github.com/gocql/gocql"
)

var session *gocql.Session

func connectToCassandra() {
	hostsEnv := os.Getenv("CASSANDRA_HOSTS")
	if hostsEnv == "" {
		hostsEnv = "127.0.0.1"
	}
	hosts := strings.Split(hostsEnv, ",")

	cluster := gocql.NewCluster(hosts...)
	cluster.Port = 9042
	cluster.Keyspace = "ticketsnatcher"
	cluster.Consistency = gocql.Quorum 
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
	cluster.ReconnectInterval = 5 * time.Second

	var err error
	for i := 0; i < 30; i++ {
		session, err = cluster.CreateSession()
		if err == nil {
			fmt.Println("✅ Połączono z klastrem Cassandra:", hosts)
			return
		}
		fmt.Printf("⏳ Oczekiwanie na klaster (%d/30)... Błąd: %v\n", i+1, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("❌ Nie udało się połączyć z Cassandrą:", err)
}


func AttemptBooking(req CreateRequest) (*Reservation, error) {
	if len(req.SeatNumbers) == 0 {
		return nil, fmt.Errorf("no seats requested")
	}

	for _, seatNum := range req.SeatNumbers {
		var status string
		queryCheck := `SELECT status FROM seats WHERE event_id = ? AND section_id = ? AND seat_number = ?`
		if err := session.Query(queryCheck, req.EventID, req.SectionID, seatNum).Scan(&status); err == nil {
			if status == "SOLD" {
				return nil, fmt.Errorf("conflict: seat %d is already permanently SOLD", seatNum)
			}
		}
	}

	batch := session.NewBatch(gocql.LoggedBatch)
	for _, seatNum := range req.SeatNumbers {
		query := `UPDATE seats SET status = 'SOLD', user_id = ?, last_update = toTimestamp(now()) 
                  WHERE event_id = ? AND section_id = ? AND seat_number = ?`
		batch.Query(query, req.UserID, req.EventID, req.SectionID, seatNum)
	}

	if err := session.ExecuteBatch(batch); err != nil {
		return nil, fmt.Errorf("batch execution failed: %v", err)
	}
	
	//Symulacja laga
	time.Sleep(200 * time.Millisecond)

	for _, seatNum := range req.SeatNumbers {
		var winnerID string
		var status string
		queryVerify := `SELECT user_id, status FROM seats WHERE event_id = ? AND section_id = ? AND seat_number = ?`

		if err := session.Query(queryVerify, req.EventID, req.SectionID, seatNum).Scan(&winnerID, &status); err != nil {
			return nil, fmt.Errorf("verify read failed for seat %d", seatNum)
		}

		if winnerID != req.UserID || status != "SOLD" {
			return nil, fmt.Errorf("conflict: seat %d lost to %s", seatNum, winnerID)
		}
	}

	return createReservationLog(req)
}

func createReservationLog(req CreateRequest) (*Reservation, error) {
	id := gocql.TimeUUID().String()
	now := time.Now()
	
	seatsStr := fmt.Sprintf("%v", req.SeatNumbers)
	seatsStr = strings.Trim(seatsStr, "[]") 

	query := `INSERT INTO reservations (id, event_id, section_id, seat_numbers, user_id, user_name, res_timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)`

	if err := session.Query(query, id, req.EventID, req.SectionID, seatsStr, req.UserID, req.UserName, now).Exec(); err != nil {
		return nil, err
	}

	return &Reservation{
		ID:          id,
		EventID:     req.EventID,
		SectionID:   req.SectionID,
		SeatNumbers: req.SeatNumbers,
		UserID:      req.UserID,
		UserName:    req.UserName,
		Timestamp:   now,
	}, nil
}

func GetReservations() ([]Reservation, error) {
	var reservations []Reservation
	
	query := `SELECT id, event_id, section_id, seat_numbers, user_id, user_name, res_timestamp FROM reservations`
	iter := session.Query(query).Iter()

	var id, eventID, sectionID, seatsStr, userID, userName string
	var rawTime time.Time

	for iter.Scan(&id, &eventID, &sectionID, &seatsStr, &userID, &userName, &rawTime) {
		
		var seatNums []int
		for _, s := range strings.Fields(seatsStr) {
			if n, err := strconv.Atoi(s); err == nil {
				seatNums = append(seatNums, n)
			}
		}

		reservations = append(reservations, Reservation{
			ID:          id,
			EventID:     eventID,
			SectionID:   sectionID,
			SeatNumbers: seatNums,
			UserID:      userID,
			UserName:    userName,
			Timestamp:   rawTime,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return reservations, nil
}