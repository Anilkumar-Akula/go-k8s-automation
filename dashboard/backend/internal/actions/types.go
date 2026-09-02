// Package actions defines the dashboard's control operations: a small,
// fixed set of typed requests (never arbitrary kubectl-equivalents),
// each with its own validation. Restart/rollback/apply/scale is the
// complete list for v1 — see the project README before adding another.
package actions

import "fmt"

type RestartRequest struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Reason    string `json:"reason"`
}

func (r RestartRequest) Validate() error {
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if r.Pod == "" {
		return fmt.Errorf("pod is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

type RollbackRequest struct {
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	Reason     string `json:"reason"`
}

func (r RollbackRequest) Validate() error {
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if r.Deployment == "" {
		return fmt.Errorf("deployment is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// ApplyRequest applies the currently-cached recommendation for one
// container back to its owning Deployment. namespace/pod/container
// together identify which cached Recommendation to use — pod pins it to
// the exact one the operator saw on screen, even though the patch itself
// lands on the Deployment (all replicas share the same template).
type ApplyRequest struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Reason    string `json:"reason"`
}

func (r ApplyRequest) Validate() error {
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if r.Pod == "" {
		return fmt.Errorf("pod is required")
	}
	if r.Container == "" {
		return fmt.Errorf("container is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

type ScaleRequest struct {
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	Replicas   int32  `json:"replicas"`
	Reason     string `json:"reason"`
}

func (r ScaleRequest) Validate(maxReplicas int32) error {
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if r.Deployment == "" {
		return fmt.Errorf("deployment is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	if r.Replicas < 1 {
		return fmt.Errorf("replicas must be >= 1")
	}
	if r.Replicas > maxReplicas {
		return fmt.Errorf("replicas must be <= %d", maxReplicas)
	}
	return nil
}
