// Package rollback finds a Deployment's previous stable ReplicaSet and
// rolls the Deployment back to it, capping and pacing attempts per
// workload.
package rollback

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const revisionAnnotation = "deployment.kubernetes.io/revision"

// PreviousReplicaSet finds the ReplicaSet the Deployment controller kept
// for the revision immediately before dep's current one, so rollback can
// restore its pod template — the same revision history `kubectl rollout
// undo` reads, instead of tracking our own. Returns nil if there is none
// (e.g. the Deployment has never rolled out more than once).
func PreviousReplicaSet(ctx context.Context, clientset kubernetes.Interface, dep *appsv1.Deployment) (*appsv1.ReplicaSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("build selector: %w", err)
	}

	rsList, err := clientset.AppsV1().ReplicaSets(dep.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("list replicasets: %w", err)
	}

	var owned []appsv1.ReplicaSet
	for _, rs := range rsList.Items {
		if ownedByDeployment(rs.OwnerReferences, dep.UID) {
			owned = append(owned, rs)
		}
	}
	sort.Slice(owned, func(i, j int) bool {
		return revisionOf(owned[i].Annotations) > revisionOf(owned[j].Annotations)
	})

	currentRev := revisionOf(dep.Annotations)
	for i := range owned {
		if rev := revisionOf(owned[i].Annotations); rev != 0 && rev < currentRev {
			return &owned[i], nil
		}
	}
	return nil, nil
}

func revisionOf(annotations map[string]string) int64 {
	rev, _ := strconv.ParseInt(annotations[revisionAnnotation], 10, 64)
	return rev
}

func ownedByDeployment(refs []metav1.OwnerReference, uid types.UID) bool {
	for _, r := range refs {
		if r.UID == uid {
			return true
		}
	}
	return false
}
