// Package watcher runs a self-reconnecting Deployment watch. A raw
// client-go Watch can terminate at any time (API server restart, watch
// timeout, network blip); Run detects that and re-establishes it instead
// of silently going quiet.
package watcher

import (
	"context"
	"log/slog"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s-rollout-manager/internal/metrics"
)

// Run watches deployments in namespace ("" for all namespaces) and calls
// handler for every event, until ctx is cancelled.
func Run(ctx context.Context, clientset kubernetes.Interface, namespace string, handler func(dep *appsv1.Deployment, eventType string)) {
	reconnecting := false

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if reconnecting {
			metrics.WatchReconnects.Inc()
		}

		w, err := clientset.AppsV1().Deployments(namespace).Watch(ctx, metav1.ListOptions{})
		if err != nil {
			slog.Error("watch failed, retrying", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			reconnecting = true
			continue
		}

	drain:
		for {
			select {
			case <-ctx.Done():
				w.Stop()
				return
			case event, ok := <-w.ResultChan():
				if !ok {
					break drain
				}
				dep, ok := event.Object.(*appsv1.Deployment)
				if !ok {
					continue
				}
				handler(dep, string(event.Type))
			}
		}
		w.Stop()
		slog.Warn("deployment watch channel closed, reconnecting")
		reconnecting = true
	}
}
