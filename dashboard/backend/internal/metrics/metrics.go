// Package metrics exposes Prometheus counters/histograms for the
// dashboard-api itself (its control actions), served on /metrics —
// separate from the per-controller metrics it scrapes in internal/promscrape.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ActionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dashboard_actions_total",
		Help: "Total control actions attempted, by project and action.",
	}, []string{"project", "action"})

	ActionsSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dashboard_actions_success_total",
		Help: "Control actions that completed successfully, by project and action.",
	}, []string{"project", "action"})

	ActionsFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dashboard_actions_failed_total",
		Help: "Control actions that failed, by project and action.",
	}, []string{"project", "action"})

	ActionDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dashboard_action_duration_seconds",
		Help:    "Time spent executing a control action, by project and action.",
		Buckets: prometheus.DefBuckets,
	}, []string{"project", "action"})
)
