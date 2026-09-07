// Package analyzer keeps a rolling window of usage samples per container
// and turns it into request/limit recommendations, VPA-style: recommend
// off observed percentiles rather than a fixed rule of thumb.
package analyzer

import (
	"sync"
	"time"
)

// window holds the most recent cpu/mem usage samples for one container.
type window struct {
	mu       sync.Mutex
	cap      int
	cpu      []int64
	mem      []int64
	lastSeen time.Time
}

func newWindow(cap int) *window {
	return &window{cap: cap}
}

func (w *window) add(cpuMilli, memBytes int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cpu = pushCapped(w.cpu, cpuMilli, w.cap)
	w.mem = pushCapped(w.mem, memBytes, w.cap)
	w.lastSeen = time.Now()
}

func pushCapped(s []int64, v int64, cap int) []int64 {
	s = append(s, v)
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}

// snapshot returns copies of the current samples, safe to sort/read
// without holding the window lock.
func (w *window) snapshot() (cpu, mem []int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	cpu = append([]int64(nil), w.cpu...)
	mem = append([]int64(nil), w.mem...)
	return cpu, mem
}

func (w *window) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.cpu)
}

func (w *window) staleSince(ttl time.Duration) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return time.Since(w.lastSeen) > ttl
}

// percentile returns the p-th percentile (0-100) of sorted using nearest-rank.
// values must already be sorted ascending.
func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := max(int((p/100)*float64(len(sorted)-1)), 0)
	idx = min(idx, len(sorted)-1)
	return sorted[idx]
}
