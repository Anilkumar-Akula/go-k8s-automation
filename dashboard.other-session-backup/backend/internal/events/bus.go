package events

import (
	"fmt"
	"sync"
	"time"

	"k8s-automation-dashboard-backend/internal/models"
)

type EventBus struct {
	mu           sync.RWMutex
	subscribers  []chan models.LiveEvent
	recentEvents []models.LiveEvent
	maxRecent    int
}

func NewEventBus() *EventBus {
	b := &EventBus{
		subscribers:  make([]chan models.LiveEvent, 0),
		recentEvents: make([]models.LiveEvent, 0, 100),
		maxRecent:    100,
	}
	b.seedInitialEvents()
	return b
}

func (b *EventBus) Subscribe() (chan models.LiveEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan models.LiveEvent, 50)
	b.subscribers = append(b.subscribers, ch)

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, sub := range b.subscribers {
			if sub == ch {
				b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
				close(ch)
				break
			}
		}
	}
	return ch, unsubscribe
}

func (b *EventBus) Publish(module models.Module, severity models.Severity, resource, namespace, message string) models.LiveEvent {
	now := time.Now()
	id := fmt.Sprintf("evt-%d", now.UnixNano())
	formatted := fmt.Sprintf("%s  %-12s %-10s %s", now.Format("15:04:05"), module, resource, message)

	evt := models.LiveEvent{
		ID:        id,
		Timestamp: now,
		Module:    module,
		Severity:  severity,
		Resource:  resource,
		Namespace: namespace,
		Message:   message,
		Formatted: formatted,
	}

	b.mu.Lock()
	b.recentEvents = append([]models.LiveEvent{evt}, b.recentEvents...)
	if len(b.recentEvents) > b.maxRecent {
		b.recentEvents = b.recentEvents[:b.maxRecent]
	}

	// Broadcast to active subscribers without blocking
	for _, ch := range b.subscribers {
		select {
		case ch <- evt:
		default:
		}
	}
	b.mu.Unlock()

	return evt
}

func (b *EventBus) GetRecent(limit int) []models.LiveEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > len(b.recentEvents) {
		limit = len(b.recentEvents)
	}
	res := make([]models.LiveEvent, limit)
	copy(res, b.recentEvents[:limit])
	return res
}

func (b *EventBus) seedInitialEvents() {
	now := time.Now()
	initial := []struct {
		offset   time.Duration
		module   models.Module
		severity models.Severity
		res      string
		msg      string
	}{
		{-10 * time.Second, models.ModuleScaler, models.SeverityInfo, "webapp", "1 → 10 replicas"},
		{-35 * time.Second, models.ModuleOptimizer, models.SeverityWarning, "api", "CPU under-provisioned (drift > 30%)"},
		{-70 * time.Second, models.ModuleHealer, models.SeveritySuccess, "worker", "Pod restarted (CrashLoopBackOff resolved)"},
		{-120 * time.Second, models.ModuleRollout, models.SeverityInfo, "backend", "Rollback completed (revision 4 restored)"},
	}

	for _, item := range initial {
		t := now.Add(item.offset)
		evt := models.LiveEvent{
			ID:        fmt.Sprintf("evt-seed-%d", t.UnixNano()),
			Timestamp: t,
			Module:    item.module,
			Severity:  item.severity,
			Resource:  item.res,
			Message:   item.msg,
			Formatted: fmt.Sprintf("%s  %-12s %-10s %s", t.Format("15:04:05"), item.module, item.res, item.msg),
		}
		b.recentEvents = append(b.recentEvents, evt)
	}
}
