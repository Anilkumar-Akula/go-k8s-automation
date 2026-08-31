package analyzer

import (
	"slices"
	"sync"
	"time"
)

// Recommendation is the suggested request/limit for one container,
// derived from the observed usage distribution: request tracks typical
// (p50) usage, limit covers spikes (p90).
type Recommendation struct {
	ReqCPUMilli int64
	LimCPUMilli int64
	ReqMemBytes int64
	LimMemBytes int64
}

// Margins scale percentile usage up to a recommended value, leaving
// headroom above what was actually observed.
type Margins struct {
	CPURequest float64
	CPULimit   float64
	MemRequest float64
	MemLimit   float64
}

// Drift describes a container whose configured request/limit has grown
// too far from what the observed usage recommends.
type Drift struct {
	Resource    string // "cpu" or "memory"
	Field       string // "request" or "limit"
	Direction   string // "under" (configured too low) or "over" (configured too high)
	Current     int64
	Recommended int64
}

// Manager tracks a rolling usage window per container, keyed by
// "namespace/pod/container".
type Manager struct {
	mu      sync.RWMutex
	windows map[string]*window
	cap     int
}

func NewManager(windowSize int) *Manager {
	return &Manager{windows: make(map[string]*window), cap: windowSize}
}

// Add records one usage sample for key, creating its window on first use.
func (m *Manager) Add(key string, cpuMilli, memBytes int64) {
	m.mu.Lock()
	w, ok := m.windows[key]
	if !ok {
		w = newWindow(m.cap)
		m.windows[key] = w
	}
	m.mu.Unlock()
	w.add(cpuMilli, memBytes)
}

// Count returns the number of samples collected for key so far.
func (m *Manager) Count(key string) int {
	m.mu.RLock()
	w, ok := m.windows[key]
	m.mu.RUnlock()
	if !ok {
		return 0
	}
	return w.count()
}

// Recommend computes a request/limit recommendation for key from its
// current usage window. ok is false if the window doesn't exist.
func (m *Manager) Recommend(key string, margins Margins) (Recommendation, bool) {
	m.mu.RLock()
	w, ok := m.windows[key]
	m.mu.RUnlock()
	if !ok {
		return Recommendation{}, false
	}

	cpu, mem := w.snapshot()
	if len(cpu) == 0 {
		return Recommendation{}, false
	}
	slices.Sort(cpu)
	slices.Sort(mem)

	rec := Recommendation{
		ReqCPUMilli: scale(percentile(cpu, 50), margins.CPURequest),
		LimCPUMilli: scale(percentile(cpu, 90), margins.CPULimit),
		ReqMemBytes: scale(percentile(mem, 50), margins.MemRequest),
		LimMemBytes: scale(percentile(mem, 90), margins.MemLimit),
	}
	// A limit recommendation below the request recommendation would be
	// nonsensical (and rejected by the API server if ever applied).
	if rec.LimCPUMilli < rec.ReqCPUMilli {
		rec.LimCPUMilli = rec.ReqCPUMilli
	}
	if rec.LimMemBytes < rec.ReqMemBytes {
		rec.LimMemBytes = rec.ReqMemBytes
	}
	return rec, true
}

func scale(v int64, margin float64) int64 {
	return int64(float64(v) * margin)
}

// GC drops windows that haven't received a sample in ttl, so Pods that
// were deleted or rescheduled don't leak memory forever.
func (m *Manager) GC(ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, w := range m.windows {
		if w.staleSince(ttl) {
			delete(m.windows, k)
		}
	}
}

// Tracked returns the number of containers currently being watched.
func (m *Manager) Tracked() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.windows)
}

// CompareDrift compares current (0 = unset) against recommended and
// reports every resource/field pair that differs by more than
// threshold (a fraction, e.g. 0.3 for 30%). An unset current value is
// always reported as "under" drift, since there's nothing to compare
// a margin against.
func CompareDrift(resource, field string, current, recommended int64, threshold float64) (Drift, bool) {
	if current == 0 {
		if recommended == 0 {
			return Drift{}, false
		}
		return Drift{Resource: resource, Field: field, Direction: "under", Current: current, Recommended: recommended}, true
	}
	diff := float64(recommended-current) / float64(current)
	switch {
	case diff > threshold:
		return Drift{Resource: resource, Field: field, Direction: "under", Current: current, Recommended: recommended}, true
	case diff < -threshold:
		return Drift{Resource: resource, Field: field, Direction: "over", Current: current, Recommended: recommended}, true
	default:
		return Drift{}, false
	}
}
