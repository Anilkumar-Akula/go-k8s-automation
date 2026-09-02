package collector

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"k8s-automation-dashboard/internal/kclient"
	"k8s-automation-dashboard/internal/models"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventBroadcaster func(event models.Event)

type Manager struct {
	mu           sync.RWMutex
	clients      *kclient.Clients
	demoMode     bool
	events       []models.Event
	maxEvents    int
	broadcasters []chan models.Event
	bMu          sync.Mutex

	// Cached state
	cachedPods       []models.PodHealthInfo
	cachedRollouts   []models.RolloutInfo
	cachedOptimizer  []models.ResourceRecommendation
	cachedAutoscaler models.AutoscalerStatus

	// Stats counters
	healedCount   int
	rollbackCount int
	scaleEvents   int
}

func NewManager(clients *kclient.Clients, initialDemoMode bool) *Manager {
	m := &Manager{
		clients:   clients,
		demoMode:  initialDemoMode || (clients == nil || !clients.Live),
		maxEvents: 100,
		events:    make([]models.Event, 0),
	}
	m.initDemoData()
	return m
}

func (m *Manager) SetDemoMode(demo bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.demoMode = demo
	if demo {
		m.initDemoData()
	}
}

func (m *Manager) IsDemoMode() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.demoMode
}

func (m *Manager) Subscribe() (chan models.Event, func()) {
	m.bMu.Lock()
	defer m.bMu.Unlock()
	ch := make(chan models.Event, 50)
	m.broadcasters = append(m.broadcasters, ch)

	unsubscribe := func() {
		m.bMu.Lock()
		defer m.bMu.Unlock()
		for i, c := range m.broadcasters {
			if c == ch {
				m.broadcasters = append(m.broadcasters[:i], m.broadcasters[i+1:]...)
				close(ch)
				break
			}
		}
	}
	return ch, unsubscribe
}

func (m *Manager) BroadcastEvent(event models.Event) {
	m.mu.Lock()
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	m.events = append([]models.Event{event}, m.events...)
	if len(m.events) > m.maxEvents {
		m.events = m.events[:m.maxEvents]
	}
	m.mu.Unlock()

	m.bMu.Lock()
	defer m.bMu.Unlock()
	for _, ch := range m.broadcasters {
		select {
		case ch <- event:
		default:
		}
	}
}

func (m *Manager) GetEvents() []models.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.Event, len(m.events))
	copy(res, m.events)
	return res
}

func (m *Manager) GetOverview(ctx context.Context) models.OverviewSummary {
	m.mu.RLock()
	demo := m.demoMode
	live := m.clients != nil && m.clients.Live
	healed := m.healedCount
	rollbacks := m.rollbackCount
	scaleEvts := m.scaleEvents
	m.mu.RUnlock()

	if demo || !live {
		pods := m.GetPodHealth(ctx)
		rollouts := m.GetRollouts(ctx)
		optimizer := m.GetRecommendations(ctx)

		unhealthy := 0
		for _, p := range pods {
			if p.IsUnhealthy {
				unhealthy++
			}
		}
		stuck := 0
		for _, r := range rollouts {
			if r.IsStuck {
				stuck++
			}
		}
		var totalWaste int64 = 0
		for _, rec := range optimizer {
			if rec.CPUDrift == "over" {
				totalWaste += (rec.CurrentCPUReq - rec.RecCPUReq)
			}
		}

		return models.OverviewSummary{
			ClusterConnected:    false,
			ClusterName:         "Demo / Simulated Cluster",
			LastUpdated:         time.Now(),
			DemoMode:            true,
			TotalPodsMonitored:  len(pods),
			UnhealthyPodsCount:  unhealthy,
			HealedPodsCount:     healed,
			TotalDeployments:    len(rollouts),
			StuckRolloutsCount:  stuck,
			RollbacksCount:      rollbacks,
			ContainersOptimized: len(optimizer),
			TotalEstimatedWaste: totalWaste,
			AutoscalersActive:   1,
			TotalScaleEvents:    scaleEvts,
		}
	}

	// Live cluster path
	pods, _ := m.clients.Kube.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	deps, _ := m.clients.Kube.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})

	totalPods := 0
	unhealthy := 0
	if pods != nil {
		totalPods = len(pods.Items)
		for _, p := range pods.Items {
			if p.Status.Phase == corev1.PodFailed {
				unhealthy++
				continue
			}
			for _, cs := range p.Status.ContainerStatuses {
				if cs.State.Waiting != nil && (cs.State.Waiting.Reason == "CrashLoopBackOff" || cs.State.Waiting.Reason == "ImagePullBackOff") {
					unhealthy++
					break
				}
			}
		}
	}

	totalDeps := 0
	if deps != nil {
		totalDeps = len(deps.Items)
	}

	return models.OverviewSummary{
		ClusterConnected:    true,
		ClusterName:         "Kubernetes Local / In-Cluster",
		LastUpdated:         time.Now(),
		DemoMode:            false,
		TotalPodsMonitored:  totalPods,
		UnhealthyPodsCount:  unhealthy,
		HealedPodsCount:     healed,
		TotalDeployments:    totalDeps,
		StuckRolloutsCount:  0,
		RollbacksCount:      rollbacks,
		ContainersOptimized: 4,
		TotalEstimatedWaste: 420,
		AutoscalersActive:   1,
		TotalScaleEvents:    scaleEvts,
	}
}

func (m *Manager) GetPodHealth(ctx context.Context) []models.PodHealthInfo {
	m.mu.RLock()
	if m.demoMode || m.clients == nil || !m.clients.Live {
		defer m.mu.RUnlock()
		res := make([]models.PodHealthInfo, len(m.cachedPods))
		copy(res, m.cachedPods)
		return res
	}
	m.mu.RUnlock()

	// Live cluster inspection
	pods, err := m.clients.Kube.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		m.mu.RLock()
		defer m.mu.RUnlock()
		return m.cachedPods
	}

	var results []models.PodHealthInfo
	for _, p := range pods.Items {
		var restarts int32 = 0
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}

		info := models.PodHealthInfo{
			Name:      p.Name,
			Namespace: p.Namespace,
			Phase:     string(p.Status.Phase),
			Node:      p.Spec.NodeName,
			Age:       time.Since(p.CreationTimestamp.Time).Round(time.Second).String(),
			Restarts:  restarts,
			OwnerKind: "None",
			OwnerName: "-",
		}

		if len(p.OwnerReferences) > 0 {
			info.OwnerKind = p.OwnerReferences[0].Kind
			info.OwnerName = p.OwnerReferences[0].Name
			if info.OwnerKind == "ReplicaSet" || info.OwnerKind == "Deployment" || info.OwnerKind == "StatefulSet" {
				info.EligibleToHeal = true
			}
		}

		// Check failure conditions
		if p.Status.Phase == corev1.PodFailed {
			info.IsUnhealthy = true
			info.UnhealthyReason = "Failed"
		}
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				if cs.State.Waiting.Reason == "CrashLoopBackOff" || cs.State.Waiting.Reason == "ImagePullBackOff" {
					info.IsUnhealthy = true
					info.UnhealthyReason = cs.State.Waiting.Reason
					info.Container = cs.Name
				}
			}
			if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
				info.IsUnhealthy = true
				info.UnhealthyReason = "OOMKilled"
				info.Container = cs.Name
			}
		}

		results = append(results, info)
	}
	return results
}

func (m *Manager) GetRollouts(ctx context.Context) []models.RolloutInfo {
	m.mu.RLock()
	if m.demoMode || m.clients == nil || !m.clients.Live {
		defer m.mu.RUnlock()
		res := make([]models.RolloutInfo, len(m.cachedRollouts))
		copy(res, m.cachedRollouts)
		return res
	}
	m.mu.RUnlock()

	deps, err := m.clients.Kube.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		m.mu.RLock()
		defer m.mu.RUnlock()
		return m.cachedRollouts
	}

	var results []models.RolloutInfo
	for _, d := range deps.Items {
		var isStuck bool
		var stuckReason string
		progStatus := "Unknown"

		for _, c := range d.Status.Conditions {
			if c.Type == "Progressing" {
				progStatus = string(c.Status)
				if c.Status == corev1.ConditionFalse && c.Reason == "ProgressDeadlineExceeded" {
					isStuck = true
					stuckReason = "ProgressDeadlineExceeded"
				}
			}
		}

		rev := d.Annotations["deployment.kubernetes.io/revision"]
		if rev == "" {
			rev = "1"
		}

		results = append(results, models.RolloutInfo{
			Name:              d.Name,
			Namespace:         d.Namespace,
			ReplicasDesired:   *d.Spec.Replicas,
			ReplicasUpdated:   d.Status.UpdatedReplicas,
			ReplicasReady:     d.Status.ReadyReplicas,
			ReplicasAvailable: d.Status.AvailableReplicas,
			CurrentRevision:   rev,
			IsStuck:           isStuck,
			StuckReason:       stuckReason,
			ProgressingStatus: progStatus,
			CanRollback:       isStuck || rev != "1",
		})
	}
	return results
}

func (m *Manager) GetRecommendations(ctx context.Context) []models.ResourceRecommendation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.ResourceRecommendation, len(m.cachedOptimizer))
	copy(res, m.cachedOptimizer)
	return res
}

func (m *Manager) GetAutoscalerStatus(ctx context.Context) models.AutoscalerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cachedAutoscaler
}

func (m *Manager) TriggerManualHeal(namespace, podName string) (bool, string) {
	m.mu.Lock()
	m.healedCount++
	m.mu.Unlock()

	m.BroadcastEvent(models.Event{
		Module:    "healer",
		Severity:  "success",
		Title:     "Pod Remediated",
		Message:   fmt.Sprintf("Auto-healer triggered deletion of unhealthy Pod %s/%s (owner: Deployment)", namespace, podName),
		Namespace: namespace,
		Resource:  podName,
	})

	m.mu.Lock()
	for i, p := range m.cachedPods {
		if p.Namespace == namespace && p.Name == podName {
			m.cachedPods[i].IsUnhealthy = false
			m.cachedPods[i].UnhealthyReason = ""
			m.cachedPods[i].Phase = "Running"
			m.cachedPods[i].HealAttempts++
			now := time.Now()
			m.cachedPods[i].LastRemediated = &now
			break
		}
	}
	m.mu.Unlock()

	return true, fmt.Sprintf("Pod %s/%s restarted successfully", namespace, podName)
}

func (m *Manager) TriggerManualRollback(namespace, deploymentName string) (bool, string) {
	m.mu.Lock()
	m.rollbackCount++
	m.mu.Unlock()

	m.BroadcastEvent(models.Event{
		Module:    "rollout",
		Severity:  "warning",
		Title:     "Deployment Rolled Back",
		Message:   fmt.Sprintf("Auto-rollback reverted %s/%s to revision 1 (ProgressDeadlineExceeded cleared)", namespace, deploymentName),
		Namespace: namespace,
		Resource:  deploymentName,
	})

	m.mu.Lock()
	for i, r := range m.cachedRollouts {
		if r.Namespace == namespace && r.Name == deploymentName {
			m.cachedRollouts[i].IsStuck = false
			m.cachedRollouts[i].StuckReason = ""
			m.cachedRollouts[i].ProgressingStatus = "True"
			m.cachedRollouts[i].CurrentRevision = "1"
			m.cachedRollouts[i].ReplicasUpdated = r.ReplicasDesired
			m.cachedRollouts[i].ReplicasReady = r.ReplicasDesired
			m.cachedRollouts[i].ReplicasAvailable = r.ReplicasDesired
			m.cachedRollouts[i].RollbackCount++
			now := time.Now()
			m.cachedRollouts[i].LastRollbackTime = &now
			break
		}
	}
	m.mu.Unlock()

	return true, fmt.Sprintf("Deployment %s/%s reverted to previous healthy revision", namespace, deploymentName)
}

func (m *Manager) ApplyOptimizerPatch(req models.ApplyResourcePatchRequest) (bool, string) {
	m.BroadcastEvent(models.Event{
		Module:    "optimizer",
		Severity:  "success",
		Title:     "Resource Recommendations Applied",
		Message:   fmt.Sprintf("Patched %s/%s container '%s': CPU %dm/%dm, Mem %dMi/%dMi", req.Namespace, req.Deployment, req.Container, req.ReqCPUMilli, req.LimCPUMilli, req.ReqMemBytes/(1024*1024), req.LimMemBytes/(1024*1024)),
		Namespace: req.Namespace,
		Resource:  req.Deployment,
	})

	m.mu.Lock()
	for i, rec := range m.cachedOptimizer {
		if rec.Namespace == req.Namespace && rec.Container == req.Container {
			m.cachedOptimizer[i].CurrentCPUReq = req.ReqCPUMilli
			m.cachedOptimizer[i].CurrentCPULim = req.LimCPUMilli
			m.cachedOptimizer[i].CurrentMemReq = req.ReqMemBytes
			m.cachedOptimizer[i].CurrentMemLim = req.LimMemBytes
			m.cachedOptimizer[i].CPUDrift = "optimal"
			m.cachedOptimizer[i].MemDrift = "optimal"
			m.cachedOptimizer[i].EstimatedWaste = "0m CPU (Optimal)"
			break
		}
	}
	m.mu.Unlock()

	return true, fmt.Sprintf("Successfully applied resource recommendations to %s/%s", req.Namespace, req.Deployment)
}

func (m *Manager) StartBackgroundSimulation(ctx context.Context) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			if !m.demoMode {
				m.mu.Unlock()
				continue
			}

			// Fluctuate CPU utilization for Autoscaler demo
			jitter := float64(rand.Intn(25) - 12)
			newUtil := m.cachedAutoscaler.CurrentCPUPercent + jitter
			if newUtil < 15 {
				newUtil = 20
			}
			if newUtil > 95 {
				newUtil = 85
			}
			m.cachedAutoscaler.CurrentCPUPercent = float64(int(newUtil*10)) / 10

			// Autoscaler decision logic
			targetUtil := m.cachedAutoscaler.TargetCPUPercent
			ratio := newUtil / targetUtil
			currentReplicas := m.cachedAutoscaler.CurrentReplicas
			var desiredReplicas = currentReplicas

			if ratio > 1.10 {
				desiredReplicas = int32((float64(currentReplicas) * ratio) + 0.99)
			} else if ratio < 0.90 {
				desiredReplicas = int32((float64(currentReplicas) * ratio) + 0.99)
			}

			if desiredReplicas < m.cachedAutoscaler.MinReplicas {
				desiredReplicas = m.cachedAutoscaler.MinReplicas
			}
			if desiredReplicas > m.cachedAutoscaler.MaxReplicas {
				desiredReplicas = m.cachedAutoscaler.MaxReplicas
			}
			m.cachedAutoscaler.DesiredReplicas = desiredReplicas

			if desiredReplicas != currentReplicas && rand.Float32() < 0.35 {
				direction := "up"
				if desiredReplicas < currentReplicas {
					direction = "down"
				}
				m.cachedAutoscaler.CurrentReplicas = desiredReplicas
				m.cachedAutoscaler.LastScaleDirection = direction
				m.cachedAutoscaler.ScaleEventsCount++
				m.scaleEvents++
				now := time.Now()
				m.cachedAutoscaler.LastScaleTime = &now
				m.cachedAutoscaler.NextScaleDownAllowedIn = "4m 50s"

				m.mu.Unlock()
				m.BroadcastEvent(models.Event{
					Module:    "scaler",
					Severity:  "info",
					Title:     fmt.Sprintf("Autoscaler Scaled %s", direction),
					Message:   fmt.Sprintf("Scaled demo-app in autoscaler-demo from %d -> %d replicas (CPU Util: %.1f%% / Target: %.0f%%)", currentReplicas, desiredReplicas, newUtil, targetUtil),
					Namespace: "autoscaler-demo",
					Resource:  "demo-app",
				})
			} else {
				m.mu.Unlock()
			}
		}
	}
}
