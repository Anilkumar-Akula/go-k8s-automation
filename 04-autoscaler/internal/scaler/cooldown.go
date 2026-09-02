package scaler

import (
	"sync"
	"time"
)

// Cooldown gates scale-up and scale-down events independently, mirroring
// HPA's stabilization windows: scale up fast to absorb load, scale down
// slow to avoid flapping on a momentary dip.
type Cooldown struct {
	mu       sync.Mutex
	lastUp   time.Time
	lastDown time.Time
}

// Allow reports whether a scale event in direction ("up" or "down") is
// past its cooldown window. A zero last-event time always allows.
func (c *Cooldown) Allow(direction string, window time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	last := c.lastUp
	if direction == "down" {
		last = c.lastDown
	}
	return last.IsZero() || time.Since(last) >= window
}

// Record marks a scale event in direction as having just happened.
func (c *Cooldown) Record(direction string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if direction == "down" {
		c.lastDown = time.Now()
		return
	}
	c.lastUp = time.Now()
}
