// Package metrics exposes Prometheus counters/gauges for the autoscaler,
// served on /metrics by the metrics HTTP server in cmd/autoscaler.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CurrentReplicas = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "autoscaler_current_replicas",
		Help: "Current replica count of the target Deployment.",
	}, []string{"namespace", "deployment"})

	DesiredReplicas = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "autoscaler_desired_replicas",
		Help: "Replica count the autoscaler computed from observed utilization.",
	}, []string{"namespace", "deployment"})

	CurrentUtilizationPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "autoscaler_current_utilization_percent",
		Help: "Observed CPU usage as a percentage of configured CPU requests.",
	}, []string{"namespace", "deployment"})

	ScaleEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "autoscaler_scale_events_total",
		Help: "Number of times the autoscaler changed the target Deployment's replica count, by direction.",
	}, []string{"namespace", "deployment", "direction"})
)
