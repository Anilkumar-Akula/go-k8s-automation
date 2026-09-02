package controllers

import (
	"context"

	"dashboard-api/internal/promscrape"
)

// AutoscalerSnapshot mirrors project 04's Prometheus metrics
// (autoscaler_*) for one target Deployment.
type AutoscalerSnapshot struct {
	CurrentReplicas    float64 `json:"currentReplicas"`
	DesiredReplicas    float64 `json:"desiredReplicas"`
	UtilizationPercent float64 `json:"utilizationPercent"`
	ScaleUpEvents      float64 `json:"scaleUpEvents"`
	ScaleDownEvents    float64 `json:"scaleDownEvents"`
}

func CollectAutoscaler(ctx context.Context, url, namespace, deployment string) (AutoscalerSnapshot, error) {
	f, err := promscrape.Fetch(ctx, url)
	if err != nil {
		return AutoscalerSnapshot{}, err
	}
	snap := AutoscalerSnapshot{
		CurrentReplicas:    labeledValue(f.GaugeSeries("autoscaler_current_replicas"), namespace, deployment),
		DesiredReplicas:    labeledValue(f.GaugeSeries("autoscaler_desired_replicas"), namespace, deployment),
		UtilizationPercent: labeledValue(f.GaugeSeries("autoscaler_current_utilization_percent"), namespace, deployment),
	}
	for _, s := range f.CounterSeries("autoscaler_scale_events_total") {
		if s.Labels["namespace"] != namespace || s.Labels["deployment"] != deployment {
			continue
		}
		switch s.Labels["direction"] {
		case "up":
			snap.ScaleUpEvents = s.Value
		case "down":
			snap.ScaleDownEvents = s.Value
		}
	}
	return snap, nil
}

func labeledValue(series []promscrape.Series, namespace, deployment string) float64 {
	for _, s := range series {
		if s.Labels["namespace"] == namespace && s.Labels["deployment"] == deployment {
			return s.Value
		}
	}
	return 0
}
