package poller

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	"dashboard-api/internal/config"
	"dashboard-api/internal/controllers"
	"dashboard-api/internal/events"
	"dashboard-api/internal/k8sinfo"
)

// Cache holds the latest snapshot of everything the API serves reads
// from, refreshed by Run on each poll tick. Safe for concurrent reads
// from HTTP handlers while Run writes to it.
type Cache struct {
	mu sync.RWMutex

	Cluster    k8sinfo.ClusterSummary
	Healer     controllers.HealerSnapshot
	Rollout    controllers.RolloutSnapshot
	Optimizer  controllers.OptimizerSnapshot
	Autoscaler controllers.AutoscalerSnapshot
	LastPoll   time.Time
}

func NewCache() *Cache { return &Cache{} }

func (c *Cache) Snapshot() Cache {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Cache{
		Cluster: c.Cluster, Healer: c.Healer, Rollout: c.Rollout,
		Optimizer: c.Optimizer, Autoscaler: c.Autoscaler, LastPoll: c.LastPoll,
	}
}

// Run polls the cluster and every controller every cfg.PollInterval until
// ctx is cancelled, updating cache and appending synthesized events to
// store. The first poll only seeds the cache — diffing (and therefore
// events) starts from the second poll onward.
func Run(ctx context.Context, cfg config.Config, clientset kubernetes.Interface, cache *Cache, store *events.Store) {
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	primed := false
	for {
		pollOnce(ctx, cfg, clientset, cache, store, primed)
		primed = true
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func pollOnce(ctx context.Context, cfg config.Config, clientset kubernetes.Interface, cache *Cache, store *events.Store, primed bool) {
	prev := cache.Snapshot()

	cluster, err := k8sinfo.Summary(ctx, clientset)
	if err != nil {
		slog.Error("cluster summary failed", "error", err)
	}

	healer, err := controllers.CollectHealer(ctx, cfg.AutoHealer.MetricsURL)
	if err != nil {
		slog.Warn("auto-healer scrape failed", "error", err)
		healer = prev.Healer
	}
	rollout, err := controllers.CollectRollout(ctx, cfg.RolloutManager.MetricsURL)
	if err != nil {
		slog.Warn("rollout-manager scrape failed", "error", err)
		rollout = prev.Rollout
	}
	optimizer, err := controllers.CollectOptimizer(ctx, cfg.ResourceOptimizer.MetricsURL)
	if err != nil {
		slog.Warn("resource-optimizer scrape failed", "error", err)
		optimizer = prev.Optimizer
	}
	autoscaler, err := controllers.CollectAutoscaler(ctx, cfg.Autoscaler.MetricsURL, cfg.Autoscaler.Namespace, cfg.Autoscaler.Deployment)
	if err != nil {
		slog.Warn("autoscaler scrape failed", "error", err)
		autoscaler = prev.Autoscaler
	}

	if primed {
		now := time.Now().Format(time.RFC3339)
		emit := func(evs []events.Event) {
			for _, e := range evs {
				e.Time = now
				store.Add(e)
				slog.Info("event", "source", e.Source, "target", e.Target, "message", e.Message)
			}
		}
		emit(diffHealerEvents(cfg.AutoHealer.Namespace, prev.Healer, healer))
		emit(diffRolloutEvents(cfg.RolloutManager.Namespace, prev.Rollout, rollout))
		emit(diffOptimizerEvents(prev.Optimizer, optimizer))
		emit(diffAutoscalerEvents(cfg.Autoscaler.Namespace, cfg.Autoscaler.Deployment, prev.Autoscaler, autoscaler))
	}

	cache.mu.Lock()
	cache.Cluster = cluster
	cache.Healer = healer
	cache.Rollout = rollout
	cache.Optimizer = optimizer
	cache.Autoscaler = autoscaler
	cache.LastPoll = time.Now()
	cache.mu.Unlock()
}
