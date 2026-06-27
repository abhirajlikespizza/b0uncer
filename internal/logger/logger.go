package logger

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db *sql.DB
	mu sync.Mutex
)

type Event struct {
	ID         int    `json:"id"`
	Timestamp  string `json:"timestamp"`
	Command    string `json:"command"`
	Action     string `json:"action"`
	Reason     string `json:"reason"`
	RiskScore  int    `json:"risk_score"`
	DurationMs int    `json:"duration_ms"`
}

type Stats struct {
	Total   int `json:"total"`
	Blocked int `json:"blocked"`
	Warned  int `json:"warned"`
	Allowed int `json:"allowed"`
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

func Init() error {
	dbDir := filepath.Join(homeDir(), ".b0uncer")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("creating db dir: %w", err)
	}

	var err error
	db, err = sql.Open("sqlite3", filepath.Join(dbDir, "audit.db"))
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS events (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp   DATETIME DEFAULT CURRENT_TIMESTAMP,
		command     TEXT NOT NULL,
		action      TEXT NOT NULL,
		reason      TEXT NOT NULL,
		risk_score  INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL
	)`)
	return err
}

func Log(command, action, reason string, riskScore, durationMs int) error {
	if db == nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	_, err := db.Exec(
		"INSERT INTO events (command, action, reason, risk_score, duration_ms) VALUES (?, ?, ?, ?, ?)",
		command, action, reason, riskScore, durationMs,
	)
	return err
}

func Recent(limit int) ([]Event, error) {
	if db == nil {
		return []Event{}, nil
	}
	rows, err := db.Query(
		"SELECT id, timestamp, command, action, reason, risk_score, duration_ms FROM events ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Command, &e.Action, &e.Reason, &e.RiskScore, &e.DurationMs); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func GetStats() (Stats, error) {
	var s Stats
	if db == nil {
		return s, nil
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&s.Total); err != nil {
		return s, err
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM events WHERE action = 'block'").Scan(&s.Blocked); err != nil {
		return s, err
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM events WHERE action = 'warn'").Scan(&s.Warned); err != nil {
		return s, err
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM events WHERE action = 'allow'").Scan(&s.Allowed); err != nil {
		return s, err
	}
	return s, nil
}

func Clear() error {
	if db == nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	_, err := db.Exec("DELETE FROM events")
	return err
}
