package store

import (
	"sync"
	"time"

	"k8s-automation-dashboard-backend/internal/models"
)

// AuditStore provides persistent logging and retrieval of all automation decisions and user actions.
type AuditStore interface {
	Record(entry models.AuditLogEntry) (models.AuditLogEntry, error)
	List(limit int, module models.Module) ([]models.AuditLogEntry, error)
	GetRecentActivity(limit int) ([]models.AuditLogEntry, error)
	GetStats() (total int, byModule map[models.Module]int)
}

// MemoryAuditStore is a thread-safe, in-memory implementation of AuditStore.
type MemoryAuditStore struct {
	mu      sync.RWMutex
	entries []models.AuditLogEntry
	nextID  int64
}

func NewMemoryAuditStore() *MemoryAuditStore {
	s := &MemoryAuditStore{
		entries: make([]models.AuditLogEntry, 0, 1000),
		nextID:  1,
	}
	s.seedInitialLogs()
	return s
}

func (s *MemoryAuditStore) Record(entry models.AuditLogEntry) (models.AuditLogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry.ID = s.nextID
	s.nextID++
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Prepend for newest-first order
	s.entries = append([]models.AuditLogEntry{entry}, s.entries...)
	if len(s.entries) > 2000 {
		s.entries = s.entries[:2000]
	}
	return entry, nil
}

func (s *MemoryAuditStore) List(limit int, module models.Module) ([]models.AuditLogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.AuditLogEntry
	for _, entry := range s.entries {
		if module != "" && entry.Module != module {
			continue
		}
		result = append(result, entry)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (s *MemoryAuditStore) GetRecentActivity(limit int) ([]models.AuditLogEntry, error) {
	return s.List(limit, "")
}

func (s *MemoryAuditStore) GetStats() (int, map[models.Module]int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[models.Module]int)
	for _, e := range s.entries {
		counts[e.Module]++
	}
	return len(s.entries), counts
}

func (s *MemoryAuditStore) seedInitialLogs() {
	now := time.Now()
	initial := []models.AuditLogEntry{
		{
			ID:        1,
			Timestamp: now.Add(-3 * time.Minute),
			Module:    models.ModuleScaler,
			Action:    "SCALE_UP",
			Target:    "demo-app",
			Namespace: "autoscaler-demo",
			Actor:     "system-controller",
			Status:    "success",
			Details:   "Scaled from 3 to 4 replicas (CPU utilization 72.4% > target 50%)",
		},
		{
			ID:        2,
			Timestamp: now.Add(-8 * time.Minute),
			Module:    models.ModuleRollout,
			Action:    "AUTOMATED_ROLLBACK",
			Target:    "checkout-v2",
			Namespace: "production",
			Actor:     "system-controller",
			Status:    "success",
			Details:   "Rollback to revision 4 completed (ProgressDeadlineExceeded cleared)",
		},
		{
			ID:        3,
			Timestamp: now.Add(-14 * time.Minute),
			Module:    models.ModuleHealer,
			Action:    "REMEDIATE_POD",
			Target:    "auth-gateway-55d8c7-4k8p2",
			Namespace: "production",
			Actor:     "system-controller",
			Status:    "success",
			Details:   "Deleted pod in CrashLoopBackOff state (Owner: Deployment/auth-gateway)",
		},
		{
			ID:        4,
			Timestamp: now.Add(-22 * time.Minute),
			Module:    models.ModuleOptimizer,
			Action:    "APPLY_RECOMMENDATION",
			Target:    "payment-service",
			Namespace: "production",
			Actor:     "user-admin",
			Status:    "success",
			Details:   "Updated container requests to 480m CPU / 490MiB RAM",
		},
	}

	for _, entry := range initial {
		s.entries = append([]models.AuditLogEntry{entry}, s.entries...)
		if entry.ID >= s.nextID {
			s.nextID = entry.ID + 1
		}
	}
}
