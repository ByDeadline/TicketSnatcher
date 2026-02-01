package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	EventID   = "1"
	SectionID = "A"
)

func main() {
	connectToCassandra()
	defer session.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n" + strings.Repeat("=", 40))
		fmt.Println("   🎟️  TICKET SNATCHER (Cluster Edition) 🎟️")
		fmt.Println(strings.Repeat("=", 40))
		fmt.Println("1. Kup bilet (Interaktywnie)")
		fmt.Println("2. Pokaż wszystkie rezerwacje")
		fmt.Println("3. 💣 Uruchom STRESS TEST (Wielowątkowy)")
		fmt.Println("0. Wyjście")
		fmt.Print("\nWybierz opcję: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			handleInteractiveBooking(reader)
		case "2":
			handleListReservations()
		case "3":
			handleStressTest(reader)
		case "0":
			fmt.Println("Do widzenia!")
			os.Exit(0)
		default:
			fmt.Println("Nieznana opcja.")
		}
	}
}

func handleInteractiveBooking(r *bufio.Reader) {
	fmt.Print("Podaj numer miejsca (np. 101): ")
	seatStr, _ := r.ReadString('\n')
	seatNum, err := strconv.Atoi(strings.TrimSpace(seatStr))
	if err != nil {
		fmt.Println("Błąd: To nie jest liczba.")
		return
	}

	userID := fmt.Sprintf("cli_user_%d", rand.Intn(1000))
	req := CreateRequest{
		EventID:     EventID,
		SectionID:   SectionID,
		SeatNumbers: []int{seatNum},
		UserID:      userID,
		UserName:    "ConsoleUser",
	}

	fmt.Printf("Próba rezerwacji miejsca %d...\n", seatNum)
	start := time.Now()
	
	res, err := AttemptBooking(req) 
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ BŁĄD: %v (Czas: %v)\n", err, duration)
	} else {
		fmt.Printf("✅ SUKCES! ID Rezerwacji: %s (Czas: %v)\n", res.ID, duration)
	}
}

func handleListReservations() {
	reservations, err := GetReservations()
	if err != nil {
		fmt.Println("Błąd bazy:", err)
		return
	}
	fmt.Printf("\nZnaleziono %d rezerwacji:\n", len(reservations))
	for _, r := range reservations {
		fmt.Printf("- [%s] %s zarezerwował miejsca %v\n", r.Timestamp.Format("15:04:05"), r.UserName, r.SeatNumbers)
	}
}

func handleStressTest(r *bufio.Reader) {
	fmt.Print("Ile wątków (np. 50): ")
	tStr, _ := r.ReadString('\n')
	threads, _ := strconv.Atoi(strings.TrimSpace(tStr))
	if threads <= 0 { threads = 10 }

	fmt.Print("O które miejsce walczyć? (np. 999): ")
	sStr, _ := r.ReadString('\n')
	targetSeat, _ := strconv.Atoi(strings.TrimSpace(sStr))

	fmt.Printf("\n🚀 START: %d wątków walczy o miejsce %d...\n", threads, targetSeat)
	
	var wg sync.WaitGroup
	results := make(chan string, threads)
	
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := CreateRequest{
				EventID:     EventID,
				SectionID:   SectionID,
				SeatNumbers: []int{targetSeat},
				UserID:      fmt.Sprintf("bot_%d", id),
				UserName:    "StressBot",
			}
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			
			_, err := AttemptBooking(req)
			if err == nil {
				results <- "SUCCESS"
			} else {
				if strings.Contains(err.Error(), "conflict") {
					results <- "CONFLICT"
				} else {
					results <- "ERROR"
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)

	success := 0
	conflicts := 0
	errors := 0

	for res := range results {
		switch res {
		case "SUCCESS": success++
		case "CONFLICT": conflicts++
		case "ERROR": errors++
		}
	}

	fmt.Println("\n--- WYNIKI TESTU ---")
	fmt.Printf("Sukcesy (201):   %d (Powinno być 1)\n", success)
	fmt.Printf("Konflikty (409): %d\n", conflicts)
	fmt.Printf("Błędy:           %d\n", errors)
	
	if success == 1 && conflicts == threads-1 {
		fmt.Println("TEST SPÓJNOŚCI ZALICZONY IDEALNIE")
	} else {
		fmt.Println("TEST NIEJEDNOZNACZNY (możliwe błędy sieci lub bazy)")
	}
}