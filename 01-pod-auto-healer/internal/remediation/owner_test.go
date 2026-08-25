package remediation

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestResolveWorkloadKey(t *testing.T) {
	t.Run("bare pod refused", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "bare"}}

		_, ok, err := ResolveWorkloadKey(context.Background(), clientset, pod)
		if err != nil || ok {
			t.Fatalf("got ok=%v err=%v, want ok=false err=nil", ok, err)
		}
	})

	t.Run("job owner refused", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "ns",
				Name:            "job-pod",
				OwnerReferences: []metav1.OwnerReference{{Kind: "Job", Name: "myjob"}},
			},
		}

		_, ok, err := ResolveWorkloadKey(context.Background(), clientset, pod)
		if err != nil || ok {
			t.Fatalf("got ok=%v err=%v, want ok=false err=nil", ok, err)
		}
	})

	t.Run("statefulset owner allowed", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "ns",
				Name:            "sts-0",
				OwnerReferences: []metav1.OwnerReference{{Kind: "StatefulSet", Name: "mysts"}},
			},
		}

		key, ok, err := ResolveWorkloadKey(context.Background(), clientset, pod)
		if err != nil || !ok {
			t.Fatalf("got ok=%v err=%v, want ok=true err=nil", ok, err)
		}
		if want := "StatefulSet/ns/mysts"; key != want {
			t.Errorf("key = %q, want %q", key, want)
		}
	})

	t.Run("replicaset owned by deployment resolves to deployment", func(t *testing.T) {
		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "ns",
				Name:            "myapp-abc123",
				OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "myapp"}},
			},
		}
		clientset := fake.NewSimpleClientset(rs)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "ns",
				Name:            "myapp-abc123-xyz",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "myapp-abc123"}},
			},
		}

		key, ok, err := ResolveWorkloadKey(context.Background(), clientset, pod)
		if err != nil || !ok {
			t.Fatalf("got ok=%v err=%v, want ok=true err=nil", ok, err)
		}
		if want := "Deployment/ns/myapp"; key != want {
			t.Errorf("key = %q, want %q", key, want)
		}
	})

	t.Run("bare replicaset resolves to replicaset", func(t *testing.T) {
		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "bare-rs"},
		}
		clientset := fake.NewSimpleClientset(rs)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       "ns",
				Name:            "bare-rs-xyz",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "bare-rs"}},
			},
		}

		key, ok, err := ResolveWorkloadKey(context.Background(), clientset, pod)
		if err != nil || !ok {
			t.Fatalf("got ok=%v err=%v, want ok=true err=nil", ok, err)
		}
		if want := "ReplicaSet/ns/bare-rs"; key != want {
			t.Errorf("key = %q, want %q", key, want)
		}
	})
}
