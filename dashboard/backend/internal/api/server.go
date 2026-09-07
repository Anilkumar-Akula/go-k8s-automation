// Package api wires the dashboard's REST endpoints. Read endpoints answer
// from the poller's in-memory cache and event store — they never call out
// to Kubernetes or a controller directly, so a slow scrape can't turn into
// a slow HTTP request. Action endpoints (POST /api/v1/actions/*) are the
// one place this API writes to Kubernetes, through internal/actions.
package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"

	"dashboard-api/internal/actions"
	"dashboard-api/internal/audit"
	"dashboard-api/internal/auth"
	"dashboard-api/internal/config"
	"dashboard-api/internal/events"
	"dashboard-api/internal/idempotency"
	"dashboard-api/internal/poller"
)

type Server struct {
	cfg       config.Config
	clientset kubernetes.Interface
	cache     *poller.Cache
	store     *events.Store
	executor  *actions.Executor
	audit     *audit.Store
	idem      *idempotency.Guard
	auth      *auth.Authenticator
}

func NewServer(cfg config.Config, clientset kubernetes.Interface, cache *poller.Cache, store *events.Store, executor *actions.Executor, auditStore *audit.Store, idem *idempotency.Guard, authn *auth.Authenticator) *Server {
	return &Server{cfg: cfg, clientset: clientset, cache: cache, store: store, executor: executor, audit: auditStore, idem: idem, auth: authn}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.Handle("GET /metrics", promhttp.Handler())

	mux.HandleFunc("GET /api/v1/overview", s.handleOverview)
	mux.HandleFunc("GET /api/v1/auto-healer", s.handleHealer)
	mux.HandleFunc("GET /api/v1/rollout-manager", s.handleRollout)
	mux.HandleFunc("GET /api/v1/resource-optimizer", s.handleOptimizer)
	mux.HandleFunc("GET /api/v1/autoscaler", s.handleAutoscaler)
	mux.HandleFunc("GET /api/v1/events", s.handleEvents)
	mux.HandleFunc("GET /api/v1/events/stream", s.handleEventsStream)

	// Live lookups: the aggregate snapshots above only carry counts by
	// reason, not names — these back the action confirmation dialogs
	// with an actual, currently-unhealthy target.
	mux.HandleFunc("GET /api/v1/auto-healer/pods", s.handleHealerPods)
	mux.HandleFunc("GET /api/v1/rollout-manager/deployments", s.handleRolloutDeployments)

	mux.HandleFunc("POST /api/v1/actions/healer/restart", s.handleActionRestart)
	mux.HandleFunc("POST /api/v1/actions/rollout/rollback", s.handleActionRollback)
	mux.HandleFunc("POST /api/v1/actions/optimizer/apply", s.handleActionApply)
	mux.HandleFunc("POST /api/v1/actions/autoscaler/scale", s.handleActionScale)

	mux.HandleFunc("GET /api/v1/audit", s.handleAudit)

	// auth (innermost, sees the real method/path) then CORS (outermost,
	// so a preflight OPTIONS never needs a token).
	return withCORS(s.auth.Middleware(mux))
}

// withCORS allows the Vite dev server (a different origin) to call this
// API during local development.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
