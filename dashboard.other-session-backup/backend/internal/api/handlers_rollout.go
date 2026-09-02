package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s-automation-dashboard-backend/internal/models"
)

func (s *Server) handleRolloutDeployments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deployments := s.kube.GetRollouts(ctx)

	stuckCount := 0
	for _, d := range deployments {
		if d.IsStuck {
			stuckCount++
		}
	}

	recentLogs, _ := s.store.List(15, models.ModuleRollout)

	resp := models.RolloutResponse{
		TotalDeployments: len(deployments),
		StuckDeployments: stuckCount,
		TotalRollbacks:   len(recentLogs),
		Deployments:      deployments,
		RecentRollbacks:  recentLogs,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRolloutRollback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Namespace  string `json:"namespace"`
		Deployment string `json:"deployment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	ctx := r.Context()
	if err := s.kube.RollbackDeployment(ctx, body.Namespace, body.Deployment); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.store.Record(models.AuditLogEntry{
		Module:    models.ModuleRollout,
		Action:    "MANUAL_ROLLBACK",
		Target:    body.Deployment,
		Namespace: body.Namespace,
		Actor:     "user-admin",
		Status:    "success",
		Details:   fmt.Sprintf("Rolled back deployment %s in %s to previous revision", body.Deployment, body.Namespace),
	})

	s.bus.Publish(
		models.ModuleRollout,
		models.SeverityWarning,
		body.Deployment,
		body.Namespace,
		"Rollback completed (reverted to previous revision)",
	)

	s.metrics.RollbacksTotal.WithLabelValues(body.Namespace, body.Deployment).Inc()

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Successfully initiated rollback for deployment %s/%s", body.Namespace, body.Deployment),
	})
}
