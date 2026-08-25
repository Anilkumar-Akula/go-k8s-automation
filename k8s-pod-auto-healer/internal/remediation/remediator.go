package remediation

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Remediate deletes pod so its controller (Deployment/ReplicaSet/
// StatefulSet) replaces it. Callers must confirm via ResolveWorkloadKey
// that the pod is safe to delete before calling this.
func Remediate(ctx context.Context, clientset kubernetes.Interface, pod *corev1.Pod) error {
	err := clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("delete pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	return nil
}
