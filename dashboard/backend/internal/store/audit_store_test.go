package store

import (
	"testing"

	"k8s-automation-dashboard-backend/internal/models"
)

func TestAuditStoreRecordAndList(t *testing.T) {
	s := NewMemoryAuditStore()

	entry := models.AuditLogEntry{
		Module:    models.ModuleHealer,
		Action:    "RESTART_POD",
		Target:    "test-pod",
		Namespace: "default",
		Actor:     "system-controller",
		Status:    "success",
		Details:   "Test restart",
	}

	recorded, err := s.Record(entry)
	if err != nil {
		t.Fatalf("failed to record entry: %v", err)
	}
	if recorded.ID == 0 {
		t.Errorf("expected non-zero ID")
	}

	list, err := s.List(10, models.ModuleHealer)
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}
	if len(list) == 0 {
		t.Errorf("expected at least 1 entry for module Healer")
	}
	if list[0].Target != "test-pod" {
		t.Errorf("expected newest entry target 'test-pod', got '%s'", list[0].Target)
	}
}

func TestAuditStoreGetStats(t *testing.T) {
	s := NewMemoryAuditStore()
	total, counts := s.GetStats()

	if total == 0 {
		t.Errorf("expected non-zero initial seeded logs")
	}
	if counts[models.ModuleScaler] == 0 {
		t.Errorf("expected scaler logs count > 0")
	}
}
