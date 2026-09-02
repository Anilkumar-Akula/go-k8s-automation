package models

import "time"

// Severity level for live events and alerts
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeveritySuccess Severity = "success"
	SeverityWarning Severity = "warning"
	SeverityDanger  Severity = "danger"
)

// Module identifier for controllers
type Module string

const (
	ModuleHealer    Module = "AUTO-HEALER"
	ModuleRollout   Module = "ROLLOUT"
	ModuleOptimizer Module = "RESOURCE"
	ModuleScaler    Module = "AUTO-SCALER"
	ModuleSystem    Module = "SYSTEM"
)

// LiveEvent represents a real-time event streamed via SSE.
type LiveEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Module    Module    `json:"module"`
	Severity  Severity  `json:"severity"`
	Resource  string    `json:"resource"`
	Namespace string    `json:"namespace,omitempty"`
	Message   string    `json:"message"`
	Formatted string    `json:"formatted"` // e.g. "23:41:10 AUTO-SCALER webapp 1 → 10 replicas"
}

// AuditLogEntry represents a durable record of a remediation, rollback, right-sizing patch, or scale action.
type AuditLogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Module    Module    `json:"module"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Namespace string    `json:"namespace"`
	Actor     string    `json:"actor"` // "system-controller" or "user-admin"
	Status    string    `json:"status"` // "success", "failed", "in-progress"
	Details   string    `json:"details"`
}

// ClusterHealth summary for the Overview page
type ClusterHealth struct {
	Status              string    `json:"status"` // "Healthy", "Degraded", "Warning"
	TotalNodes          int       `json:"totalNodes"`
	ReadyNodes          int       `json:"readyNodes"`
	TotalPods           int       `json:"totalPods"`
	RunningPods         int       `json:"runningPods"`
	PendingPods         int       `json:"pendingPods"`
	FailedPods          int       `json:"failedPods"`
	TotalDeployments    int       `json:"totalDeployments"`
	ClusterConnected    bool      `json:"clusterConnected"`
	ClusterName         string    `json:"clusterName"`
	K8sVersion          string    `json:"k8sVersion"`
	LastUpdated         time.Time `json:"lastUpdated"`
}

// AutomationHealth shows the runtime status of each of the 4 controllers
type ControllerStatus struct {
	Module            Module    `json:"module"`
	Name              string    `json:"name"`
	Status            string    `json:"status"` // "Running", "Degraded", "Stopped"
	ActiveGoroutines  int       `json:"activeGoroutines"`
	TotalActionsTaken int       `json:"totalActionsTaken"`
	LastActionTime    *time.Time `json:"lastActionTime,omitempty"`
}

// OverviewResponse is the payload for GET /api/v1/overview
type OverviewResponse struct {
	Cluster       ClusterHealth      `json:"cluster"`
	Controllers   []ControllerStatus `json:"controllers"`
	RecentEvents  []LiveEvent        `json:"recentEvents"`
	ScalingStats  ScalingOverview    `json:"scalingStats"`
	WasteOverview WasteOverview      `json:"wasteOverview"`
}

type ScalingOverview struct {
	ActiveAutoscalers int `json:"activeAutoscalers"`
	TotalScaleEvents  int `json:"totalScaleEvents"`
	ScaleUpsLast24h   int `json:"scaleUpsLast24h"`
	ScaleDownsLast24h int `json:"scaleDownsLast24h"`
}

type WasteOverview struct {
	OverProvisionedPods  int   `json:"overProvisionedPods"`
	UnderProvisionedPods int   `json:"underProvisionedPods"`
	TotalWasteCpuMilli   int64 `json:"totalWasteCpuMilli"`
	TotalWasteMemBytes   int64 `json:"totalWasteMemBytes"`
}

// PodAutoHealer state & evaluations
type HealerPodInfo struct {
	Name            string     `json:"name"`
	Namespace       string     `json:"namespace"`
	Phase           string     `json:"phase"`
	Node            string     `json:"node"`
	Age             string     `json:"age"`
	IsUnhealthy     bool       `json:"isUnhealthy"`
	UnhealthyReason string     `json:"unhealthyReason,omitempty"` // CrashLoopBackOff, ImagePullBackOff, OOMKilled, Failed
	Container       string     `json:"container,omitempty"`
	RestartCount    int32      `json:"restartCount"`
	OwnerKind       string     `json:"ownerKind"`
	OwnerName       string     `json:"ownerName"`
	EligibleToHeal  bool       `json:"eligibleToHeal"`
	HealAttempts    int        `json:"healAttempts"`
	NextRetryIn     string     `json:"nextRetryIn,omitempty"`
	LastRemediated  *time.Time `json:"lastRemediated,omitempty"`
}

type HealerResponse struct {
	TotalMonitored       int             `json:"totalMonitored"`
	UnhealthyCount       int             `json:"unhealthyCount"`
	CrashLoopCount       int             `json:"crashLoopCount"`
	OOMKilledCount       int             `json:"oomKilledCount"`
	SuccessfulRecoveries int             `json:"successfulRecoveries"`
	Pods                 []HealerPodInfo `json:"pods"`
	RecentHealingEvents  []AuditLogEntry `json:"recentHealingEvents"`
}

// Rollout Manager state
type RolloutDeploymentInfo struct {
	Name              string     `json:"name"`
	Namespace         string     `json:"namespace"`
	ReplicasDesired   int32      `json:"replicasDesired"`
	ReplicasUpdated   int32      `json:"replicasUpdated"`
	ReplicasReady     int32      `json:"replicasReady"`
	ReplicasAvailable int32      `json:"replicasAvailable"`
	CurrentRevision   string     `json:"currentRevision"`
	PreviousRevision  string     `json:"previousRevision,omitempty"`
	IsStuck           bool       `json:"isStuck"`
	StuckReason       string     `json:"stuckReason,omitempty"` // ProgressDeadlineExceeded
	ProgressingStatus string     `json:"progressingStatus"`    // True, False, Unknown
	RollbackCount     int        `json:"rollbackCount"`
	LastRollbackTime  *time.Time `json:"lastRollbackTime,omitempty"`
	RecoveryStatus    string     `json:"recoveryStatus"` // "Healthy", "Rolling Back", "Stuck"
	CanRollback       bool       `json:"canRollback"`
}

type RolloutResponse struct {
	TotalDeployments int                     `json:"totalDeployments"`
	StuckDeployments int                     `json:"stuckDeployments"`
	TotalRollbacks   int                     `json:"totalRollbacks"`
	Deployments      []RolloutDeploymentInfo `json:"deployments"`
	RecentRollbacks  []AuditLogEntry         `json:"recentRollbacks"`
}

// Resource Optimizer state
type OptimizerRecommendation struct {
	Key             string `json:"key"` // namespace/pod/container
	Namespace       string `json:"namespace"`
	Pod             string `json:"pod"`
	Container       string `json:"container"`
	SamplesCount    int    `json:"samplesCount"`
	CurrentCPUReq   int64  `json:"currentCpuReqMilli"`
	CurrentCPULim   int64  `json:"currentCpuLimMilli"`
	CurrentMemReq   int64  `json:"currentMemReqBytes"`
	CurrentMemLim   int64  `json:"currentMemLimBytes"`
	RecCPUReq       int64  `json:"recCpuReqMilli"`
	RecCPULim       int64  `json:"recCpuLimMilli"`
	RecMemReq       int64  `json:"recMemReqBytes"`
	RecMemLim       int64  `json:"recMemLimBytes"`
	CPUDrift        string `json:"cpuDrift"` // "over", "under", "optimal"
	MemDrift        string `json:"memDrift"` // "over", "under", "optimal"
	EstimatedWaste  string `json:"estimatedWaste"`
	UsageP50CPU     int64  `json:"usageP50CpuMilli"`
	UsageP90CPU     int64  `json:"usageP90CpuMilli"`
	UsageP50Mem     int64  `json:"usageP50MemBytes"`
	UsageP90Mem     int64  `json:"usageP90MemBytes"`
}

type OptimizerResponse struct {
	TotalTracked        int                       `json:"totalTracked"`
	OverProvisioned     int                       `json:"overProvisioned"`
	UnderProvisioned    int                       `json:"underProvisioned"`
	OptimalCount        int                       `json:"optimalCount"`
	TotalWasteCPUMilli  int64                     `json:"totalWasteCpuMilli"`
	Recommendations     []OptimizerRecommendation `json:"recommendations"`
	RecentAppliedPatches []AuditLogEntry          `json:"recentAppliedPatches"`
}

// Auto-Scaler state
type ScalerTargetInfo struct {
	Namespace              string     `json:"namespace"`
	Deployment             string     `json:"deployment"`
	CurrentReplicas        int32      `json:"currentReplicas"`
	DesiredReplicas        int32      `json:"desiredReplicas"`
	MinReplicas            int32      `json:"minReplicas"`
	MaxReplicas            int32      `json:"maxReplicas"`
	TargetCPUPercent       float64    `json:"targetCpuPercent"`
	CurrentCPUPercent      float64    `json:"currentCpuPercent"`
	Tolerance              float64    `json:"tolerance"`
	ScaleUpCooldownSec     int        `json:"scaleUpCooldownSec"`
	ScaleDownCooldownSec   int        `json:"scaleDownCooldownSec"`
	CooldownActive         bool       `json:"cooldownActive"`
	NextScaleDownAllowedIn string     `json:"nextScaleDownAllowedIn,omitempty"`
	LastScaleDirection     string     `json:"lastScaleDirection,omitempty"`
	LastScaleTime          *time.Time `json:"lastScaleTime,omitempty"`
	ScaleEventsCount       int        `json:"scaleEventsCount"`
}

type ScalerResponse struct {
	ActiveScalers int                `json:"activeScalers"`
	TotalEvents   int                `json:"totalEvents"`
	Scalers       []ScalerTargetInfo `json:"scalers"`
	ScalingHistory []AuditLogEntry   `json:"scalingHistory"`
}
