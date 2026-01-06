CREATE KEYSPACE IF NOT EXISTS ticketsnatcher 
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE ticketsnatcher;

-- (TU BĘDĄ KONFLIKTY)
CREATE TABLE IF NOT EXISTS seats (
    event_id text,
    seat_number int,
    status text,      -- 'AVAILABLE', 'SOLD'
    user_id text,     -- Kto kupił
    last_update timestamp,
    PRIMARY KEY (event_id, seat_number)
);

CREATE TABLE IF NOT EXISTS events (
    id text PRIMARY KEY,
    name text,
    event_date timestamp
);

CREATE TABLE IF NOT EXISTS users (
    id text PRIMARY KEY,
    name text
);

CREATE TABLE IF NOT EXISTS reservations (
    id text PRIMARY KEY,
    event_id text,
    event_name text,
    user_id text,
    user_name text,
    res_timestamp timestamp
);

INSERT INTO seats (event_id, seat_number, status, user_id) VALUES ('1', 100, 'AVAILABLE', null);

INSERT INTO events (id, name, event_date) VALUES ('1', 'Conference', 1735689600000);
INSERT INTO users (id, name) VALUES ('u1', 'Alice');
INSERT INTO users (id, name) VALUES ('u2', 'Bob');