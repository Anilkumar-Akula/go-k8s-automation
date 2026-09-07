// Package events holds the dashboard's recent-activity feed: a small
// in-memory ring buffer fed by the poller. Persistent audit history
// (SQLite/Postgres) is a later phase — this is enough for the Overview
// and Live Events pages to show what just happened.
package events

import "sync"

type Event struct {
	Seq     int64  `json:"seq"`
	Time    string `json:"time"`
	Source  string `json:"source"` // "auto-healer" | "rollout-manager" | "resource-optimizer" | "autoscaler"
	Target  string `json:"target"` // namespace/pod or namespace/deployment
	Message string `json:"message"`
}

// Store is a fixed-capacity, newest-first ring buffer, safe for
// concurrent use by the poller (writer) and HTTP handlers (readers).
type Store struct {
	mu   sync.RWMutex
	cap  int
	buf  []Event
	next int64
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
	s.next++
	e.Seq = s.next
	s.buf = append([]Event{e}, s.buf...)
	if len(s.buf) > s.cap {
		s.buf = s.buf[:s.cap]
	}
}

// Since returns events with Seq > afterSeq, oldest-first — for the SSE
// stream to send only what a client hasn't seen yet.
func (s *Store) Since(afterSeq int64) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Event
	for _, e := range s.buf { // buf is newest-first
		if e.Seq <= afterSeq {
			break
		}
		out = append(out, e)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
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
