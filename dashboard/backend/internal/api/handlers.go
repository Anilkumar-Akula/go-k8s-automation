package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	snap := s.cache.Snapshot()
	writeJSON(w, map[string]any{
		"cluster":  snap.Cluster,
		"lastPoll": snap.LastPoll,
		"automations": []map[string]any{
			{"name": "auto-healer", "namespace": s.cfg.AutoHealer.Namespace, "podsChecked": snap.Healer.PodsChecked, "remediations": snap.Healer.RemediationTotal},
			{"name": "rollout-manager", "namespace": s.cfg.RolloutManager.Namespace, "deploymentsChecked": snap.Rollout.DeploymentsChecked, "rollbacks": snap.Rollout.RollbackTotal},
			{"name": "resource-optimizer", "namespace": s.cfg.ResourceOptimizer.Namespace, "containersTracked": snap.Optimizer.ContainersTracked, "driftCount": len(snap.Optimizer.Drift)},
			{"name": "autoscaler", "namespace": s.cfg.Autoscaler.Namespace, "currentReplicas": snap.Autoscaler.CurrentReplicas, "desiredReplicas": snap.Autoscaler.DesiredReplicas},
		},
		"recentEvents": s.store.List(10),
	})
}

func (s *Server) handleHealer(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.cache.Snapshot().Healer)
}

func (s *Server) handleRollout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.cache.Snapshot().Rollout)
}

func (s *Server) handleOptimizer(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.cache.Snapshot().Optimizer)
}

func (s *Server) handleAutoscaler(w http.ResponseWriter, r *http.Request) {
	snap := s.cache.Snapshot().Autoscaler
	writeJSON(w, map[string]any{
		"namespace":          s.cfg.Autoscaler.Namespace,
		"deployment":         s.cfg.Autoscaler.Deployment,
		"currentReplicas":    snap.CurrentReplicas,
		"desiredReplicas":    snap.DesiredReplicas,
		"utilizationPercent": snap.UtilizationPercent,
		"scaleUpEvents":      snap.ScaleUpEvents,
		"scaleDownEvents":    snap.ScaleDownEvents,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	writeJSON(w, s.store.List(limit))
}
