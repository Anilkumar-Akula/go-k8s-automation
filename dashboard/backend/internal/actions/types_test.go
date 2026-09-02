package actions

import "testing"

func TestRestartRequestValidate(t *testing.T) {
	cases := []struct {
		name string
		req  RestartRequest
		ok   bool
	}{
		{"valid", RestartRequest{Namespace: "ns", Pod: "p", Reason: "flaky"}, true},
		{"missing namespace", RestartRequest{Pod: "p", Reason: "flaky"}, false},
		{"missing pod", RestartRequest{Namespace: "ns", Reason: "flaky"}, false},
		{"missing reason", RestartRequest{Namespace: "ns", Pod: "p"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err == nil) != tc.ok {
				t.Errorf("Validate() error = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}

func TestRollbackRequestValidate(t *testing.T) {
	valid := RollbackRequest{Namespace: "ns", Deployment: "d", Reason: "stuck"}
	if err := valid.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
	if err := (RollbackRequest{Deployment: "d", Reason: "stuck"}).Validate(); err == nil {
		t.Error("expected error for missing namespace")
	}
}

func TestApplyRequestValidate(t *testing.T) {
	valid := ApplyRequest{Namespace: "ns", Pod: "p", Container: "c", Reason: "drift"}
	if err := valid.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
	if err := (ApplyRequest{Namespace: "ns", Pod: "p", Reason: "drift"}).Validate(); err == nil {
		t.Error("expected error for missing container")
	}
}

func TestScaleRequestValidate(t *testing.T) {
	cases := []struct {
		name string
		req  ScaleRequest
		max  int32
		ok   bool
	}{
		{"valid", ScaleRequest{Namespace: "ns", Deployment: "d", Replicas: 5, Reason: "r"}, 20, true},
		{"zero replicas", ScaleRequest{Namespace: "ns", Deployment: "d", Replicas: 0, Reason: "r"}, 20, false},
		{"negative replicas", ScaleRequest{Namespace: "ns", Deployment: "d", Replicas: -1, Reason: "r"}, 20, false},
		{"over max", ScaleRequest{Namespace: "ns", Deployment: "d", Replicas: 21, Reason: "r"}, 20, false},
		{"at max", ScaleRequest{Namespace: "ns", Deployment: "d", Replicas: 20, Reason: "r"}, 20, true},
		{"missing reason", ScaleRequest{Namespace: "ns", Deployment: "d", Replicas: 5}, 20, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate(tc.max)
			if (err == nil) != tc.ok {
				t.Errorf("Validate(%d) error = %v, want ok=%v", tc.max, err, tc.ok)
			}
		})
	}
}
