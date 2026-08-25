package remediation

import (
	"sync"
	"time"
)

// Retrier caps remediation attempts per workload key and computes
// exponential backoff between them. Entries older than ttl are garbage
// collected so long-running processes don't leak memory over the
// cluster's lifetime.
type Retrier struct {
	mu       sync.Mutex
	attempts map[string]retryEntry
	max      int
	initial  time.Duration
	ttl      time.Duration
	stopCh   chan struct{}
}

type retryEntry struct {
	count    int
	lastSeen time.Time
}

func NewRetrier(max int, initial, ttl time.Duration) *Retrier {
	r := &Retrier{
		attempts: make(map[string]retryEntry),
		max:      max,
		initial:  initial,
		ttl:      ttl,
		stopCh:   make(chan struct{}),
	}
	go r.gc()
	return r
}

// Stop ends the background GC goroutine.
func (r *Retrier) Stop() {
	close(r.stopCh)
}

func (r *Retrier) gc() {
	ticker := time.NewTicker(r.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case now := <-ticker.C:
			r.mu.Lock()
			for k, e := range r.attempts {
				if now.Sub(e.lastSeen) > r.ttl {
					delete(r.attempts, k)
				}
			}
			r.mu.Unlock()
		}
	}
}

// Attempt records an attempt for key and returns its number and the
// backoff to wait before acting. ok is false once max has been exceeded,
// in which case the caller should give up.
func (r *Retrier) Attempt(key string) (attempt int, backoff time.Duration, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.attempts[key]
	e.count++
	e.lastSeen = time.Now()
	r.attempts[key] = e

	if e.count > r.max {
		return e.count, 0, false
	}
	return e.count, Backoff(r.initial, e.count), true
}

// Backoff returns initial * 2^(attempt-1): attempt 1 -> initial, 2 -> 2x, 3 -> 4x, ...
func Backoff(initial time.Duration, attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return initial * time.Duration(uint64(1)<<uint(attempt-1))
}
