// Package metrics exposes Prometheus counters/histograms for the healer,
// served on /metrics by the metrics HTTP server in cmd/auto-healer.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PodsChecked = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auto_healer_pods_checked_total",
		Help: "Total pod events processed.",
	})

	UnhealthyPods = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auto_healer_unhealthy_pods_total",
		Help: "Unhealthy pods detected, by reason.",
	}, []string{"reason"})

	RemediationTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auto_healer_remediation_total",
		Help: "Total remediation attempts (pod deletes) executed.",
	})

	RemediationFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auto_healer_remediation_failures_total",
		Help: "Total remediation attempts that failed.",
	})

	WatchReconnects = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auto_healer_watch_reconnects_total",
		Help: "Total times the pod watch had to be reconnected.",
	})

	RemediationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "auto_healer_remediation_duration_seconds",
		Help:    "Time spent deleting a pod during remediation.",
		Buckets: prometheus.DefBuckets,
	})
)
