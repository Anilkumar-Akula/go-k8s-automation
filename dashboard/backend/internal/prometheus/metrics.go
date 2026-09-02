package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsCollector struct {
	Registry *prometheus.Registry

	// Healer metrics
	PodsHealedTotal     *prometheus.CounterVec
	RemediationAttempts *prometheus.CounterVec

	// Rollout metrics
	RollbacksTotal *prometheus.CounterVec

	// Optimizer metrics
	ResourceDriftGauge *prometheus.GaugeVec

	// Scaler metrics
	CurrentReplicasGauge *prometheus.GaugeVec
	DesiredReplicasGauge *prometheus.GaugeVec
	CPUUtilizationGauge  *prometheus.GaugeVec
}

func NewMetricsCollector() *MetricsCollector {
	reg := prometheus.NewRegistry()
	factory := promauto.With(reg)

	return &MetricsCollector{
		Registry: reg,
		PodsHealedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "k8s_dashboard_pods_healed_total",
				Help: "Total number of pods auto-healed",
			},
			[]string{"namespace", "reason"},
		),
		RemediationAttempts: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "k8s_dashboard_remediation_attempts_total",
				Help: "Total number of remediation attempts triggered",
			},
			[]string{"namespace", "pod"},
		),
		RollbacksTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "k8s_dashboard_rollbacks_total",
				Help: "Total number of rollbacks executed",
			},
			[]string{"namespace", "deployment"},
		),
		ResourceDriftGauge: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "k8s_dashboard_resource_drift_ratio",
				Help: "Observed resource drift ratio vs recommendation",
			},
			[]string{"namespace", "container", "resource"},
		),
		CurrentReplicasGauge: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "k8s_dashboard_scaler_current_replicas",
				Help: "Current replica count for autoscaler target",
			},
			[]string{"namespace", "deployment"},
		),
		DesiredReplicasGauge: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "k8s_dashboard_scaler_desired_replicas",
				Help: "Desired replica count computed by autoscaler",
			},
			[]string{"namespace", "deployment"},
		),
		CPUUtilizationGauge: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "k8s_dashboard_scaler_cpu_utilization_percent",
				Help: "Observed CPU utilization percent",
			},
			[]string{"namespace", "deployment"},
		),
	}
}
