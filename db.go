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
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 10} // Zwiększone retry dla chaos testów
	cluster.ReconnectInterval = 1 * time.Second
	cluster.Timeout = 2 * time.Second

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
	
	time.Sleep(200 * time.Millisecond)

	for _, seatNum := range req.SeatNumbers {
		var winnerID string
		var status string
		queryVerify := `SELECT user_id, status FROM seats WHERE event_id = ? AND section_id = ? AND seat_number = ?`

		if err := session.Query(queryVerify, req.EventID, req.SectionID, seatNum).Scan(&winnerID, &status); err != nil {
			return nil, fmt.Errorf("verify read failed for seat %d (network error?)", seatNum)
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

func CancelReservation(reservationID string) error {
	var eventID, sectionID, seatsStr string
	queryGet := `SELECT event_id, section_id, seat_numbers FROM reservations WHERE id = ?`
	
	if err := session.Query(queryGet, reservationID).Scan(&eventID, &sectionID, &seatsStr); err != nil {
		if err == gocql.ErrNotFound {
			return fmt.Errorf("rezerwacja nie istnieje")
		}
		return err
	}

	var seatNums []int
	for _, s := range strings.Fields(seatsStr) {
		if n, err := strconv.Atoi(s); err == nil {
			seatNums = append(seatNums, n)
		}
	}

	batch := session.NewBatch(gocql.LoggedBatch)
	
	batch.Query(`DELETE FROM reservations WHERE id = ?`, reservationID)

	for _, seatNum := range seatNums {
		queryFree := `UPDATE seats SET status = 'AVAILABLE', user_id = null 
		              WHERE event_id = ? AND section_id = ? AND seat_number = ?`
		batch.Query(queryFree, eventID, sectionID, seatNum)
	}

	if err := session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("błąd podczas anulowania: %v", err)
	}

	return nil
}

func TestMultiSectorBatch() error {
	fmt.Println("--- ROZPOCZYNAM TEST LOGGED BATCH (Cross-Partition) ---")
	
	// 1. Upewnij się, że miejsca są wolne (Reset)
	session.Query("UPDATE seats SET status='AVAILABLE', user_id=null WHERE event_id='1' AND section_id='A' AND seat_number=900").Exec()
	session.Query("UPDATE seats SET status='AVAILABLE', user_id=null WHERE event_id='1' AND section_id='B' AND seat_number=900").Exec()

	// 2. Przygotuj BATCH
	// Zauważ: Modyfikujemy dwie RÓŻNE partycje (Section A i Section B)
	batch := session.NewBatch(gocql.LoggedBatch)
	
	userID := "batch_tester_01"
	
	// Zapytanie 1: Sekcja A
	batch.Query(`UPDATE seats SET status='SOLD', user_id=? 
		WHERE event_id='1' AND section_id='A' AND seat_number=900`, userID)
		
	// Zapytanie 2: Sekcja B
	batch.Query(`UPDATE seats SET status='SOLD', user_id=? 
		WHERE event_id='1' AND section_id='B' AND seat_number=900`, userID)

	// Dodaj wpis do rezerwacji (żeby zachować spójność logiczną)
	resID := gocql.TimeUUID().String()
	batch.Query(`INSERT INTO reservations (id, event_id, section_id, seat_numbers, user_id, user_name, res_timestamp) 
		VALUES (?, ?, ?, ?, ?, ?, toTimestamp(now()))`, 
		resID, "1", "MULTI-SECTOR", "900(A), 900(B)", userID, "BatchTester")

	fmt.Println("Wysyłanie BATCHA do klastra...")
	
	// 3. Wykonanie
	if err := session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("BATCH Failed: %v", err)
	}
	
	fmt.Println("✅ BATCH wykonany pomyślnie.")
	return nil
}

func CheckBatchResults() {
	var userA, statusA string
	var userB, statusB string
	
	session.Query("SELECT user_id, status FROM seats WHERE event_id='1' AND section_id='A' AND seat_number=900").Scan(&userA, &statusA)
	
	session.Query("SELECT user_id, status FROM seats WHERE event_id='1' AND section_id='B' AND seat_number=900").Scan(&userB, &statusB)
	
	fmt.Printf("Stan Sekcji A (Miejsce 900): User=%s, Status=%s\n", userA, statusA)
	fmt.Printf("Stan Sekcji B (Miejsce 900): User=%s, Status=%s\n", userB, statusB)
	
	if userA == "batch_tester_01" && userB == "batch_tester_01" {
		fmt.Println("SUKCES: Obie partycje zostały zaktualizowane atomowo!")
	} else {
		fmt.Println("error!")
	}
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