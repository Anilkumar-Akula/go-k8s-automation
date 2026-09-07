package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
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
		"recentEvents": s.store.List(10, ""),
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
	writeJSON(w, s.store.List(limit, r.URL.Query().Get("source")))
}

// handleEventsStream pushes new events over SSE as they land in the
// store, so the Overview/Live Events pages update without polling.
// Polling the existing ring buffer on a short ticker is simplest here —
// event volume is low enough that a pub/sub fan-out would be
// complexity this dashboard doesn't need yet.
func (s *Server) handleEventsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var lastSeq int64
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		for _, e := range s.store.Since(lastSeq) {
			body, _ := json.Marshal(e)
			w.Write([]byte("data: "))
			w.Write(body)
			w.Write([]byte("\n\n"))
			lastSeq = e.Seq
		}
		flusher.Flush()

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
