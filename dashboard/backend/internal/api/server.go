package api

import (
	"encoding/json"
	"net/http"

	"k8s-automation-dashboard-backend/internal/events"
	"k8s-automation-dashboard-backend/internal/kubernetes"
	"k8s-automation-dashboard-backend/internal/prometheus"
	"k8s-automation-dashboard-backend/internal/store"
)

type Server struct {
	kube    *kubernetes.ClusterClient
	store   store.AuditStore
	bus     *events.EventBus
	metrics *prometheus.MetricsCollector
	mux     *http.ServeMux
}

func NewServer(
	kube *kubernetes.ClusterClient,
	store store.AuditStore,
	bus *events.EventBus,
	metrics *prometheus.MetricsCollector,
) *Server {
	s := &Server{
		kube:    kube,
		store:   store,
		bus:     bus,
		metrics: metrics,
		mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.corsMiddleware(s.mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) registerRoutes() {
	// Overview
	s.mux.HandleFunc("GET /api/v1/overview", s.handleOverview)

	// 01 Pod Auto-Healer
	s.mux.HandleFunc("GET /api/v1/healer/pods", s.handleHealerPods)
	s.mux.HandleFunc("POST /api/v1/healer/remediate", s.handleHealerRemediate)

	// 02 Rollout Manager
	s.mux.HandleFunc("GET /api/v1/rollout/deployments", s.handleRolloutDeployments)
	s.mux.HandleFunc("POST /api/v1/rollout/rollback", s.handleRolloutRollback)

	// 03 Resource Optimizer
	s.mux.HandleFunc("GET /api/v1/optimizer/recommendations", s.handleOptimizerRecommendations)
	s.mux.HandleFunc("POST /api/v1/optimizer/apply", s.handleOptimizerApply)

	// 04 Horizontal Autoscaler
	s.mux.HandleFunc("GET /api/v1/scaler/status", s.handleScalerStatus)
	s.mux.HandleFunc("POST /api/v1/scaler/scale", s.handleScalerScale)

	// Live Events & Audit
	s.mux.HandleFunc("GET /api/v1/events/stream", s.handleEventsStream)
	s.mux.HandleFunc("GET /api/v1/events/recent", s.handleEventsRecent)
	s.mux.HandleFunc("GET /api/v1/audit/logs", s.handleAuditLogs)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
