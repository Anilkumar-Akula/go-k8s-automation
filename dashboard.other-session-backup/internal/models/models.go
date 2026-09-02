package models

import "time"

// Event represents a live automation event streamed to the dashboard UI.
type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Module    string    `json:"module"` // "healer", "rollout", "optimizer", "scaler"
	Severity  string    `json:"severity"` // "info", "warning", "success", "danger"
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Namespace string    `json:"namespace,omitempty"`
	Resource  string    `json:"resource,omitempty"`
}

// OverviewSummary holds top-level counters and health statuses across all 4 modules.
type OverviewSummary struct {
	ClusterConnected    bool      `json:"clusterConnected"`
	ClusterName         string    `json:"clusterName"`
	LastUpdated         time.Time `json:"lastUpdated"`
	DemoMode            bool      `json:"demoMode"`
	TotalPodsMonitored  int       `json:"totalPodsMonitored"`
	UnhealthyPodsCount  int       `json:"unhealthyPodsCount"`
	HealedPodsCount     int       `json:"healedPodsCount"`
	TotalDeployments    int       `json:"totalDeployments"`
	StuckRolloutsCount  int       `json:"stuckRolloutsCount"`
	RollbacksCount      int       `json:"rollbacksCount"`
	ContainersOptimized int       `json:"containersOptimized"`
	TotalEstimatedWaste int64     `json:"totalEstimatedWasteMilli"` // millicores
	AutoscalersActive   int       `json:"autoscalersActive"`
	TotalScaleEvents    int       `json:"totalScaleEvents"`
}

// PodHealthInfo represents the state of a single Pod monitored by 01-pod-auto-healer.
type PodHealthInfo struct {
	Name            string    `json:"name"`
	Namespace       string    `json:"namespace"`
	Phase           string    `json:"phase"`
	Node            string    `json:"node"`
	Age             string    `json:"age"`
	IsUnhealthy     bool      `json:"isUnhealthy"`
	UnhealthyReason string    `json:"unhealthyReason,omitempty"` // CrashLoopBackOff, ImagePullBackOff, OOMKilled, Failed
	Container       string    `json:"container,omitempty"`
	Restarts        int32     `json:"restarts"`
	OwnerKind       string    `json:"ownerKind"`       // Deployment, StatefulSet, ReplicaSet, DaemonSet, Job, None
	OwnerName       string    `json:"ownerName"`
	EligibleToHeal  bool      `json:"eligibleToHeal"`
	HealAttempts    int       `json:"healAttempts"`
	NextRetryIn     string    `json:"nextRetryIn,omitempty"`
	LastRemediated  *time.Time `json:"lastRemediated,omitempty"`
}

// RolloutInfo represents deployment rollout status for 02-rollout-manager.
type RolloutInfo struct {
	Name               string   `json:"name"`
	Namespace          string   `json:"namespace"`
	ReplicasDesired    int32    `json:"replicasDesired"`
	ReplicasUpdated    int32    `json:"replicasUpdated"`
	ReplicasReady      int32    `json:"replicasReady"`
	ReplicasAvailable  int32    `json:"replicasAvailable"`
	CurrentRevision    string   `json:"currentRevision"`
	PreviousRevision   string   `json:"previousRevision,omitempty"`
	IsStuck            bool     `json:"isStuck"`
	StuckReason        string   `json:"stuckReason,omitempty"`
	ProgressingStatus  string   `json:"progressingStatus"` // True, False, Unknown
	RollbackCount      int      `json:"rollbackCount"`
	LastRollbackTime   *time.Time `json:"lastRollbackTime,omitempty"`
	CanRollback        bool     `json:"canRollback"`
}

// ResourceRecommendation represents right-sizing suggestions for 03-resource-optimizer.
type ResourceRecommendation struct {
	Key             string  `json:"key"` // "namespace/pod/container"
	Namespace       string  `json:"namespace"`
	Pod             string  `json:"pod"`
	Container       string  `json:"container"`
	SamplesCount    int     `json:"samplesCount"`
	CurrentCPUReq   int64   `json:"currentCpuReqMilli"`
	CurrentCPULim   int64   `json:"currentCpuLimMilli"`
	CurrentMemReq   int64   `json:"currentMemReqBytes"`
	CurrentMemLim   int64   `json:"currentMemLimBytes"`
	RecCPUReq       int64   `json:"recCpuReqMilli"`
	RecCPULim       int64   `json:"recCpuLimMilli"`
	RecMemReq       int64   `json:"recMemReqBytes"`
	RecMemLim       int64   `json:"recMemLimBytes"`
	CPUDrift        string  `json:"cpuDrift"`    // "over", "under", "optimal"
	MemDrift        string  `json:"memDrift"`    // "over", "under", "optimal"
	EstimatedWaste  string  `json:"estimatedWaste"`
	UsageP50CPU     int64   `json:"usageP50CpuMilli"`
	UsageP90CPU     int64   `json:"usageP90CpuMilli"`
	UsageP50Mem     int64   `json:"usageP50MemBytes"`
	UsageP90Mem     int64   `json:"usageP90MemBytes"`
}

// AutoscalerStatus represents horizontal autoscaling metrics for 04-autoscaler.
type AutoscalerStatus struct {
	Namespace             string    `json:"namespace"`
	Deployment            string    `json:"deployment"`
	CurrentReplicas       int32     `json:"currentReplicas"`
	DesiredReplicas       int32     `json:"desiredReplicas"`
	MinReplicas           int32     `json:"minReplicas"`
	MaxReplicas           int32     `json:"maxReplicas"`
	TargetCPUPercent      float64   `json:"targetCpuPercent"`
	CurrentCPUPercent     float64   `json:"currentCpuPercent"`
	Tolerance             float64   `json:"tolerance"`
	ScaleUpCooldownSec    int       `json:"scaleUpCooldownSec"`
	ScaleDownCooldownSec  int       `json:"scaleDownCooldownSec"`
	LastScaleDirection    string    `json:"lastScaleDirection,omitempty"` // "up", "down", "none"
	LastScaleTime         *time.Time `json:"lastScaleTime,omitempty"`
	NextScaleDownAllowedIn string   `json:"nextScaleDownAllowedIn,omitempty"`
	ScaleEventsCount      int       `json:"scaleEventsCount"`
}

// ApplyResourcePatchRequest is the payload to apply recommendations to a Deployment.
type ApplyResourcePatchRequest struct {
	Namespace   string `json:"namespace"`
	Deployment  string `json:"deployment"`
	Container   string `json:"container"`
	ReqCPUMilli int64  `json:"reqCpuMilli"`
	LimCPUMilli int64  `json:"limCpuMilli"`
	ReqMemBytes int64  `json:"reqMemBytes"`
	LimMemBytes int64  `json:"limMemBytes"`
}
