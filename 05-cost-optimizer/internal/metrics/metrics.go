// Package metrics exposes Prometheus counters/gauges for the resource
// optimizer, served on /metrics by the metrics HTTP server in
// cmd/cost-optimizer.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SamplesCollected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cost_optimizer_samples_collected_total",
		Help: "Total container usage samples collected from metrics-server.",
	})

	ContainersTracked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cost_optimizer_containers_tracked",
		Help: "Number of containers currently in the usage window.",
	})

	RecommendedCPURequestMillicores = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cost_optimizer_recommended_cpu_request_millicores",
		Help: "Recommended CPU request, in millicores, based on observed p50 usage.",
	}, []string{"namespace", "pod", "container"})

	RecommendedCPULimitMillicores = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cost_optimizer_recommended_cpu_limit_millicores",
		Help: "Recommended CPU limit, in millicores, based on observed p90 usage.",
	}, []string{"namespace", "pod", "container"})

	RecommendedMemoryRequestBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cost_optimizer_recommended_memory_request_bytes",
		Help: "Recommended memory request, in bytes, based on observed p50 usage.",
	}, []string{"namespace", "pod", "container"})

	RecommendedMemoryLimitBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cost_optimizer_recommended_memory_limit_bytes",
		Help: "Recommended memory limit, in bytes, based on observed p90 usage.",
	}, []string{"namespace", "pod", "container"})

	WasteDollarsPerMonth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cost_optimizer_waste_dollars_per_month",
		Help: "Estimated $/month spent on requested CPU/memory beyond the recommended amount, per container.",
	}, []string{"namespace", "pod", "container"})

	TotalWasteDollarsPerMonth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cost_optimizer_total_waste_dollars_per_month",
		Help: "Estimated $/month of over-provisioned CPU/memory across all currently sampled containers.",
	})
)
