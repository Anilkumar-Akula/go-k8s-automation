package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"k8s-pod-auto-healer/internal/config"
	"k8s-pod-auto-healer/internal/detector"
	"k8s-pod-auto-healer/internal/kclient"
	"k8s-pod-auto-healer/internal/metrics"
	"k8s-pod-auto-healer/internal/remediation"
	"k8s-pod-auto-healer/internal/watcher"
)

func main() {
	cfg := config.Load()

	clientset, err := kclient.New()
	if err != nil {
		slog.Error("failed to build kubernetes client", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	retrier := remediation.NewRetrier(cfg.MaxRemediationAttempts, cfg.InitialBackoff, cfg.RetryTTL)
	defer retrier.Stop()

	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: promhttp.Handler()}
	go func() {
		slog.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	var wg sync.WaitGroup
	var inFlight sync.Map // pod "namespace/name" -> struct{}, guards against duplicate remediation goroutines

	handler := func(pod *corev1.Pod, eventType string) {
		metrics.PodsChecked.Inc()

		if eventType == "DELETED" {
			slog.Info("pod deleted", "namespace", pod.Namespace, "pod", pod.Name)
			return
		}

		// Delete only marks a pod for termination (default 30s grace
		// period); the container can keep restarting inside that window
		// and fire more unhealthy events for a pod that's already on its
		// way out. Don't pile another remediation on top of that.
		if pod.DeletionTimestamp != nil {
			return
		}

		if hi, container, count := detector.HighRestartCount(pod, cfg.RestartThreshold); hi {
			slog.Warn("high restart count", "namespace", pod.Namespace, "pod", pod.Name, "container", container, "restarts", count)
		}

		result := detector.CheckPodHealth(pod)
		if !result.Unhealthy {
			return
		}
		metrics.UnhealthyPods.WithLabelValues(result.Reason).Inc()
		slog.Warn("unhealthy pod detected",
			"namespace", pod.Namespace, "pod", pod.Name,
			"reason", result.Reason, "container", result.Container)

		// A single unhealthy pod fires several MODIFIED events in quick
		// succession (each container-status field change is its own
		// event). Without this guard each one spawns its own remediation
		// goroutine, and they race through the retry budget in seconds
		// instead of pacing real attempts.
		podKey := pod.Namespace + "/" + pod.Name
		if _, alreadyRunning := inFlight.LoadOrStore(podKey, struct{}{}); alreadyRunning {
			return
		}

		wg.Add(1)
		go func(pod *corev1.Pod) {
			defer wg.Done()
			defer inFlight.Delete(podKey)
			remediate(ctx, clientset, retrier, pod)
		}(pod)
	}

	slog.Info("watching pods", "namespace", cfg.Namespace)
	watcher.Run(ctx, clientset, cfg.Namespace, handler)

	slog.Info("shutdown signal received, draining in-flight remediations")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		slog.Warn("shutdown timed out waiting for in-flight remediations")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = metricsSrv.Shutdown(shutdownCtx)
	slog.Info("shutdown complete")
}

// remediate resolves the pod's owning workload, applies the per-workload
// retry cap and backoff, then deletes the pod.
func remediate(ctx context.Context, clientset kubernetes.Interface, retrier *remediation.Retrier, pod *corev1.Pod) {
	key, ok, err := remediation.ResolveWorkloadKey(ctx, clientset, pod)
	if err != nil {
		slog.Error("failed to resolve pod owner", "namespace", pod.Namespace, "pod", pod.Name, "error", err)
		return
	}
	if !ok {
		slog.Warn("refusing to remediate: bare pod or unsupported owner kind", "namespace", pod.Namespace, "pod", pod.Name)
		return
	}

	attempt, backoff, ok := retrier.Attempt(key)
	if !ok {
		slog.Error("max remediation attempts reached, giving up", "workload", key)
		return
	}

	slog.Info("remediation scheduled", "workload", key, "attempt", attempt, "backoff", backoff)
	select {
	case <-ctx.Done():
		return
	case <-time.After(backoff):
	}

	start := time.Now()
	metrics.RemediationTotal.Inc()
	err = remediation.Remediate(ctx, clientset, pod)
	metrics.RemediationDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.RemediationFailures.Inc()
		slog.Error("remediation failed", "namespace", pod.Namespace, "pod", pod.Name, "error", err)
		return
	}
	slog.Info("pod deleted", "namespace", pod.Namespace, "pod", pod.Name, "workload", key)
}
