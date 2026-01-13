import requests
import concurrent.futures
import time
import uuid
import sys
import random
import signal
from collections import Counter
from dataclasses import dataclass

# --- KONFIGURACJA ---
BASE_URL = "http://localhost:1234"
RESERVATIONS_URL = f"{BASE_URL}/reservations"
EVENT_ID = "1"
SECTION_ID = "A"  # Nowe pole wymagane przez backend

@dataclass
class Result:
    phase: str
    status_code: int
    duration: float
    error: str = ""

# --- GENERATORY ŻĄDAŃ ---

def make_reservation(seat_nums):
    """
    Próba rezerwacji listy miejsc (ATOMOWA REZERWACJA).
    seat_nums: lista intów, np. [101, 102]
    """
    start = time.time()
    payload = {
        "event_id": EVENT_ID,
        "section_id": SECTION_ID,
        "seat_numbers": seat_nums,  # Backend oczekuje tablicy!
        "user_id": f"user_{uuid.uuid4().hex[:6]}",
        "user_name": "StressBot"
    }
    try:
        resp = requests.post(RESERVATIONS_URL, json=payload, timeout=2) # Krótki timeout dla testu chaosu
        return Result("WRITE", resp.status_code, time.time() - start)
    except Exception as e:
        return Result("WRITE", 0, time.time() - start, str(e))

def read_data():
    """Odczyt danych (GET)"""
    start = time.time()
    try:
        resp = requests.get(RESERVATIONS_URL, timeout=2)
        return Result("READ", resp.status_code, time.time() - start)
    except Exception as e:
        return Result("READ", 0, time.time() - start, str(e))

# --- SCENARIUSZE TESTOWE ---

def test_integrity():
    """Scenariusz 1: Walka o te same miejsca (Spójność)"""
    target_seats = [random.randint(100000, 999999)] # Walczymy o jedno miejsce (jako lista)
    threads = 50
    
    print(f"\n[INTEGRITY] {threads} wątków walczy o miejsce {target_seats}...")
    results = []
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=threads) as executor:
        futures = [executor.submit(make_reservation, target_seats) for _ in range(threads)]
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())
            sys.stdout.write(".")
            sys.stdout.flush()
    
    print("\n")
    counts = Counter(r.status_code for r in results)
    print(f"Wynik: Sukcesy (201): {counts[201]} | Konflikty (409): {counts[409]} | Błędy: {counts[0]}")
    
    if counts[201] == 1 and counts[409] == threads - 1:
        print("✅ TEST ZALICZONY: Idealna spójność.")
    else:
        print("⚠️  TEST NIEJEDNOZNACZNY: Sprawdź logi.")

def test_load():
    """Scenariusz 2: Zalewanie bazy nowymi rezerwacjami (Wydajność)"""
    count = 500
    base_seat = random.randint(10000, 9000000)
    # Każde żądanie to rezerwacja 1 miejsca, ale unikalnego
    seats_list = [[s] for s in range(base_seat, base_seat + count)]
    
    print(f"\n[LOAD] Próba sprzedaży {count} biletów...")
    start = time.time()
    results = []
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=50) as executor:
        futures = [executor.submit(make_reservation, s) for s in seats_list]
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())
    
    duration = time.time() - start
    success = sum(1 for r in results if r.status_code == 201)
    print(f"Czas: {duration:.2f}s | RPS: {len(results)/duration:.2f} | Skuteczność: {success}/{count}")

def test_batch_booking():
    """Scenariusz 3: Rezerwacje grupowe (Atomowość Batcha)"""
    threads = 10
    # Każdy wątek próbuje kupić TE SAME 3 miejsca na raz [A, B, C]
    target_seats = [random.randint(1000,9000) for _ in range(3)] 
    
    print(f"\n[BATCH] {threads} wątków walczy o PAKIET miejsc {target_seats}...")
    results = []
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=threads) as executor:
        futures = [executor.submit(make_reservation, target_seats) for _ in range(threads)]
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())

    counts = Counter(r.status_code for r in results)
    print(f"Wynik: {counts[201]} wygranych pakietów. (Powinno być 1)")
    if counts[201] > 1:
        print("❌ BŁĄD: Sprzedano ten sam pakiet kilka razy!")
    else:
        print("✅ TEST BATCH OK.")

def test_chaos_monkey():
    """Scenariusz 4: Chaos Monkey (Zabijanie noda w locie)"""
    print("\n💀 [CHAOS MODE] Uruchamiam ciągły ruch (20 req/s).")
    print("👉 W TYM MOMENCIE możesz zabić węzeł Cassandry (np. 'docker stop ...')")
    print("👉 Naciśnij CTRL+C aby zakończyć test.\n")
    
    time.sleep(2)
    
    running = True
    def signal_handler(sig, frame):
        nonlocal running
        running = False
        print("\n🛑 Zatrzymywanie...")

    signal.signal(signal.SIGINT, signal_handler)

    total_reqs = 0
    errors = 0
    successes = 0
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        while running:
            batch_futures = []
            # Wypuszczamy paczkę 20 zapytań
            for _ in range(20):
                if random.random() < 0.3: # 30% to zapisy
                    s = random.randint(100000, 900000)
                    batch_futures.append(executor.submit(make_reservation, [s]))
                else: # 70% to odczyty
                    batch_futures.append(executor.submit(read_data))
            
            # Czekamy na wyniki tej paczki
            for f in concurrent.futures.as_completed(batch_futures):
                res = f.result()
                total_reqs += 1
                if res.status_code in [200, 201, 409]: # 409 to też poprawna odpowiedź (konflikt logiczny)
                    successes += 1
                else:
                    errors += 1 # 0 (timeout) lub 500 (błąd serwera)
            
            # Raportowanie co sekundę
            sys.stdout.write(f"\r[STATUS] Req: {total_reqs} | OK: {successes} | ERR: {errors} (Ostatni błąd: {res.error if res.status_code == 0 else 'Brak'})   ")
            sys.stdout.flush()
            time.sleep(0.5)
            
    print("\n\n--- RAPORT CHAOSU ---")
    print(f"Przetrwało zapytań: {successes}")
    print(f"Padło (Timeout/Err): {errors}")
    if errors > 0 and successes > 0:
        print("Wniosek: System działał częściowo lub z przerwami (typowe dla awarii węzła).")

# --- MENU GŁÓWNE ---

def main():
    while True:
        print("\n" + "="*40)
        print("   💣  TICKET SNATCHER - STRESS TESTER  💣")
        print("="*40)
        print("1. Test Integralności (Pojedyncze miejsce)")
        print("2. Test Wydajności (Zalewanie bazy)")
        print("3. Test Batch (Atomowość grupowa)")
        print("4. 💀 CHAOS MODE (Zabij Noda teraz!)")
        print("0. Wyjście")
        
        choice = input("\nWybierz opcję: ")
        
        if choice == "1":
            test_integrity()
        elif choice == "2":
            test_load()
        elif choice == "3":
            test_batch_booking()
        elif choice == "4":
            test_chaos_monkey()
        elif choice == "0":
            print("Bye!")
            sys.exit(0)
        else:
            print("Nieznana opcja.")
        
        input("\n[Enter] aby wrócić do menu...")

if __name__ == "__main__":
    main()      