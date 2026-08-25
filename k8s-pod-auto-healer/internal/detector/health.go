// Package detector decides whether a Pod is unhealthy enough to remediate.
package detector

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Result describes a health check outcome. Reason is empty when the pod
// is healthy.
type Result struct {
	Unhealthy bool
	Reason    string // Failed, CrashLoopBackOff, ImagePullBackOff, OOMKilled
	Container string
}

// CheckPodHealth flags the first failure condition found. It does not
// consider restart count alone a remediation trigger — see
// HighRestartCount, which is logged separately since CrashLoopBackOff
// already covers the "keeps restarting" case.
func CheckPodHealth(pod *corev1.Pod) Result {
	if pod.Status.Phase == corev1.PodFailed {
		return Result{Unhealthy: true, Reason: "Failed"}
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			reason := cs.State.Waiting.Reason
			if strings.Contains(reason, "CrashLoopBackOff") {
				return Result{Unhealthy: true, Reason: "CrashLoopBackOff", Container: cs.Name}
			}
			if strings.Contains(reason, "ImagePullBackOff") {
				return Result{Unhealthy: true, Reason: "ImagePullBackOff", Container: cs.Name}
			}
		}

		if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
			return Result{Unhealthy: true, Reason: "OOMKilled", Container: cs.Name}
		}
	}

	return Result{}
}

// HighRestartCount reports the first container whose restart count meets
// threshold. It's informational only — it does not by itself mark the
// pod unhealthy.
func HighRestartCount(pod *corev1.Pod, threshold int32) (bool, string, int32) {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.RestartCount >= threshold {
			return true, cs.Name, cs.RestartCount
		}
	}
	return false, "", 0
}
