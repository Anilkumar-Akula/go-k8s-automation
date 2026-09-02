// Package idempotency prevents a retried POST (double-click, browser
// retry, flaky network) from executing a control action twice: the
// client sends the same Idempotency-Key on both attempts, and only the
// first one runs.
package idempotency

import (
	"sync"
	"time"
)

type entry struct {
	status int
	body   []byte
	seenAt time.Time
}

// Guard caches the response for each key it has seen, for ttl.
type Guard struct {
	mu      sync.Mutex
	entries map[string]entry
	ttl     time.Duration
}

func NewGuard(ttl time.Duration) *Guard {
	return &Guard{entries: make(map[string]entry), ttl: ttl}
}

// Check returns the cached (status, body) for key if it was recorded
// within ttl, so the caller can replay it instead of re-executing.
func (g *Guard) Check(key string) (status int, body []byte, found bool) {
	if key == "" {
		return 0, nil, false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	e, ok := g.entries[key]
	if !ok || time.Since(e.seenAt) > g.ttl {
		return 0, nil, false
	}
	return e.status, e.body, true
}

// Record caches the response for key. A no-op if key is empty (no
// Idempotency-Key header sent).
func (g *Guard) Record(key string, status int, body []byte) {
	if key == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entries[key] = entry{status: status, body: body, seenAt: time.Now()}
	g.gcLocked()
}

// gcLocked drops expired entries. Called opportunistically from Record
// rather than on a ticker — action volume is low enough that this never
// meaningfully lags behind ttl.
func (g *Guard) gcLocked() {
	cutoff := time.Now().Add(-g.ttl)
	for k, e := range g.entries {
		if e.seenAt.Before(cutoff) {
			delete(g.entries, k)
		}
	}
}
