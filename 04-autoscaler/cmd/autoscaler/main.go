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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"k8s-autoscaler/internal/config"
	"k8s-autoscaler/internal/kclient"
	"k8s-autoscaler/internal/metrics"
	"k8s-autoscaler/internal/scaler"
	"k8s-autoscaler/internal/usage"
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

	slog.Info("autoscaler starting",
		"namespace", cfg.Namespace, "deployment", cfg.Deployment, "poll_interval", cfg.PollInterval,
		"target_cpu_percent", cfg.TargetCPUPercent, "min_replicas", cfg.MinReplicas, "max_replicas", cfg.MaxReplicas)

	var cooldown scaler.Cooldown
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutdown signal received")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = metricsSrv.Shutdown(shutdownCtx)
			slog.Info("shutdown complete")
			return
		case <-ticker.C:
			poll(ctx, clientset, metricsClient, cfg, &cooldown)
		}
	}
}

func poll(ctx context.Context, clientset kubernetes.Interface, metricsClient metricsv.Interface, cfg config.Config, cooldown *scaler.Cooldown) {
	snap, err := usage.Collect(ctx, clientset, metricsClient, cfg.Namespace, cfg.Deployment)
	if err != nil {
		slog.Error("failed to collect usage", "error", err)
		return
	}

	metrics.CurrentReplicas.WithLabelValues(cfg.Namespace, cfg.Deployment).Set(float64(snap.CurrentReplicas))
	metrics.CurrentUtilizationPercent.WithLabelValues(cfg.Namespace, cfg.Deployment).Set(snap.UtilizationPercent)

	desired := scaler.Decide(snap.CurrentReplicas, snap.UtilizationPercent, cfg.TargetCPUPercent, cfg.Tolerance, cfg.MinReplicas, cfg.MaxReplicas)
	metrics.DesiredReplicas.WithLabelValues(cfg.Namespace, cfg.Deployment).Set(float64(desired))

	if desired == snap.CurrentReplicas {
		return
	}

	direction := "up"
	cooldownWindow := cfg.ScaleUpCooldown
	if desired < snap.CurrentReplicas {
		direction = "down"
		cooldownWindow = cfg.ScaleDownCooldown
	}

	if !cooldown.Allow(direction, cooldownWindow) {
		slog.Info("scale decision suppressed by cooldown",
			"namespace", cfg.Namespace, "deployment", cfg.Deployment,
			"direction", direction, "current", snap.CurrentReplicas, "desired", desired)
		return
	}

	if err := applyScale(ctx, clientset, cfg.Namespace, cfg.Deployment, desired); err != nil {
		slog.Error("failed to apply scale", "error", err)
		return
	}
	cooldown.Record(direction)
	metrics.ScaleEvents.WithLabelValues(cfg.Namespace, cfg.Deployment, direction).Inc()
	slog.Info("scaled deployment",
		"namespace", cfg.Namespace, "deployment", cfg.Deployment,
		"direction", direction, "from", snap.CurrentReplicas, "to", desired,
		"utilization_percent", snap.UtilizationPercent, "target_percent", cfg.TargetCPUPercent)
}

// applyScale patches the Deployment's scale subresource rather than the
// full spec — the same narrow write HPA itself performs, needing only
// deployments/scale RBAC instead of write access to the whole Deployment.
func applyScale(ctx context.Context, clientset kubernetes.Interface, namespace, name string, replicas int32) error {
	current, err := clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	current.Spec.Replicas = replicas
	_, err = clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, current, metav1.UpdateOptions{})
	return err
}
