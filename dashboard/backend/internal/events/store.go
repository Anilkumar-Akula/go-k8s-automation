// Package events holds the dashboard's recent-activity feed: a small
// in-memory ring buffer fed by the poller. Persistent audit history
// (SQLite/Postgres) is a later phase — this is enough for the Overview
// and Live Events pages to show what just happened.
package events

import "sync"

type Event struct {
	Time    string `json:"time"`
	Source  string `json:"source"` // "auto-healer" | "rollout-manager" | "resource-optimizer" | "autoscaler"
	Target  string `json:"target"` // namespace/pod or namespace/deployment
	Message string `json:"message"`
}

// Store is a fixed-capacity, newest-first ring buffer, safe for
// concurrent use by the poller (writer) and HTTP handlers (readers).
type Store struct {
	mu  sync.RWMutex
	cap int
	buf []Event
}

func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 100
	}
	return &Store{cap: capacity}
}

// Add prepends an event, dropping the oldest once capacity is exceeded.
func (s *Store) Add(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = append([]Event{e}, s.buf...)
	if len(s.buf) > s.cap {
		s.buf = s.buf[:s.cap]
	}
}

// List returns up to limit of the most recent events (0 = all buffered),
// optionally restricted to one source ("" = no filter).
func (s *Store) List(limit int, source string) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []Event
	for _, e := range s.buf {
		if source == "" || e.Source == source {
			matched = append(matched, e)
		}
	}
	if limit <= 0 || limit > len(matched) {
		limit = len(matched)
	}
	out := make([]Event, limit)
	copy(out, matched[:limit])
	return out
}
