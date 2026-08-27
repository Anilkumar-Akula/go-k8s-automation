package detector

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStuckRollout(t *testing.T) {
	cases := []struct {
		name       string
		conditions []appsv1.DeploymentCondition
		wantStuck  bool
	}{
		{
			name:       "no conditions",
			conditions: nil,
			wantStuck:  false,
		},
		{
			name: "progressing true",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
			},
			wantStuck: false,
		},
		{
			name: "progress deadline exceeded",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"},
			},
			wantStuck: true,
		},
		{
			name: "available condition false is not a stuck rollout",
			conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse, Reason: "MinimumReplicasUnavailable"},
			},
			wantStuck: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
				Status:     appsv1.DeploymentStatus{Conditions: tc.conditions},
			}
			got := StuckRollout(dep)
			if got.Stuck != tc.wantStuck {
				t.Errorf("StuckRollout() = %+v, want Stuck=%v", got, tc.wantStuck)
			}
		})
	}
}
