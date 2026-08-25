// Package watcher runs a self-reconnecting Pod watch. A raw client-go
// Watch can terminate at any time (API server restart, watch timeout,
// network blip); Run detects that and re-establishes it instead of
// silently going quiet.
package watcher

import (
	"context"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s-pod-auto-healer/internal/metrics"
)

// Run watches pods in namespace ("" for all namespaces) and calls handler
// for every event, until ctx is cancelled.
func Run(ctx context.Context, clientset kubernetes.Interface, namespace string, handler func(pod *corev1.Pod, eventType string)) {
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

		w, err := clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{})
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
				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					continue
				}
				handler(pod, string(event.Type))
			}
		}
		w.Stop()
		slog.Warn("pod watch channel closed, reconnecting")
		reconnecting = true
	}
}
