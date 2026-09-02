// Package poller periodically scrapes each controller and the cluster,
// caches the latest snapshot of each, and diffs consecutive snapshots
// into human-readable events for the Live Events feed.
package poller

import (
	"fmt"

	"dashboard-api/internal/controllers"
	"dashboard-api/internal/events"
)

// The diff functions below are pure (no I/O, no clock reads) so they're
// unit-testable directly: given a previous and current snapshot, what
// happened in between? The poller fills in Event.Time before storing.

func diffHealerEvents(ns string, prev, curr controllers.HealerSnapshot) []events.Event {
	var out []events.Event
	if d := curr.RemediationTotal - prev.RemediationTotal; d > 0 {
		out = append(out, events.Event{Source: "auto-healer", Target: ns, Message: fmt.Sprintf("%.0f pod(s) remediated", d)})
	}
	if d := curr.RemediationFailures - prev.RemediationFailures; d > 0 {
		out = append(out, events.Event{Source: "auto-healer", Target: ns, Message: fmt.Sprintf("%.0f remediation attempt(s) failed", d)})
	}
	for reason, currCount := range curr.UnhealthyByReason {
		if d := currCount - prev.UnhealthyByReason[reason]; d > 0 {
			out = append(out, events.Event{Source: "auto-healer", Target: ns, Message: fmt.Sprintf("unhealthy pod detected: %s (x%.0f)", reason, d)})
		}
	}
	return out
}

func diffRolloutEvents(ns string, prev, curr controllers.RolloutSnapshot) []events.Event {
	var out []events.Event
	if d := curr.RollbackTotal - prev.RollbackTotal; d > 0 {
		out = append(out, events.Event{Source: "rollout-manager", Target: ns, Message: fmt.Sprintf("%.0f rollback(s) completed", d)})
	}
	if d := curr.RollbackFailures - prev.RollbackFailures; d > 0 {
		out = append(out, events.Event{Source: "rollout-manager", Target: ns, Message: fmt.Sprintf("%.0f rollback attempt(s) failed", d)})
	}
	for reason, currCount := range curr.StuckByReason {
		if d := currCount - prev.StuckByReason[reason]; d > 0 {
			out = append(out, events.Event{Source: "rollout-manager", Target: ns, Message: fmt.Sprintf("stuck rollout detected: %s (x%.0f)", reason, d)})
		}
	}
	return out
}

func diffOptimizerEvents(prev, curr controllers.OptimizerSnapshot) []events.Event {
	prevKeys := make(map[string]bool, len(prev.Drift))
	for _, d := range prev.Drift {
		prevKeys[driftKey(d)] = true
	}
	// The underlying counter increments every poll a workload stays
	// drifted, not just on first detection — so key presence (not the
	// count delta) is what marks a new event, or every poll would emit
	// one for as long as the drift persists.
	var out []events.Event
	for _, d := range curr.Drift {
		if !prevKeys[driftKey(d)] {
			target := d.Namespace + "/" + d.Pod + "/" + d.Container
			out = append(out, events.Event{
				Source: "resource-optimizer", Target: target,
				Message: fmt.Sprintf("%s %s-provisioned (%s)", d.Resource, driftWord(d.Direction), d.Field),
			})
		}
	}
	return out
}

func driftKey(d controllers.DriftEntry) string {
	return d.Namespace + "/" + d.Pod + "/" + d.Container + "/" + d.Resource + "/" + d.Field + "/" + d.Direction
}

func driftWord(direction string) string {
	if direction == "under" {
		return "under"
	}
	return "over"
}

func diffAutoscalerEvents(ns, deployment string, prev, curr controllers.AutoscalerSnapshot) []events.Event {
	if prev.CurrentReplicas == curr.CurrentReplicas {
		return nil
	}
	target := ns + "/" + deployment
	return []events.Event{{
		Source: "autoscaler", Target: target,
		Message: fmt.Sprintf("%.0f -> %.0f replicas", prev.CurrentReplicas, curr.CurrentReplicas),
	}}
}
