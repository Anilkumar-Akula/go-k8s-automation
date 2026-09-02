package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s-automation-dashboard-backend/internal/models"
)

func (s *Server) handleOptimizerRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recs := s.kube.GetOptimizerRecommendations(ctx)

	over, under, optimal := 0, 0, 0
	var totalWaste int64 = 0

	for _, rec := range recs {
		if rec.CPUDrift == "over" {
			over++
			totalWaste += (rec.CurrentCPUReq - rec.RecCPUReq)
		} else if rec.CPUDrift == "under" {
			under++
		} else {
			optimal++
		}
	}

	recentPatches, _ := s.store.List(15, models.ModuleOptimizer)

	resp := models.OptimizerResponse{
		TotalTracked:         len(recs),
		OverProvisioned:      over,
		UnderProvisioned:     under,
		OptimalCount:         optimal,
		TotalWasteCPUMilli:   totalWaste,
		Recommendations:      recs,
		RecentAppliedPatches: recentPatches,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOptimizerApply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Namespace   string `json:"namespace"`
		Deployment  string `json:"deployment"`
		Container   string `json:"container"`
		ReqCPUMilli int64  `json:"reqCpuMilli"`
		LimCPUMilli int64  `json:"limCpuMilli"`
		ReqMemBytes int64  `json:"reqMemBytes"`
		LimMemBytes int64  `json:"limMemBytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	ctx := r.Context()
	if err := s.kube.ApplyOptimizerPatch(ctx, body.Namespace, body.Deployment, body.Container, body.ReqCPUMilli, body.LimCPUMilli, body.ReqMemBytes, body.LimMemBytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.store.Record(models.AuditLogEntry{
		Module:    models.ModuleOptimizer,
		Action:    "APPLY_RESOURCE_PATCH",
		Target:    fmt.Sprintf("%s/%s", body.Deployment, body.Container),
		Namespace: body.Namespace,
		Actor:     "user-admin",
		Status:    "success",
		Details:   fmt.Sprintf("Applied right-sizing patch: %dm CPU / %dMi RAM", body.ReqCPUMilli, body.ReqMemBytes/(1024*1024)),
	})

	s.bus.Publish(
		models.ModuleOptimizer,
		models.SeveritySuccess,
		body.Deployment,
		body.Namespace,
		fmt.Sprintf("Right-sizing patch applied: %dm CPU requests", body.ReqCPUMilli),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Successfully applied resource right-sizing to %s/%s", body.Namespace, body.Deployment),
	})
}
