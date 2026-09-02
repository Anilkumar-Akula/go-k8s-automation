package scaler

import "testing"

func TestDecide(t *testing.T) {
	cases := []struct {
		name                           string
		current                        int32
		currentUtil, targetUtil, toler float64
		min, max, want                 int32
	}{
		{"no data holds steady", 3, 0, 50, 0.1, 1, 10, 3},
		{"double utilization doubles replicas", 2, 100, 50, 0.1, 1, 10, 4},
		{"half utilization halves replicas", 4, 25, 50, 0.1, 1, 10, 2},
		{"within tolerance holds steady", 3, 52, 50, 0.1, 1, 10, 3},
		{"clamped to max", 5, 500, 50, 0.1, 1, 10, 10},
		{"clamped to min", 5, 1, 50, 0.1, 2, 10, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.current, tc.currentUtil, tc.targetUtil, tc.toler, tc.min, tc.max)
			if got != tc.want {
				t.Errorf("Decide(%d, %v, %v, %v, %d, %d) = %d, want %d",
					tc.current, tc.currentUtil, tc.targetUtil, tc.toler, tc.min, tc.max, got, tc.want)
			}
		})
	}
}

func TestCooldown(t *testing.T) {
	var c Cooldown
	if !c.Allow("up", 0) {
		t.Fatal("zero-value cooldown should allow the first event")
	}
	c.Record("up")
	if c.Allow("up", 1<<62) { // effectively "forever"
		t.Fatal("should not allow immediately after recording")
	}
	if !c.Allow("down", 1<<62) {
		t.Fatal("recording 'up' should not block 'down'")
	}
}
