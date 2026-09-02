// Package k8sinfo reads coarse cluster health (nodes, pod counts) — the
// numbers the Overview page needs that don't come from any controller's
// own metrics.
package k8sinfo

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ClusterSummary struct {
	NodeCount   int
	NodesReady  int
	PodsTotal   int
	PodsRunning int
	PodsPending int
	PodsFailed  int
}

func Summary(ctx context.Context, clientset kubernetes.Interface) (ClusterSummary, error) {
	var summary ClusterSummary

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return summary, err
	}
	summary.NodeCount = len(nodes.Items)
	for _, n := range nodes.Items {
		if nodeReady(n) {
			summary.NodesReady++
		}
	}

	pods, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return summary, err
	}
	summary.PodsTotal = len(pods.Items)
	for _, p := range pods.Items {
		switch p.Status.Phase {
		case corev1.PodRunning:
			summary.PodsRunning++
		case corev1.PodPending:
			summary.PodsPending++
		case corev1.PodFailed:
			summary.PodsFailed++
		}
	}
	return summary, nil
}

func nodeReady(n corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
