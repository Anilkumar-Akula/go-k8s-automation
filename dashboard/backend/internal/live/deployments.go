package live

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type StuckDeployment struct {
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	Reason     string `json:"reason"`
}

// ListStuckDeployments mirrors project 02's detector.StuckRollout: the
// Deployment controller's own Progressing condition, not a timer we'd
// have to reimplement.
func ListStuckDeployments(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]StuckDeployment, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []StuckDeployment
	for _, dep := range deployments.Items {
		if reason, stuck := deploymentStuckReason(&dep); stuck {
			out = append(out, StuckDeployment{Namespace: dep.Namespace, Deployment: dep.Name, Reason: reason})
		}
	}
	return out, nil
}

func deploymentStuckReason(dep *appsv1.Deployment) (string, bool) {
	for _, c := range dep.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing &&
			c.Status == corev1.ConditionFalse &&
			c.Reason == "ProgressDeadlineExceeded" {
			return c.Reason, true
		}
	}
	return "", false
}
