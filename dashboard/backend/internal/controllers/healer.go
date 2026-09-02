package controllers

import (
	"context"

	"dashboard-api/internal/promscrape"
)

// HealerSnapshot mirrors project 01's Prometheus metrics
// (auto_healer_*).
type HealerSnapshot struct {
	PodsChecked         float64
	RemediationTotal    float64
	RemediationFailures float64
	WatchReconnects     float64
	UnhealthyByReason   map[string]float64
}

func CollectHealer(ctx context.Context, url string) (HealerSnapshot, error) {
	f, err := promscrape.Fetch(ctx, url)
	if err != nil {
		return HealerSnapshot{}, err
	}
	snap := HealerSnapshot{
		PodsChecked:         f.CounterSum("auto_healer_pods_checked_total"),
		RemediationTotal:    f.CounterSum("auto_healer_remediation_total"),
		RemediationFailures: f.CounterSum("auto_healer_remediation_failures_total"),
		WatchReconnects:     f.CounterSum("auto_healer_watch_reconnects_total"),
		UnhealthyByReason:   map[string]float64{},
	}
	for _, s := range f.CounterSeries("auto_healer_unhealthy_pods_total") {
		snap.UnhealthyByReason[s.Labels["reason"]] = s.Value
	}
	return snap, nil
}
