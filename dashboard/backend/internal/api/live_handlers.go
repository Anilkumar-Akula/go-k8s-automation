package api

import (
	"encoding/json"
	"net/http"

	"dashboard-api/internal/live"
)

func (s *Server) handleHealerPods(w http.ResponseWriter, r *http.Request) {
	pods, err := live.ListUnhealthyPods(r.Context(), s.clientset, s.cfg.AutoHealer.Namespace)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(actionError{Error: err.Error()})
		return
	}
	writeJSON(w, pods)
}

func (s *Server) handleRolloutDeployments(w http.ResponseWriter, r *http.Request) {
	deployments, err := live.ListStuckDeployments(r.Context(), s.clientset, s.cfg.RolloutManager.Namespace)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(actionError{Error: err.Error()})
		return
	}
	writeJSON(w, deployments)
}
