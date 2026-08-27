package rollback

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Rollback sets dep's pod template to previousRS's and updates it. The
// Deployment controller reconciles that as an ordinary spec change: since
// the resulting template hash matches previousRS, it scales that
// ReplicaSet back up instead of creating a new one — exactly what
// `kubectl rollout undo` does.
func Rollback(ctx context.Context, clientset kubernetes.Interface, dep *appsv1.Deployment, previousRS *appsv1.ReplicaSet) error {
	fresh, err := clientset.AppsV1().Deployments(dep.Namespace).Get(ctx, dep.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get deployment %s/%s: %w", dep.Namespace, dep.Name, err)
	}

	fresh.Spec.Template = previousRS.Spec.Template
	if _, err := clientset.AppsV1().Deployments(dep.Namespace).Update(ctx, fresh, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update deployment %s/%s: %w", dep.Namespace, dep.Name, err)
	}
	return nil
}
