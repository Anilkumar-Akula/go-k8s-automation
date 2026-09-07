package sampler

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

// withPodMetrics stubs the fake metrics clientset's List call, since its
// default GVK-to-resource pluralization ("podmetricses") doesn't match the
// generated client's actual resource name ("pods"), so pre-seeding it via
// NewSimpleClientset(objects...) silently does nothing.
func withPodMetrics(items ...metricsv1beta1.PodMetrics) *metricsfake.Clientset {
	mc := metricsfake.NewSimpleClientset()
	mc.PrependReactor("list", "pods", func(action kubetesting.Action) (bool, runtime.Object, error) {
		return true, &metricsv1beta1.PodMetricsList{Items: items}, nil
	})
	return mc
}

func TestCollectPairsUsageWithSpec(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "web",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	podMetrics := metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Containers: []metricsv1beta1.ContainerMetrics{{
			Name: "web",
			Usage: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		}},
	}

	clientset := fake.NewSimpleClientset(pod)
	metricsClient := withPodMetrics(podMetrics)

	samples, err := Collect(context.Background(), clientset, metricsClient, "")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("len(samples) = %d, want 1", len(samples))
	}

	s := samples[0]
	if s.Key() != "default/app/web" {
		t.Errorf("Key() = %q, want default/app/web", s.Key())
	}
	if s.CPUMilli != 50 {
		t.Errorf("CPUMilli = %d, want 50", s.CPUMilli)
	}
	if s.MemBytes != 64*1024*1024 {
		t.Errorf("MemBytes = %d, want %d", s.MemBytes, 64*1024*1024)
	}
	if s.ReqCPUMilli != 100 {
		t.Errorf("ReqCPUMilli = %d, want 100", s.ReqCPUMilli)
	}
	if s.LimCPUMilli != 500 {
		t.Errorf("LimCPUMilli = %d, want 500", s.LimCPUMilli)
	}
}

func TestCollectSkipsMetricsWithoutMatchingPod(t *testing.T) {
	podMetrics := metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "ghost", Namespace: "default"},
		Containers: []metricsv1beta1.ContainerMetrics{{
			Name:  "web",
			Usage: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m")},
		}},
	}

	clientset := fake.NewSimpleClientset()
	metricsClient := withPodMetrics(podMetrics)

	samples, err := Collect(context.Background(), clientset, metricsClient, "")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("len(samples) = %d, want 0 for pod not found in spec list", len(samples))
	}
}
