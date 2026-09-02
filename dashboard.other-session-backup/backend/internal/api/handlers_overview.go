package api

import (
	"net/http"
	"time"

	"k8s-automation-dashboard-backend/internal/models"
)

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cluster := s.kube.GetClusterHealth(ctx)
	recentEvents := s.bus.GetRecent(10)

	totalLogs, counts := s.store.GetStats()
	now := time.Now()

	controllers := []models.ControllerStatus{
		{
			Module:            models.ModuleHealer,
			Name:              "01 — Pod Auto-Healer",
			Status:            "Running",
			ActiveGoroutines:  2,
			TotalActionsTaken: counts[models.ModuleHealer],
			LastActionTime:    &now,
		},
		{
			Module:            models.ModuleRollout,
			Name:              "02 — Deployment Rollout Manager",
			Status:            "Running",
			ActiveGoroutines:  1,
			TotalActionsTaken: counts[models.ModuleRollout],
			LastActionTime:    &now,
		},
		{
			Module:            models.ModuleOptimizer,
			Name:              "03 — Resource Optimizer",
			Status:            "Running",
			ActiveGoroutines:  1,
			TotalActionsTaken: counts[models.ModuleOptimizer],
			LastActionTime:    &now,
		},
		{
			Module:            models.ModuleScaler,
			Name:              "04 — Horizontal Autoscaler",
			Status:            "Running",
			ActiveGoroutines:  1,
			TotalActionsTaken: counts[models.ModuleScaler],
			LastActionTime:    &now,
		},
	}

	recs := s.kube.GetOptimizerRecommendations(ctx)
	over, under := 0, 0
	var wasteCPU int64 = 0
	var wasteMem int64 = 0
	for _, rec := range recs {
		if rec.CPUDrift == "over" {
			over++
			wasteCPU += (rec.CurrentCPUReq - rec.RecCPUReq)
			wasteMem += (rec.CurrentMemReq - rec.RecMemReq)
		} else if rec.CPUDrift == "under" {
			under++
		}
	}

	scalers := s.kube.GetScalers(ctx)
	scaleUps := 0
	scaleDowns := 0
	for _, sc := range scalers {
		if sc.LastScaleDirection == "up" {
			scaleUps += sc.ScaleEventsCount
		} else {
			scaleDowns += sc.ScaleEventsCount
		}
	}

	resp := models.OverviewResponse{
		Cluster:      cluster,
		Controllers:  controllers,
		RecentEvents: recentEvents,
		ScalingStats: models.ScalingOverview{
			ActiveAutoscalers: len(scalers),
			TotalScaleEvents:  counts[models.ModuleScaler] + totalLogs,
			ScaleUpsLast24h:   scaleUps,
			ScaleDownsLast24h: scaleDowns,
		},
		WasteOverview: models.WasteOverview{
			OverProvisionedPods:  over,
			UnderProvisionedPods: under,
			TotalWasteCpuMilli:   wasteCPU,
			TotalWasteMemBytes:   wasteMem,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}
