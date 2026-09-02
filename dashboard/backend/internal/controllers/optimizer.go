package controllers

import (
	"context"

	"dashboard-api/internal/promscrape"
)

// Recommendation is one container's current request/limit recommendation,
// read from project 03's resource_optimizer_recommended_* gauges.
type Recommendation struct {
	Namespace, Pod, Container string
	ReqCPUMilli, LimCPUMilli  float64
	ReqMemBytes, LimMemBytes  float64
}

// DriftEntry is one container/resource/field combination flagged as
// drifted, from resource_optimizer_drift_detected_total.
type DriftEntry struct {
	Namespace, Pod, Container  string
	Resource, Field, Direction string
	Count                      float64
}

type OptimizerSnapshot struct {
	ContainersTracked float64
	SamplesCollected  float64
	Recommendations   []Recommendation
	Drift             []DriftEntry
}

func CollectOptimizer(ctx context.Context, url string) (OptimizerSnapshot, error) {
	f, err := promscrape.Fetch(ctx, url)
	if err != nil {
		return OptimizerSnapshot{}, err
	}
	snap := OptimizerSnapshot{
		ContainersTracked: f.GaugeSum("resource_optimizer_containers_tracked"),
		SamplesCollected:  f.CounterSum("resource_optimizer_samples_collected_total"),
	}

	byKey := map[string]*Recommendation{}
	get := func(labels map[string]string) *Recommendation {
		key := labels["namespace"] + "/" + labels["pod"] + "/" + labels["container"]
		r, ok := byKey[key]
		if !ok {
			r = &Recommendation{Namespace: labels["namespace"], Pod: labels["pod"], Container: labels["container"]}
			byKey[key] = r
		}
		return r
	}
	for _, s := range f.GaugeSeries("resource_optimizer_recommended_cpu_request_millicores") {
		get(s.Labels).ReqCPUMilli = s.Value
	}
	for _, s := range f.GaugeSeries("resource_optimizer_recommended_cpu_limit_millicores") {
		get(s.Labels).LimCPUMilli = s.Value
	}
	for _, s := range f.GaugeSeries("resource_optimizer_recommended_memory_request_bytes") {
		get(s.Labels).ReqMemBytes = s.Value
	}
	for _, s := range f.GaugeSeries("resource_optimizer_recommended_memory_limit_bytes") {
		get(s.Labels).LimMemBytes = s.Value
	}
	for _, r := range byKey {
		snap.Recommendations = append(snap.Recommendations, *r)
	}

	for _, s := range f.CounterSeries("resource_optimizer_drift_detected_total") {
		snap.Drift = append(snap.Drift, DriftEntry{
			Namespace: s.Labels["namespace"], Pod: s.Labels["pod"], Container: s.Labels["container"],
			Resource: s.Labels["resource"], Field: s.Labels["field"], Direction: s.Labels["direction"],
			Count: s.Value,
		})
	}
	return snap, nil
}
