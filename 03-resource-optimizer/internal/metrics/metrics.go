// Package metrics exposes Prometheus counters/gauges for the resource
// optimizer, served on /metrics by the metrics HTTP server in
// cmd/resource-optimizer.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SamplesCollected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "resource_optimizer_samples_collected_total",
		Help: "Total container usage samples collected from metrics-server.",
	})

	ContainersTracked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "resource_optimizer_containers_tracked",
		Help: "Number of containers currently in the usage window.",
	})

	RecommendedCPURequestMillicores = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_optimizer_recommended_cpu_request_millicores",
		Help: "Recommended CPU request, in millicores, based on observed p50 usage.",
	}, []string{"namespace", "pod", "container"})

	RecommendedCPULimitMillicores = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_optimizer_recommended_cpu_limit_millicores",
		Help: "Recommended CPU limit, in millicores, based on observed p90 usage.",
	}, []string{"namespace", "pod", "container"})

	RecommendedMemoryRequestBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_optimizer_recommended_memory_request_bytes",
		Help: "Recommended memory request, in bytes, based on observed p50 usage.",
	}, []string{"namespace", "pod", "container"})

	RecommendedMemoryLimitBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_optimizer_recommended_memory_limit_bytes",
		Help: "Recommended memory limit, in bytes, based on observed p90 usage.",
	}, []string{"namespace", "pod", "container"})

	DriftDetected = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "resource_optimizer_drift_detected_total",
		Help: "Containers whose configured request/limit diverged from the recommendation, by resource/field/direction.",
	}, []string{"namespace", "pod", "container", "resource", "field", "direction"})
)
