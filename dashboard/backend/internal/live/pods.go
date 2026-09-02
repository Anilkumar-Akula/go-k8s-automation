// Package live answers "which specific pod/deployment is unhealthy right
// now" — the aggregate controller snapshots (internal/controllers) only
// carry counts by reason, not names, so a Restart/Rollback confirmation
// dialog needs somewhere to get an actual target from. Reuses the exact
// detection conditions projects 01 and 02 already use, read-only.
package live

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type UnhealthyPod struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Reason    string `json:"reason"`
}

// ListUnhealthyPods mirrors project 01's detector.CheckPodHealth.
func ListUnhealthyPods(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]UnhealthyPod, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []UnhealthyPod
	for _, pod := range pods.Items {
		if reason, unhealthy := podUnhealthyReason(&pod); unhealthy {
			out = append(out, UnhealthyPod{Namespace: pod.Namespace, Pod: pod.Name, Reason: reason})
		}
	}
	return out, nil
}

func podUnhealthyReason(pod *corev1.Pod) (string, bool) {
	if pod.Status.Phase == corev1.PodFailed {
		return "Failed", true
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			reason := cs.State.Waiting.Reason
			if strings.Contains(reason, "CrashLoopBackOff") {
				return "CrashLoopBackOff", true
			}
			if strings.Contains(reason, "ImagePullBackOff") {
				return "ImagePullBackOff", true
			}
		}
		if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
			return "OOMKilled", true
		}
	}
	return "", false
}
