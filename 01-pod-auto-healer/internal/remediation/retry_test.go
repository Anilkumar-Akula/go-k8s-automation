package remediation

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
	}
	for _, tc := range cases {
		if got := Backoff(2*time.Second, tc.attempt); got != tc.want {
			t.Errorf("Backoff(2s, %d) = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestRetrierMaxAttempts(t *testing.T) {
	r := NewRetrier(3, time.Millisecond, time.Minute)
	defer r.Stop()

	for i := 1; i <= 3; i++ {
		attempt, _, ok := r.Attempt("Deployment/ns/app")
		if !ok || attempt != i {
			t.Fatalf("attempt %d: got (attempt=%d, ok=%v), want (attempt=%d, ok=true)", i, attempt, ok, i)
		}
	}

	if _, _, ok := r.Attempt("Deployment/ns/app"); ok {
		t.Error("4th attempt should be rejected once max=3 is exceeded")
	}

	// A different workload has its own independent counter.
	if _, _, ok := r.Attempt("Deployment/ns/other"); !ok {
		t.Error("a different workload key should not be affected by another key's limit")
	}
}
