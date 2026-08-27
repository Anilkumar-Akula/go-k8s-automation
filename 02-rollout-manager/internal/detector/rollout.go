// Package detector decides whether a Deployment's rollout has stalled.
package detector

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// Result describes a rollout health check outcome. Reason is empty when
// the rollout is healthy.
type Result struct {
	Stuck  bool
	Reason string
}

// StuckRollout reports a Deployment whose rollout has stalled past its
// progressDeadlineSeconds. This mirrors exactly what `kubectl rollout
// status` waits for: the Deployment controller itself flips the
// Progressing condition to False with reason ProgressDeadlineExceeded once
// no progress (new pods becoming ready) has been observed within the
// deadline — no need to reimplement that timing logic here.
func StuckRollout(dep *appsv1.Deployment) Result {
	for _, c := range dep.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing &&
			c.Status == corev1.ConditionFalse &&
			c.Reason == "ProgressDeadlineExceeded" {
			return Result{Stuck: true, Reason: c.Reason}
		}
	}
	return Result{}
}
