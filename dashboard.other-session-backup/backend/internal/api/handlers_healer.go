package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s-automation-dashboard-backend/internal/models"
)

func (s *Server) handleHealerPods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pods := s.kube.GetPods(ctx)

	unhealthy := 0
	crashLoops := 0
	oomKills := 0

	for _, p := range pods {
		if p.IsUnhealthy {
			unhealthy++
			if p.UnhealthyReason == "CrashLoopBackOff" {
				crashLoops++
			} else if p.UnhealthyReason == "OOMKilled" {
				oomKills++
			}
		}
	}

	recentLogs, _ := s.store.List(15, models.ModuleHealer)

	resp := models.HealerResponse{
		TotalMonitored:       len(pods),
		UnhealthyCount:       unhealthy,
		CrashLoopCount:       crashLoops,
		OOMKilledCount:       oomKills,
		SuccessfulRecoveries: len(recentLogs),
		Pods:                 pods,
		RecentHealingEvents:  recentLogs,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHealerRemediate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Namespace string `json:"namespace"`
		Pod       string `json:"pod"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	ctx := r.Context()
	if err := s.kube.RemediatePod(ctx, body.Namespace, body.Pod); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Record in audit store
	s.store.Record(models.AuditLogEntry{
		Module:    models.ModuleHealer,
		Action:    "MANUAL_REMEDIATE",
		Target:    body.Pod,
		Namespace: body.Namespace,
		Actor:     "user-admin",
		Status:    "success",
		Details:   fmt.Sprintf("Remediated pod %s in namespace %s", body.Pod, body.Namespace),
	})

	// Publish to event stream
	s.bus.Publish(
		models.ModuleHealer,
		models.SeveritySuccess,
		body.Pod,
		body.Namespace,
		"Pod restarted manually via Dashboard",
	)

	s.metrics.PodsHealedTotal.WithLabelValues(body.Namespace, "manual").Inc()

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Successfully triggered remediation for pod %s/%s", body.Namespace, body.Pod),
	})
}
