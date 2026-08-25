package remediation

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResolveWorkloadKey identifies the workload that owns pod, so retries
// can be tracked per-workload instead of per-pod-name (Deployments give
// every replacement pod a new name). ok is false for pods we refuse to
// remediate: bare pods (no owner — deleting them is permanent), Jobs
// (a restart changes completion semantics), and DaemonSets.
func ResolveWorkloadKey(ctx context.Context, clientset kubernetes.Interface, pod *corev1.Pod) (key string, ok bool, err error) {
	if len(pod.OwnerReferences) == 0 {
		return "", false, nil
	}

	owner := pod.OwnerReferences[0]
	switch owner.Kind {
	case "StatefulSet":
		return fmt.Sprintf("StatefulSet/%s/%s", pod.Namespace, owner.Name), true, nil

	case "ReplicaSet":
		rs, err := clientset.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
		if err != nil {
			return "", false, err
		}
		for _, rsOwner := range rs.OwnerReferences {
			if rsOwner.Kind == "Deployment" {
				return fmt.Sprintf("Deployment/%s/%s", pod.Namespace, rsOwner.Name), true, nil
			}
		}
		return fmt.Sprintf("ReplicaSet/%s/%s", pod.Namespace, owner.Name), true, nil

	default:
		return "", false, nil
	}
}
