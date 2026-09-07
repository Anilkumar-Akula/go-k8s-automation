package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"k8s-cost-optimizer/internal/analyzer"
	"k8s-cost-optimizer/internal/config"
	"k8s-cost-optimizer/internal/cost"
	"k8s-cost-optimizer/internal/kclient"
	"k8s-cost-optimizer/internal/metrics"
	"k8s-cost-optimizer/internal/sampler"
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
	rates := cost.Rates{CPUCorePerHour: cfg.CPUCorePerHourRate, MemGiBPerHour: cfg.MemGiBPerHourRate}
	mgr := analyzer.NewManager(cfg.WindowSize)

	slog.Info("cost optimizer starting",
		"namespace", cfg.Namespace, "poll_interval", cfg.PollInterval, "min_samples", cfg.MinSamples,
		"cpu_core_hour_rate", cfg.CPUCorePerHourRate, "mem_gib_hour_rate", cfg.MemGiBPerHourRate)

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
			poll(ctx, clientset, metricsClient, cfg, margins, rates, mgr)
		}
	}
}

// wasted is one container's estimated over-provisioning cost, kept only
// long enough to sort and log the worst offenders for this poll.
type wasted struct {
	namespace, pod, container string
	dollarsPerMonth           float64
}

func poll(ctx context.Context, clientset kubernetes.Interface, metricsClient metricsv.Interface, cfg config.Config, margins analyzer.Margins, rates cost.Rates, mgr *analyzer.Manager) {
	samples, err := sampler.Collect(ctx, clientset, metricsClient, cfg.Namespace)
	if err != nil {
		slog.Error("failed to collect usage samples", "error", err)
		return
	}

	var totalWaste float64
	var worst []wasted

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

		waste := cost.MonthlyWaste(s.ReqCPUMilli, rec.ReqCPUMilli, s.ReqMemBytes, rec.ReqMemBytes, rates)
		metrics.WasteDollarsPerMonth.WithLabelValues(s.Namespace, s.Pod, s.Container).Set(waste)
		totalWaste += waste
		if waste > 0 {
			worst = append(worst, wasted{namespace: s.Namespace, pod: s.Pod, container: s.Container, dollarsPerMonth: waste})
		}
	}

	metrics.ContainersTracked.Set(float64(mgr.Tracked()))
	metrics.TotalWasteDollarsPerMonth.Set(totalWaste)
	reportWorst(worst, cfg.TopN)
}

// reportWorst logs the topN most wasteful containers this poll, most
// expensive first — the demo-friendly equivalent of the ranked report
// a real cost dashboard would render.
func reportWorst(worst []wasted, topN int) {
	if len(worst) == 0 {
		return
	}
	slices.SortFunc(worst, func(a, b wasted) int {
		switch {
		case a.dollarsPerMonth > b.dollarsPerMonth:
			return -1
		case a.dollarsPerMonth < b.dollarsPerMonth:
			return 1
		default:
			return 0
		}
	})
	if topN > len(worst) {
		topN = len(worst)
	}
	for _, w := range worst[:topN] {
		slog.Warn("over-provisioned container",
			"namespace", w.namespace, "pod", w.pod, "container", w.container,
			"estimated_waste_dollars_per_month", w.dollarsPerMonth)
	}
}
