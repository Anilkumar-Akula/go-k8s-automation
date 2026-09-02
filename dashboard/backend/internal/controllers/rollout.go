package controllers

import (
	"context"

	"dashboard-api/internal/promscrape"
)

// RolloutSnapshot mirrors project 02's Prometheus metrics
// (rollout_manager_*).
type RolloutSnapshot struct {
	DeploymentsChecked float64
	RollbackTotal      float64
	RollbackFailures   float64
	WatchReconnects    float64
	StuckByReason      map[string]float64
}

func CollectRollout(ctx context.Context, url string) (RolloutSnapshot, error) {
	f, err := promscrape.Fetch(ctx, url)
	if err != nil {
		return RolloutSnapshot{}, err
	}
	snap := RolloutSnapshot{
		DeploymentsChecked: f.CounterSum("rollout_manager_deployments_checked_total"),
		RollbackTotal:      f.CounterSum("rollout_manager_rollback_total"),
		RollbackFailures:   f.CounterSum("rollout_manager_rollback_failures_total"),
		WatchReconnects:    f.CounterSum("rollout_manager_watch_reconnects_total"),
		StuckByReason:      map[string]float64{},
	}
	for _, s := range f.CounterSeries("rollout_manager_stuck_rollouts_total") {
		snap.StuckByReason[s.Labels["reason"]] = s.Value
	}
	return snap, nil
}
