package live

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestPodUnhealthyReason(t *testing.T) {
	cases := []struct {
		name   string
		pod    *corev1.Pod
		reason string
		want   bool
	}{
		{"running healthy", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}, "", false},
		{"failed phase", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}}, "Failed", true},
		{"crash loop", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
			{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
		}}}, "CrashLoopBackOff", true},
		{"image pull backoff", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
			{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}},
		}}}, "ImagePullBackOff", true},
		{"oom killed", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
			{LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}}},
		}}}, "OOMKilled", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason, unhealthy := podUnhealthyReason(tc.pod)
			if unhealthy != tc.want || reason != tc.reason {
				t.Errorf("podUnhealthyReason() = %q, %v; want %q, %v", reason, unhealthy, tc.reason, tc.want)
			}
		})
	}
}
