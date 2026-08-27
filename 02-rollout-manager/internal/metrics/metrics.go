// Package metrics exposes Prometheus counters/histograms for the rollout
// manager, served on /metrics by the metrics HTTP server in
// cmd/rollout-manager.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DeploymentsChecked = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rollout_manager_deployments_checked_total",
		Help: "Total deployment events processed.",
	})

	StuckRollouts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rollout_manager_stuck_rollouts_total",
		Help: "Stuck rollouts detected, by reason.",
	}, []string{"reason"})

	RollbackTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rollout_manager_rollback_total",
		Help: "Total rollback attempts executed.",
	})

	RollbackFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rollout_manager_rollback_failures_total",
		Help: "Total rollback attempts that failed.",
	})

	WatchReconnects = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rollout_manager_watch_reconnects_total",
		Help: "Total times the deployment watch had to be reconnected.",
	})

	RollbackDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "rollout_manager_rollback_duration_seconds",
		Help:    "Time spent patching a Deployment back to its previous revision.",
		Buckets: prometheus.DefBuckets,
	})
)
