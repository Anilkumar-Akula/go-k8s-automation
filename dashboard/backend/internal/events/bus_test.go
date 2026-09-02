package events

import (
	"testing"
	"time"

	"k8s-automation-dashboard-backend/internal/models"
)

func TestEventBusPublishAndSubscribe(t *testing.T) {
	bus := NewEventBus()
	ch, unsubscribe := bus.Subscribe()
	defer unsubscribe()

	published := bus.Publish(models.ModuleScaler, models.SeverityInfo, "webapp", "default", "1 → 10 replicas")

	select {
	case received := <-ch:
		if received.ID != published.ID {
			t.Errorf("expected ID %s, got %s", published.ID, received.ID)
		}
		if received.Formatted == "" {
			t.Errorf("expected non-empty formatted event string")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for event on subscriber channel")
	}
}
