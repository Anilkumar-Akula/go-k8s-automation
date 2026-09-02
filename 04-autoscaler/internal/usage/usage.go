// Package usage computes a target Deployment's current average CPU
// utilization: live usage (from metrics-server) as a percentage of each
// container's configured CPU request, averaged across running Pods.
package usage

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Snapshot is one poll's view of a Deployment's replica count and
// aggregate CPU utilization.
type Snapshot struct {
	CurrentReplicas int32
	// UtilizationPercent is 0 when no Pod reported both usage and a CPU
	// request (nothing to compare against yet).
	UtilizationPercent float64
}

// Collect fetches the named Deployment, lists its Pods by label selector,
// and sums live CPU usage against configured CPU requests across all
// containers that have one set.
func Collect(ctx context.Context, clientset kubernetes.Interface, metricsClient metricsv.Interface, namespace, name string) (Snapshot, error) {
	dep, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return Snapshot{}, fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
	}

	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return Snapshot{}, fmt.Errorf("parse selector: %w", err)
	}

	snap := Snapshot{CurrentReplicas: currentReplicas(dep)}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return snap, fmt.Errorf("list pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return snap, nil
	}

	reqCPUMilli := make(map[string]int64, len(pods.Items))
	for _, p := range pods.Items {
		var sum int64
		for _, c := range p.Spec.Containers {
			sum += c.Resources.Requests.Cpu().MilliValue()
		}
		reqCPUMilli[p.Namespace+"/"+p.Name] = sum
	}

	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return snap, fmt.Errorf("list pod metrics: %w", err)
	}

	var totalUsage, totalReq int64
	for _, pm := range podMetrics.Items {
		req, ok := reqCPUMilli[pm.Namespace+"/"+pm.Name]
		if !ok || req == 0 {
			continue // no CPU request configured, nothing to compare against
		}
		var usage int64
		for _, cm := range pm.Containers {
			usage += cm.Usage.Cpu().MilliValue()
		}
		totalUsage += usage
		totalReq += req
	}
	if totalReq > 0 {
		snap.UtilizationPercent = float64(totalUsage) / float64(totalReq) * 100
	}
	return snap, nil
}

func currentReplicas(dep *appsv1.Deployment) int32 {
	if dep.Spec.Replicas != nil {
		return *dep.Spec.Replicas
	}
	return 1
}
