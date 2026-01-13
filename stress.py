import requests
import concurrent.futures
import time
import uuid
import sys
import random
from collections import Counter
from dataclasses import dataclass

# --- KONFIGURACJA ---
BASE_URL = "http://localhost:1234"
RESERVATIONS_URL = f"{BASE_URL}/reservations"
EVENT_ID = "1"

# Parametry testów
INTEGRITY_THREADS = 1000   # Ilu walczy o jedno miejsce
LOAD_COUNT = 5000          # Ile unikalnych biletów próbujemy sprzedać w teście obciążenia
MIXED_DURATION = 15       # Ile sekund ma trwać test mieszany

@dataclass
class Result:
    phase: str
    status_code: int
    duration: float
    error: str = ""

def make_reservation(seat_num):
    """Próba rezerwacji konkretnego miejsca"""
    start = time.time()
    payload = {
        "event_id": EVENT_ID,
        "seat_number": seat_num,
        "user_id": f"user_{uuid.uuid4().hex[:6]}",
        "user_name": "StressBot"
    }
    try:
        resp = requests.post(RESERVATIONS_URL, json=payload, timeout=5)
        return Result("WRITE", resp.status_code, time.time() - start)
    except Exception as e:
        return Result("WRITE", 0, time.time() - start, str(e))

def read_data():
    """Symulacja odczytu (użytkownik sprawdza dostępność)"""
    start = time.time()
    try:
        resp = requests.get(RESERVATIONS_URL, timeout=5)
        return Result("READ", resp.status_code, time.time() - start)
    except Exception as e:
        return Result("READ", 0, time.time() - start, str(e))

def print_header(title):
    print(f"\n{'='*60}")
    print(f" {title}")
    print(f"{'='*60}")

# --- FAZA 1: INTEGRALNOŚĆ ---
def test_integrity():
    print_header("FAZA 1: TEST INTEGRALNOŚCI (RACE CONDITION)")
    seat = random.randint(9000, 9999)
    print(f"[OPIS] {INTEGRITY_THREADS} wątków próbuje kupić TE SAME miejsce nr {seat}.")
    print("[CEL]  Tylko 1 sukces (201), reszta konflikty (409).")
    
    results = []
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=INTEGRITY_THREADS) as executor:
        futures = [executor.submit(make_reservation, seat) for _ in range(INTEGRITY_THREADS)]
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())
            sys.stdout.write(".")
            sys.stdout.flush()
    print("\n")

    # Analiza
    counts = Counter(r.status_code for r in results)
    success = counts[201]
    conflicts = counts[409]
    
    print(f" Wynik: {success} sukcesów, {conflicts} konfliktów.")
    if success == 1 and conflicts == INTEGRITY_THREADS - 1:
        print(" ✅ TEST ZALICZONY: System jest spójny.")
    elif success == 0:
        print(" ⚠️ OSTRZEŻENIE: Nikt nie kupił (miejsce zajęte wcześniej?).")
    else:
        print(f" ❌ BŁĄD KRYTYCZNY: Sprzedano to samo miejsce {success} razy!")

# --- FAZA 2: OBCIĄŻENIE KLASTRA ---
def test_load():
    print_header("FAZA 2: TEST WYDAJNOŚCI (CLUSTER LOAD)")
    print(f"[OPIS] Próba sprzedaży {LOAD_COUNT} RÓŻNYCH miejsc w jak najkrótszym czasie.")
    print("[CEL]  Sprawdzenie przepustowości (Requests Per Second).")

    start_time = time.time()
    results = []
    
    # Generujemy unikalne numery miejsc (np. 1000-1500)
    seats = range(1000, 1000 + LOAD_COUNT)
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=50) as executor:
        futures = [executor.submit(make_reservation, s) for s in seats]
        for i, f in enumerate(concurrent.futures.as_completed(futures)):
            results.append(f.result())
            if i % 50 == 0:
                sys.stdout.write("#")
                sys.stdout.flush()
    
    total_time = time.time() - start_time
    print("\n")
    
    # Analiza
    success = sum(1 for r in results if r.status_code == 201)
    avg_latency = sum(r.duration for r in results) / len(results)
    rps = LOAD_COUNT / total_time
    
    print(f" Czas wykonania: {total_time:.2f}s")
    print(f" Średni czas zapisu: {avg_latency*1000:.0f}ms")
    print(f" Przepustowość: {rps:.2f} req/s")
    print(f" Skuteczność: {success}/{LOAD_COUNT} ({(success/LOAD_COUNT)*100:.1f}%)")

# --- FAZA 3: RUCH MIESZANY ---
def test_mixed():
    print_header("FAZA 3: RUCH MIESZANY (READ + WRITE)")
    print(f"[OPIS] Przez {MIXED_DURATION} sekund generujemy losowy ruch (20% zapisu, 80% odczytu).")
    print("[CEL]  Symulacja realnego obciążenia 'Flash Crowd'.")

    end_time = time.time() + MIXED_DURATION
    results = []
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=20) as executor:
        while time.time() < end_time:
            # Losujemy: Czytać czy pisać?
            action = "WRITE" if random.random() < 0.2 else "READ"
            
            if action == "WRITE":
                # Losowe miejsce z dużej puli (żeby uniknąć ciągłych konfliktów)
                seat = random.randint(2000, 10000)
                futures = [executor.submit(make_reservation, seat)]
            else:
                futures = [executor.submit(read_data)]
            
            # Pobieramy wynik od razu, żeby nie zapchać pamięci
            for f in futures:
                results.append(f.result())
            
            time.sleep(0.05) # Mała pauza, żeby nie zabić localhosta

    # Analiza
    writes = [r for r in results if r.phase == "WRITE"]
    reads = [r for r in results if r.phase == "READ"]
    
    print(f"\n Wykonano łącznie: {len(results)} operacji.")
    print(f" Zapisy (Writes): {len(writes)} | Śr. czas: {sum(r.duration for r in writes)/len(writes)*1000:.0f}ms" if writes else "Brak zapisów")
    print(f" Odczyty (Reads): {len(reads)}   | Śr. czas: {sum(r.duration for r in reads)/len(reads)*1000:.0f}ms" if reads else "Brak odczytów")

if __name__ == "__main__":
    print("\n🚀 ROZPOCZYNAMY PEŁNY STRESS TEST SYSTEMU GO-TIX")
    try:
        test_integrity()
        time.sleep(1)
        test_load()
        time.sleep(1)
        test_mixed()
        print_header("KONIEC TESTU")
    except KeyboardInterrupt:
        print("\n\n⛔ Przerwano przez użytkownika.")