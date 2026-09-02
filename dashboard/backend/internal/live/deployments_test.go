package live

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestDeploymentStuckReason(t *testing.T) {
	cases := []struct {
		name string
		dep  *appsv1.Deployment
		want bool
	}{
		{"healthy", &appsv1.Deployment{Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
			{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
		}}}, false},
		{"stuck", &appsv1.Deployment{Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
			{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"},
		}}}, true},
		{"no conditions", &appsv1.Deployment{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, stuck := deploymentStuckReason(tc.dep)
			if stuck != tc.want {
				t.Errorf("deploymentStuckReason() stuck = %v, want %v", stuck, tc.want)
			}
		})
	}
}
