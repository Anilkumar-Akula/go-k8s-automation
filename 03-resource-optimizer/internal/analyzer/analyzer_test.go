package analyzer

import "testing"

func TestRecommendUsesPercentiles(t *testing.T) {
	m := NewManager(100)
	// 89 samples at the steady 100m baseline, 11 at a 500m spike: p50
	// should track the baseline, p90 should just reach into the spike.
	for range 89 {
		m.Add("ns/pod/c", 100, 100<<20)
	}
	for range 11 {
		m.Add("ns/pod/c", 500, 500<<20)
	}

	margins := Margins{CPURequest: 1.0, CPULimit: 1.0, MemRequest: 1.0, MemLimit: 1.0}
	rec, ok := m.Recommend("ns/pod/c", margins)
	if !ok {
		t.Fatal("expected recommendation")
	}
	if rec.ReqCPUMilli != 100 {
		t.Errorf("p50 request = %d, want 100", rec.ReqCPUMilli)
	}
	if rec.LimCPUMilli != 500 {
		t.Errorf("p90 limit = %d, want 500", rec.LimCPUMilli)
	}
}

func TestRecommendMissingKey(t *testing.T) {
	m := NewManager(60)
	if _, ok := m.Recommend("missing", Margins{}); ok {
		t.Error("expected ok=false for untracked key")
	}
}

func TestRecommendLimitNeverBelowRequest(t *testing.T) {
	m := NewManager(60)
	for range 5 {
		m.Add("ns/pod/c", 100, 100<<20)
	}
	// Request margin bigger than limit margin would otherwise push the
	// request recommendation above the limit recommendation.
	margins := Margins{CPURequest: 2.0, CPULimit: 1.0, MemRequest: 2.0, MemLimit: 1.0}
	rec, ok := m.Recommend("ns/pod/c", margins)
	if !ok {
		t.Fatal("expected recommendation")
	}
	if rec.LimCPUMilli < rec.ReqCPUMilli {
		t.Errorf("limit %d < request %d", rec.LimCPUMilli, rec.ReqCPUMilli)
	}
	if rec.LimMemBytes < rec.ReqMemBytes {
		t.Errorf("mem limit %d < mem request %d", rec.LimMemBytes, rec.ReqMemBytes)
	}
}

func TestCompareDrift(t *testing.T) {
	cases := []struct {
		name                 string
		current, recommended int64
		threshold            float64
		wantOK               bool
		wantDirection        string
	}{
		{"unset request flagged under", 0, 500, 0.3, true, "under"},
		{"unset and unrecommended is fine", 0, 0, 0.3, false, ""},
		{"within threshold is fine", 100, 110, 0.3, false, ""},
		{"under-provisioned", 100, 200, 0.3, true, "under"},
		{"over-provisioned", 500, 100, 0.3, true, "over"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			drift, ok := CompareDrift("cpu", "request", c.current, c.recommended, c.threshold)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if ok && drift.Direction != c.wantDirection {
				t.Errorf("direction = %q, want %q", drift.Direction, c.wantDirection)
			}
		})
	}
}

func TestManagerGC(t *testing.T) {
	m := NewManager(60)
	m.Add("ns/pod/c", 100, 100<<20)
	if m.Tracked() != 1 {
		t.Fatalf("tracked = %d, want 1", m.Tracked())
	}
	m.GC(0) // everything is "stale" immediately with a zero TTL
	if m.Tracked() != 0 {
		t.Errorf("tracked = %d, want 0 after GC", m.Tracked())
	}
}
