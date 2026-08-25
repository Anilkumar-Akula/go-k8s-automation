package detector

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestCheckPodHealth(t *testing.T) {
	cases := []struct {
		name   string
		pod    *corev1.Pod
		reason string
	}{
		{
			name:   "healthy",
			pod:    &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}},
			reason: "",
		},
		{
			name:   "failed phase",
			pod:    &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			reason: "Failed",
		},
		{
			name: "crash loop",
			pod: &corev1.Pod{Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:  "app",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
				}},
			}},
			reason: "CrashLoopBackOff",
		},
		{
			name: "image pull backoff",
			pod: &corev1.Pod{Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:  "app",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
				}},
			}},
			reason: "ImagePullBackOff",
		},
		{
			name: "oom killed",
			pod: &corev1.Pod{Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name: "app",
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"},
					},
				}},
			}},
			reason: "OOMKilled",
		},
		{
			name: "high restarts alone is not unhealthy",
			pod: &corev1.Pod{Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{Name: "app", RestartCount: 10}},
			}},
			reason: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckPodHealth(tc.pod)
			if got.Reason != tc.reason {
				t.Errorf("reason = %q, want %q", got.Reason, tc.reason)
			}
			if got.Unhealthy != (tc.reason != "") {
				t.Errorf("unhealthy = %v, want %v", got.Unhealthy, tc.reason != "")
			}
		})
	}
}

func TestHighRestartCount(t *testing.T) {
	pod := &corev1.Pod{Status: corev1.PodStatus{
		ContainerStatuses: []corev1.ContainerStatus{{Name: "app", RestartCount: 5}},
	}}

	hi, container, count := HighRestartCount(pod, 3)
	if !hi || container != "app" || count != 5 {
		t.Errorf("got (%v, %q, %d), want (true, \"app\", 5)", hi, container, count)
	}

	if hi, _, _ := HighRestartCount(pod, 6); hi {
		t.Error("threshold above restart count should not trigger")
	}
}
