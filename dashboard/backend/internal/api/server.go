// Package api wires the dashboard's REST endpoints. Every response reads
// from the poller's in-memory cache and event store — handlers never call
// out to Kubernetes or a controller directly, so a slow scrape can't turn
// into a slow HTTP request.
package api

import (
	"net/http"

	"dashboard-api/internal/config"
	"dashboard-api/internal/events"
	"dashboard-api/internal/poller"
)

type Server struct {
	cfg   config.Config
	cache *poller.Cache
	store *events.Store
}

func NewServer(cfg config.Config, cache *poller.Cache, store *events.Store) *Server {
	return &Server{cfg: cfg, cache: cache, store: store}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /api/overview", s.handleOverview)
	mux.HandleFunc("GET /api/auto-healer", s.handleHealer)
	mux.HandleFunc("GET /api/rollout-manager", s.handleRollout)
	mux.HandleFunc("GET /api/resource-optimizer", s.handleOptimizer)
	mux.HandleFunc("GET /api/autoscaler", s.handleAutoscaler)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	return withCORS(mux)
}

// withCORS allows the Vite dev server (a different origin) to call this
// API during local development.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		next.ServeHTTP(w, r)
	})
}
