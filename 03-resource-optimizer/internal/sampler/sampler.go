// Package sampler collects one CPU/memory usage reading per running
// container by cross-referencing the metrics-server API (usage) with the
// core API (configured requests/limits).
package sampler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Sample is one container's current usage against its configured
// requests/limits. A zero Req/Lim field means that resource isn't set.
type Sample struct {
	Namespace   string
	Pod         string
	Container   string
	CPUMilli    int64
	MemBytes    int64
	ReqCPUMilli int64
	ReqMemBytes int64
	LimCPUMilli int64
	LimMemBytes int64
}

// Key identifies the container this sample belongs to, for windowing.
func (s Sample) Key() string {
	return s.Namespace + "/" + s.Pod + "/" + s.Container
}

// Collect lists running Pods and their live metrics in namespace ("" for
// all namespaces) and pairs them up by pod/container name. Pods the
// metrics-server hasn't reported on yet (just started) are skipped for
// this poll rather than erroring the whole collection.
func Collect(ctx context.Context, clientset kubernetes.Interface, metricsClient metricsv.Interface, namespace string) ([]Sample, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return nil, err
	}

	podSpecs := make(map[string]*corev1.Pod, len(pods.Items))
	for i := range pods.Items {
		p := &pods.Items[i]
		podSpecs[p.Namespace+"/"+p.Name] = p
	}

	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var samples []Sample
	for _, pm := range podMetrics.Items {
		pod, ok := podSpecs[pm.Namespace+"/"+pm.Name]
		if !ok {
			continue
		}
		requests, limits := containerResources(pod)
		for _, cm := range pm.Containers {
			s := Sample{
				Namespace: pm.Namespace,
				Pod:       pm.Name,
				Container: cm.Name,
				CPUMilli:  cm.Usage.Cpu().MilliValue(),
				MemBytes:  cm.Usage.Memory().Value(),
			}
			if r, ok := requests[cm.Name]; ok {
				s.ReqCPUMilli = r.Cpu().MilliValue()
				s.ReqMemBytes = r.Memory().Value()
			}
			if l, ok := limits[cm.Name]; ok {
				s.LimCPUMilli = l.Cpu().MilliValue()
				s.LimMemBytes = l.Memory().Value()
			}
			samples = append(samples, s)
		}
	}
	return samples, nil
}

func containerResources(pod *corev1.Pod) (requests, limits map[string]corev1.ResourceList) {
	requests = make(map[string]corev1.ResourceList, len(pod.Spec.Containers))
	limits = make(map[string]corev1.ResourceList, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		requests[c.Name] = c.Resources.Requests
		limits[c.Name] = c.Resources.Limits
	}
	return requests, limits
}
