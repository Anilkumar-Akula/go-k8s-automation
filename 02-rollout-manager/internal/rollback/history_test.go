package rollback

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPreviousReplicaSet(t *testing.T) {
	depUID := types.UID("dep-uid")
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "demo"}}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo", Namespace: "ns", UID: depUID,
			Annotations: map[string]string{revisionAnnotation: "3"},
		},
		Spec: appsv1.DeploymentSpec{Selector: selector},
	}

	ownerRefs := []metav1.OwnerReference{{Kind: "Deployment", UID: depUID}}
	rs := func(name, revision string) appsv1.ReplicaSet {
		return appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: "ns",
				Labels:          selector.MatchLabels,
				Annotations:     map[string]string{revisionAnnotation: revision},
				OwnerReferences: ownerRefs,
			},
		}
	}

	rs1, rs2, rs3 := rs("demo-rev1", "1"), rs("demo-rev2", "2"), rs("demo-rev3", "3")
	client := fake.NewSimpleClientset(&rs1, &rs2, &rs3) // rs3 is the current revision, must be skipped

	got, err := PreviousReplicaSet(context.Background(), client, dep)
	if err != nil {
		t.Fatalf("PreviousReplicaSet() error = %v", err)
	}
	if got == nil {
		t.Fatal("PreviousReplicaSet() = nil, want demo-rev2")
	}
	if got.Name != "demo-rev2" {
		t.Errorf("PreviousReplicaSet() = %s, want demo-rev2 (highest revision below current)", got.Name)
	}
}

func TestPreviousReplicaSetNoHistory(t *testing.T) {
	depUID := types.UID("dep-uid")
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "demo"}}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo", Namespace: "ns", UID: depUID,
			Annotations: map[string]string{revisionAnnotation: "1"},
		},
		Spec: appsv1.DeploymentSpec{Selector: selector},
	}
	client := fake.NewSimpleClientset(&appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-rev1", Namespace: "ns",
			Labels:          selector.MatchLabels,
			Annotations:     map[string]string{revisionAnnotation: "1"},
			OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", UID: depUID}},
		},
	})

	got, err := PreviousReplicaSet(context.Background(), client, dep)
	if err != nil {
		t.Fatalf("PreviousReplicaSet() error = %v", err)
	}
	if got != nil {
		t.Errorf("PreviousReplicaSet() = %v, want nil (no revision before the only one)", got)
	}
}
