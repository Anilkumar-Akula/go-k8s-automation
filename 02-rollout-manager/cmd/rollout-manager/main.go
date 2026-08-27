package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"k8s-rollout-manager/internal/config"
	"k8s-rollout-manager/internal/detector"
	"k8s-rollout-manager/internal/kclient"
	"k8s-rollout-manager/internal/metrics"
	"k8s-rollout-manager/internal/rollback"
	"k8s-rollout-manager/internal/watcher"
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

	retrier := rollback.NewRetrier(cfg.MaxRollbackAttempts, cfg.InitialBackoff, cfg.RetryTTL)
	defer retrier.Stop()

	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: promhttp.Handler()}
	go func() {
		slog.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	var wg sync.WaitGroup
	var inFlight sync.Map     // "namespace/name" -> struct{}, guards against duplicate rollback goroutines
	var lastRollback sync.Map // "namespace/name" -> corev1.PodTemplateSpec last rolled back to

	handler := func(dep *appsv1.Deployment, eventType string) {
		metrics.DeploymentsChecked.Inc()

		if eventType == "DELETED" || dep.DeletionTimestamp != nil {
			return
		}

		result := detector.StuckRollout(dep)
		if !result.Stuck {
			return
		}

		key := dep.Namespace + "/" + dep.Name

		// The Deployment controller doesn't clear the Progressing/
		// ProgressDeadlineExceeded condition the instant we patch the
		// template back — it takes another reconcile. In that window the
		// watch keeps delivering MODIFIED events that still carry the
		// stale condition, which looks identical to a fresh stuck
		// rollout. Comparing against the template we already rolled back
		// to (rather than just deduping in-flight work) is what actually
		// breaks the loop, since it survives across separate goroutine
		// runs, not just one.
		if tmpl, ok := lastRollback.Load(key); ok && reflect.DeepEqual(tmpl, dep.Spec.Template) {
			return
		}

		metrics.StuckRollouts.WithLabelValues(result.Reason).Inc()
		slog.Warn("stuck rollout detected",
			"namespace", dep.Namespace, "deployment", dep.Name, "reason", result.Reason)

		// A stuck Deployment's status keeps getting re-synced (replica
		// counts, other conditions) which fires several MODIFIED events
		// while it's stuck. Without this guard each one spawns its own
		// rollback goroutine.
		if _, alreadyRunning := inFlight.LoadOrStore(key, struct{}{}); alreadyRunning {
			return
		}

		wg.Add(1)
		go func(dep *appsv1.Deployment) {
			defer wg.Done()
			defer inFlight.Delete(key)
			if tmpl, ok := rollbackDeployment(ctx, clientset, retrier, dep); ok {
				lastRollback.Store(key, tmpl)
			}
		}(dep)
	}

	slog.Info("watching deployments", "namespace", cfg.Namespace)
	watcher.Run(ctx, clientset, cfg.Namespace, handler)

	slog.Info("shutdown signal received, draining in-flight rollbacks")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		slog.Warn("shutdown timed out waiting for in-flight rollbacks")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = metricsSrv.Shutdown(shutdownCtx)
	slog.Info("shutdown complete")
}

// rollbackDeployment finds the previous stable ReplicaSet and, subject to
// the per-workload retry cap and backoff, rolls dep's pod template back
// to it. On success it returns the template rolled back to, so the caller
// can recognize and ignore the stale stuck-rollout events that follow
// before the Deployment controller's status catches up.
func rollbackDeployment(ctx context.Context, clientset kubernetes.Interface, retrier *rollback.Retrier, dep *appsv1.Deployment) (corev1.PodTemplateSpec, bool) {
	key := "Deployment/" + dep.Namespace + "/" + dep.Name

	previousRS, err := rollback.PreviousReplicaSet(ctx, clientset, dep)
	if err != nil {
		slog.Error("failed to look up previous replicaset", "deployment", key, "error", err)
		return corev1.PodTemplateSpec{}, false
	}
	if previousRS == nil {
		slog.Warn("no previous revision to roll back to", "deployment", key)
		return corev1.PodTemplateSpec{}, false
	}

	attempt, backoff, ok := retrier.Attempt(key)
	if !ok {
		slog.Error("max rollback attempts reached, giving up", "deployment", key)
		return corev1.PodTemplateSpec{}, false
	}

	slog.Info("rollback scheduled", "deployment", key, "attempt", attempt, "backoff", backoff, "target_replicaset", previousRS.Name)
	select {
	case <-ctx.Done():
		return corev1.PodTemplateSpec{}, false
	case <-time.After(backoff):
	}

	start := time.Now()
	metrics.RollbackTotal.Inc()
	err = rollback.Rollback(ctx, clientset, dep, previousRS)
	metrics.RollbackDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.RollbackFailures.Inc()
		slog.Error("rollback failed", "deployment", key, "error", err)
		return corev1.PodTemplateSpec{}, false
	}
	slog.Info("deployment rolled back", "deployment", key, "target_replicaset", previousRS.Name)
	return previousRS.Spec.Template, true
}
