package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"k8s-resource-optimizer/internal/analyzer"
	"k8s-resource-optimizer/internal/config"
	"k8s-resource-optimizer/internal/kclient"
	"k8s-resource-optimizer/internal/metrics"
	"k8s-resource-optimizer/internal/sampler"
)

func main() {
	cfg := config.Load()

	restConfig, err := kclient.Config()
	if err != nil {
		slog.Error("failed to build kubernetes config", "error", err)
		os.Exit(1)
	}
	clientset, err := kclient.New(restConfig)
	if err != nil {
		slog.Error("failed to build kubernetes client", "error", err)
		os.Exit(1)
	}
	metricsClient, err := kclient.NewMetrics(restConfig)
	if err != nil {
		slog.Error("failed to build metrics-server client", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: promhttp.Handler()}
	go func() {
		slog.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	margins := analyzer.Margins{
		CPURequest: cfg.CPURequestMargin,
		CPULimit:   cfg.CPULimitMargin,
		MemRequest: cfg.MemRequestMargin,
		MemLimit:   cfg.MemLimitMargin,
	}
	mgr := analyzer.NewManager(cfg.WindowSize)

	slog.Info("resource optimizer starting",
		"namespace", cfg.Namespace, "poll_interval", cfg.PollInterval, "min_samples", cfg.MinSamples)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	gcTicker := time.NewTicker(cfg.WindowTTL)
	defer gcTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutdown signal received")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = metricsSrv.Shutdown(shutdownCtx)
			slog.Info("shutdown complete")
			return
		case <-gcTicker.C:
			mgr.GC(cfg.WindowTTL)
		case <-ticker.C:
			poll(ctx, clientset, metricsClient, cfg, margins, mgr)
		}
	}
}

func poll(ctx context.Context, clientset kubernetes.Interface, metricsClient metricsv.Interface, cfg config.Config, margins analyzer.Margins, mgr *analyzer.Manager) {
	samples, err := sampler.Collect(ctx, clientset, metricsClient, cfg.Namespace)
	if err != nil {
		slog.Error("failed to collect usage samples", "error", err)
		return
	}

	for _, s := range samples {
		metrics.SamplesCollected.Inc()
		key := s.Key()
		mgr.Add(key, s.CPUMilli, s.MemBytes)

		if mgr.Count(key) < cfg.MinSamples {
			continue
		}
		rec, ok := mgr.Recommend(key, margins)
		if !ok {
			continue
		}

		metrics.RecommendedCPURequestMillicores.WithLabelValues(s.Namespace, s.Pod, s.Container).Set(float64(rec.ReqCPUMilli))
		metrics.RecommendedCPULimitMillicores.WithLabelValues(s.Namespace, s.Pod, s.Container).Set(float64(rec.LimCPUMilli))
		metrics.RecommendedMemoryRequestBytes.WithLabelValues(s.Namespace, s.Pod, s.Container).Set(float64(rec.ReqMemBytes))
		metrics.RecommendedMemoryLimitBytes.WithLabelValues(s.Namespace, s.Pod, s.Container).Set(float64(rec.LimMemBytes))

		checkDrift(s, rec, cfg.DriftThreshold)
	}

	metrics.ContainersTracked.Set(float64(mgr.Tracked()))
}

func checkDrift(s sampler.Sample, rec analyzer.Recommendation, threshold float64) {
	checks := []struct {
		resource, field      string
		current, recommended int64
	}{
		{"cpu", "request", s.ReqCPUMilli, rec.ReqCPUMilli},
		{"cpu", "limit", s.LimCPUMilli, rec.LimCPUMilli},
		{"memory", "request", s.ReqMemBytes, rec.ReqMemBytes},
		{"memory", "limit", s.LimMemBytes, rec.LimMemBytes},
	}
	for _, c := range checks {
		drift, ok := analyzer.CompareDrift(c.resource, c.field, c.current, c.recommended, threshold)
		if !ok {
			continue
		}
		metrics.DriftDetected.WithLabelValues(s.Namespace, s.Pod, s.Container, drift.Resource, drift.Field, drift.Direction).Inc()
		slog.Warn("resource drift detected",
			"namespace", s.Namespace, "pod", s.Pod, "container", s.Container,
			"resource", drift.Resource, "field", drift.Field, "direction", drift.Direction,
			"current", drift.Current, "recommended", drift.Recommended)
	}
}
