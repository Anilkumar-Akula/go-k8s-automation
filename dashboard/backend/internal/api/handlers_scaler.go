package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s-automation-dashboard-backend/internal/models"
)

func (s *Server) handleScalerStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scalers := s.kube.GetScalers(ctx)

	totalEvents := 0
	for _, sc := range scalers {
		totalEvents += sc.ScaleEventsCount
	}

	recentLogs, _ := s.store.List(15, models.ModuleScaler)

	resp := models.ScalerResponse{
		ActiveScalers:  len(scalers),
		TotalEvents:    totalEvents,
		Scalers:        scalers,
		ScalingHistory: recentLogs,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleScalerScale(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Namespace  string `json:"namespace"`
		Deployment string `json:"deployment"`
		Replicas   int32  `json:"replicas"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	ctx := r.Context()
	if err := s.kube.ScaleDeployment(ctx, body.Namespace, body.Deployment, body.Replicas); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.store.Record(models.AuditLogEntry{
		Module:    models.ModuleScaler,
		Action:    "MANUAL_SCALE",
		Target:    body.Deployment,
		Namespace: body.Namespace,
		Actor:     "user-admin",
		Status:    "success",
		Details:   fmt.Sprintf("Manually scaled %s to %d replicas", body.Deployment, body.Replicas),
	})

	s.bus.Publish(
		models.ModuleScaler,
		models.SeverityInfo,
		body.Deployment,
		body.Namespace,
		fmt.Sprintf("Manual scale applied: target %d replicas", body.Replicas),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Successfully scaled %s to %d replicas", body.Deployment, body.Replicas),
	})
}
