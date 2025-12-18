-- schema.cql

CREATE KEYSPACE IF NOT EXISTS ticketsnatcher 
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE ticketsnatcher;

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

-- Events
INSERT INTO events (id, name, event_date) VALUES ('1', 'Conference', 1735689600000);
INSERT INTO events (id, name, event_date) VALUES ('2', 'Workshop',   1735776000000);
INSERT INTO events (id, name, event_date) VALUES ('3', 'Webinar',    1735862400000);
INSERT INTO events (id, name, event_date) VALUES ('4', 'Meetup',     1735948800000);
INSERT INTO events (id, name, event_date) VALUES ('5', 'Hackathon',  1736035200000);

-- Users
INSERT INTO users (id, name) VALUES ('u1', 'Alice');
INSERT INTO users (id, name) VALUES ('u2', 'Bob');

-- Reservations
-- FIX: Added 'event_name' to columns and values so it is not NULL
INSERT INTO reservations (id, event_id, event_name, user_id, user_name, res_timestamp) 
VALUES ('r1', '1', 'Conference', 'u1', 'Alice', 1735680000000);

INSERT INTO reservations (id, event_id, event_name, user_id, user_name, res_timestamp) 
VALUES ('r2', '1', 'Conference', 'u2', 'Bob', 1735681000000);

INSERT INTO reservations (id, event_id, event_name, user_id, user_name, res_timestamp) 
VALUES ('r3', '2', 'Workshop', 'u2', 'Bob', 1735685000000);