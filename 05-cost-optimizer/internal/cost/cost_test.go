package cost

import "testing"

func TestMonthlyWaste(t *testing.T) {
	rates := Rates{CPUCorePerHour: 0.03, MemGiBPerHour: 0.004}

	// 500m CPU over-requested, memory matches recommendation exactly.
	got := MonthlyWaste(1000, 500, 1<<30, 1<<30, rates)
	want := 0.5 * 0.03 * hoursPerMonth
	if got != want {
		t.Fatalf("cpu-only waste: got %v, want %v", got, want)
	}

	// Under-provisioned isn't waste.
	if got := MonthlyWaste(200, 500, 1<<30, 2<<30, rates); got != 0 {
		t.Fatalf("under-provisioned should score 0 waste, got %v", got)
	}
}
