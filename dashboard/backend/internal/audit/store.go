// Package audit persists every control action to SQLite, so the audit
// trail survives a dashboard-api restart — unlike the in-memory event
// feed in internal/events, which is fine to lose.
package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Project   string    `json:"project"`
	Namespace string    `json:"namespace"`
	Resource  string    `json:"resource"`
	OldValue  string    `json:"oldValue,omitempty"`
	NewValue  string    `json:"newValue,omitempty"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"`
}

const schema = `
CREATE TABLE IF NOT EXISTS audit_events (
	id         TEXT PRIMARY KEY,
	timestamp  DATETIME NOT NULL,
	actor      TEXT NOT NULL,
	action     TEXT NOT NULL,
	project    TEXT NOT NULL,
	namespace  TEXT NOT NULL,
	resource   TEXT NOT NULL,
	old_value  TEXT,
	new_value  TEXT,
	reason     TEXT NOT NULL,
	status     TEXT NOT NULL
);
`

type Store struct {
	db *sql.DB
}

// Open creates/migrates the SQLite database at path (e.g. "dashboard.db"
// or ":memory:" for tests).
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Insert records one audit event, generating its ID and timestamp.
func (s *Store) Insert(ctx context.Context, e Event) (Event, error) {
	e.ID = uuid.NewString()
	e.Timestamp = time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_events (id, timestamp, actor, action, project, namespace, resource, old_value, new_value, reason, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Timestamp, e.Actor, e.Action, e.Project, e.Namespace, e.Resource, e.OldValue, e.NewValue, e.Reason, e.Status,
	)
	if err != nil {
		return Event{}, fmt.Errorf("insert audit event: %w", err)
	}
	return e, nil
}

// List returns the most recent audit events, newest first.
func (s *Store) List(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, timestamp, actor, action, project, namespace, resource, old_value, new_value, reason, status
		FROM audit_events ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		var oldValue, newValue sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Actor, &e.Action, &e.Project, &e.Namespace, &e.Resource, &oldValue, &newValue, &e.Reason, &e.Status); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		e.OldValue = oldValue.String
		e.NewValue = newValue.String
		out = append(out, e)
	}
	return out, rows.Err()
}
