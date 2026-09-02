package poller

import (
	"testing"

	"dashboard-api/internal/controllers"
)

func TestDiffHealerEvents(t *testing.T) {
	prev := controllers.HealerSnapshot{RemediationTotal: 1, UnhealthyByReason: map[string]float64{"CrashLoopBackOff": 1}}
	curr := controllers.HealerSnapshot{RemediationTotal: 3, RemediationFailures: 1, UnhealthyByReason: map[string]float64{"CrashLoopBackOff": 2, "OOMKilled": 1}}

	got := diffHealerEvents("auto-healer-demo", prev, curr)
	if len(got) != 4 { // remediated + failure + 2 reason deltas (CrashLoopBackOff, OOMKilled)
		t.Fatalf("expected 4 events, got %d: %+v", len(got), got)
	}
}

func TestDiffHealerEvents_NoChange(t *testing.T) {
	snap := controllers.HealerSnapshot{RemediationTotal: 5, UnhealthyByReason: map[string]float64{"OOMKilled": 2}}
	if got := diffHealerEvents("ns", snap, snap); len(got) != 0 {
		t.Fatalf("expected no events for unchanged snapshot, got %+v", got)
	}
}

func TestDiffAutoscalerEvents(t *testing.T) {
	prev := controllers.AutoscalerSnapshot{CurrentReplicas: 1}
	curr := controllers.AutoscalerSnapshot{CurrentReplicas: 10}
	got := diffAutoscalerEvents("autoscaler-demo", "demo-app", prev, curr)
	if len(got) != 1 || got[0].Message != "1 -> 10 replicas" {
		t.Fatalf("unexpected events: %+v", got)
	}

	if got := diffAutoscalerEvents("ns", "dep", curr, curr); len(got) != 0 {
		t.Fatalf("expected no event when replicas unchanged, got %+v", got)
	}
}

func TestDiffOptimizerEvents(t *testing.T) {
	prev := controllers.OptimizerSnapshot{}
	curr := controllers.OptimizerSnapshot{Drift: []controllers.DriftEntry{
		{Namespace: "ns", Pod: "p", Container: "c", Resource: "cpu", Field: "request", Direction: "under", Count: 1},
	}}
	got := diffOptimizerEvents(prev, curr)
	if len(got) != 1 || got[0].Message != "cpu under-provisioned (request)" {
		t.Fatalf("unexpected events: %+v", got)
	}

	if got := diffOptimizerEvents(curr, curr); len(got) != 0 {
		t.Fatalf("expected no event for unchanged drift counts, got %+v", got)
	}
}
