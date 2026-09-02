package kubernetes

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s-automation-dashboard-backend/internal/models"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

type ClusterClient struct {
	mu          sync.RWMutex
	Kube        kubernetes.Interface
	Metrics     metricsv1beta1.Interface
	Live        bool
	ClusterName string

	// Simulated state when running without active K8s cluster
	simPods        []models.HealerPodInfo
	simRollouts    []models.RolloutDeploymentInfo
	simOptimizer   []models.OptimizerRecommendation
	simScalers     []models.ScalerTargetInfo
}

func NewClusterClient() *ClusterClient {
	c := &ClusterClient{
		Live:        false,
		ClusterName: "Simulation Environment",
	}

	// Try in-cluster first
	cfg, err := rest.InClusterConfig()
	if err == nil {
		kube, kErr := kubernetes.NewForConfig(cfg)
		metrics, _ := metricsv1beta1.NewForConfig(cfg)
		if kErr == nil {
			c.Kube = kube
			c.Metrics = metrics
			c.Live = true
			c.ClusterName = "In-Cluster Kubernetes"
			return c
		}
	}

	// Fallback to kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	if _, err := os.Stat(kubeconfig); err == nil {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			kube, kErr := kubernetes.NewForConfig(cfg)
			metrics, _ := metricsv1beta1.NewForConfig(cfg)
			if kErr == nil {
				c.Kube = kube
				c.Metrics = metrics
				c.Live = true
				c.ClusterName = "Local Kubeconfig Cluster"
				return c
			}
		}
	}

	c.initSimulationData()
	return c
}

func (c *ClusterClient) GetClusterHealth(ctx context.Context) models.ClusterHealth {
	c.mu.RLock()
	live := c.Live
	name := c.ClusterName
	c.mu.RUnlock()

	if !live || c.Kube == nil {
		return models.ClusterHealth{
			Status:           "Healthy",
			TotalNodes:       3,
			ReadyNodes:       3,
			TotalPods:        18,
			RunningPods:      14,
			PendingPods:      1,
			FailedPods:       3,
			TotalDeployments: 6,
			ClusterConnected: false,
			ClusterName:      name,
			K8sVersion:       "v1.31.0-simulated",
			LastUpdated:      time.Now(),
		}
	}

	nodes, _ := c.Kube.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	pods, _ := c.Kube.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	deps, _ := c.Kube.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})

	totalNodes, readyNodes := 0, 0
	if nodes != nil {
		totalNodes = len(nodes.Items)
		for _, n := range nodes.Items {
			for _, cond := range n.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					readyNodes++
				}
			}
		}
	}

	totalPods, running, pending, failed := 0, 0, 0, 0
	if pods != nil {
		totalPods = len(pods.Items)
		for _, p := range pods.Items {
			switch p.Status.Phase {
			case corev1.PodRunning:
				running++
			case corev1.PodPending:
				pending++
			case corev1.PodFailed:
				failed++
			}
		}
	}

	totalDeps := 0
	if deps != nil {
		totalDeps = len(deps.Items)
	}

	status := "Healthy"
	if failed > 2 || readyNodes < totalNodes {
		status = "Warning"
	}

	return models.ClusterHealth{
		Status:           status,
		TotalNodes:       totalNodes,
		ReadyNodes:       readyNodes,
		TotalPods:        totalPods,
		RunningPods:      running,
		PendingPods:      pending,
		FailedPods:       failed,
		TotalDeployments: totalDeps,
		ClusterConnected: true,
		ClusterName:      name,
		K8sVersion:       "v1.31.0",
		LastUpdated:      time.Now(),
	}
}

func (c *ClusterClient) GetPods(ctx context.Context) []models.HealerPodInfo {
	c.mu.RLock()
	if !c.Live || c.Kube == nil {
		defer c.mu.RUnlock()
		res := make([]models.HealerPodInfo, len(c.simPods))
		copy(res, c.simPods)
		return res
	}
	c.mu.RUnlock()

	pods, err := c.Kube.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.simPods
	}

	var results []models.HealerPodInfo
	for _, p := range pods.Items {
		var restarts int32 = 0
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}

		info := models.HealerPodInfo{
			Name:         p.Name,
			Namespace:    p.Namespace,
			Phase:        string(p.Status.Phase),
			Node:         p.Spec.NodeName,
			Age:          time.Since(p.CreationTimestamp.Time).Round(time.Second).String(),
			RestartCount: restarts,
			OwnerKind:    "None",
			OwnerName:    "-",
		}

		if len(p.OwnerReferences) > 0 {
			info.OwnerKind = p.OwnerReferences[0].Kind
			info.OwnerName = p.OwnerReferences[0].Name
			if info.OwnerKind == "ReplicaSet" || info.OwnerKind == "Deployment" || info.OwnerKind == "StatefulSet" {
				info.EligibleToHeal = true
			}
		}

		if p.Status.Phase == corev1.PodFailed {
			info.IsUnhealthy = true
			info.UnhealthyReason = "Failed"
		}
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil && (cs.State.Waiting.Reason == "CrashLoopBackOff" || cs.State.Waiting.Reason == "ImagePullBackOff") {
				info.IsUnhealthy = true
				info.UnhealthyReason = cs.State.Waiting.Reason
				info.Container = cs.Name
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

func (c *ClusterClient) GetRollouts(ctx context.Context) []models.RolloutDeploymentInfo {
	c.mu.RLock()
	if !c.Live || c.Kube == nil {
		defer c.mu.RUnlock()
		res := make([]models.RolloutDeploymentInfo, len(c.simRollouts))
		copy(res, c.simRollouts)
		return res
	}
	c.mu.RUnlock()

	deps, err := c.Kube.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.simRollouts
	}

	var results []models.RolloutDeploymentInfo
	for _, d := range deps.Items {
		isStuck := false
		stuckReason := ""
		progStatus := "Unknown"

		for _, cond := range d.Status.Conditions {
			if cond.Type == "Progressing" {
				progStatus = string(cond.Status)
				if cond.Status == corev1.ConditionFalse && cond.Reason == "ProgressDeadlineExceeded" {
					isStuck = true
					stuckReason = "ProgressDeadlineExceeded"
				}
			}
		}

		rev := d.Annotations["deployment.kubernetes.io/revision"]
		if rev == "" {
			rev = "1"
		}

		recoveryStatus := "Healthy"
		if isStuck {
			recoveryStatus = "Stuck"
		}

		results = append(results, models.RolloutDeploymentInfo{
			Name:              d.Name,
			Namespace:         d.Namespace,
			ReplicasDesired:   *d.Spec.Replicas,
			ReplicasUpdated:   d.Status.UpdatedReplicas,
			ReplicasReady:     d.Status.ReadyReplicas,
			ReplicasAvailable: d.Status.AvailableReplicas,
			CurrentRevision:   rev,
			PreviousRevision:  "1",
			IsStuck:           isStuck,
			StuckReason:       stuckReason,
			ProgressingStatus: progStatus,
			RecoveryStatus:    recoveryStatus,
			CanRollback:       isStuck || rev != "1",
		})
	}
	return results
}

func (c *ClusterClient) GetOptimizerRecommendations(ctx context.Context) []models.OptimizerRecommendation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]models.OptimizerRecommendation, len(c.simOptimizer))
	copy(res, c.simOptimizer)
	return res
}

func (c *ClusterClient) GetScalers(ctx context.Context) []models.ScalerTargetInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]models.ScalerTargetInfo, len(c.simScalers))
	copy(res, c.simScalers)
	return res
}

func (c *ClusterClient) RemediatePod(ctx context.Context, namespace, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, p := range c.simPods {
		if p.Namespace == namespace && p.Name == name {
			c.simPods[i].IsUnhealthy = false
			c.simPods[i].UnhealthyReason = ""
			c.simPods[i].Phase = "Running"
			c.simPods[i].HealAttempts++
			now := time.Now()
			c.simPods[i].LastRemediated = &now
			return nil
		}
	}
	if c.Live && c.Kube != nil {
		return c.Kube.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}
	return nil
}

func (c *ClusterClient) RollbackDeployment(ctx context.Context, namespace, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, r := range c.simRollouts {
		if r.Namespace == namespace && r.Name == name {
			c.simRollouts[i].IsStuck = false
			c.simRollouts[i].StuckReason = ""
			c.simRollouts[i].ProgressingStatus = "True"
			c.simRollouts[i].RecoveryStatus = "Healthy"
			c.simRollouts[i].CurrentRevision = "1"
			c.simRollouts[i].ReplicasUpdated = r.ReplicasDesired
			c.simRollouts[i].ReplicasReady = r.ReplicasDesired
			c.simRollouts[i].ReplicasAvailable = r.ReplicasDesired
			c.simRollouts[i].RollbackCount++
			now := time.Now()
			c.simRollouts[i].LastRollbackTime = &now
			return nil
		}
	}
	return nil
}

func (c *ClusterClient) ApplyOptimizerPatch(ctx context.Context, namespace, deployment, container string, reqCPU, limCPU, reqMem, limMem int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, rec := range c.simOptimizer {
		if rec.Namespace == namespace && rec.Container == container {
			c.simOptimizer[i].CurrentCPUReq = reqCPU
			c.simOptimizer[i].CurrentCPULim = limCPU
			c.simOptimizer[i].CurrentMemReq = reqMem
			c.simOptimizer[i].CurrentMemLim = limMem
			c.simOptimizer[i].CPUDrift = "optimal"
			c.simOptimizer[i].MemDrift = "optimal"
			c.simOptimizer[i].EstimatedWaste = "0m CPU (Optimal Sizing)"
			return nil
		}
	}
	return nil
}

func (c *ClusterClient) ScaleDeployment(ctx context.Context, namespace, deployment string, replicas int32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, s := range c.simScalers {
		if s.Namespace == namespace && s.Deployment == deployment {
			c.simScalers[i].CurrentReplicas = replicas
			c.simScalers[i].DesiredReplicas = replicas
			c.simScalers[i].ScaleEventsCount++
			now := time.Now()
			c.simScalers[i].LastScaleTime = &now
			return nil
		}
	}
	return nil
}

func (c *ClusterClient) initSimulationData() {
	now := time.Now()
	twoMinAgo := now.Add(-2 * time.Minute)

	c.simPods = []models.HealerPodInfo{
		{
			Name:           "payment-service-67b4f8d-x9q21",
			Namespace:      "production",
			Phase:          "Running",
			Node:           "node-01",
			Age:            "4d 12h",
			IsUnhealthy:    false,
			RestartCount:   0,
			OwnerKind:      "Deployment",
			OwnerName:      "payment-service",
			EligibleToHeal: true,
		},
		{
			Name:            "auth-gateway-55d8c7-4k8p2",
			Namespace:       "production",
			Phase:           "Running",
			Node:            "node-02",
			Age:             "18m",
			IsUnhealthy:     true,
			UnhealthyReason: "CrashLoopBackOff",
			Container:       "auth-proxy",
			RestartCount:    7,
			OwnerKind:       "Deployment",
			OwnerName:       "auth-gateway",
			EligibleToHeal:  true,
			HealAttempts:    2,
			NextRetryIn:     "45s",
			LastRemediated:  &twoMinAgo,
		},
		{
			Name:            "analytics-worker-99ac2-z8t9q",
			Namespace:       "data-pipeline",
			Phase:           "Running",
			Node:            "node-03",
			Age:             "42m",
			IsUnhealthy:     true,
			UnhealthyReason: "OOMKilled",
			Container:       "spark-driver",
			RestartCount:    4,
			OwnerKind:       "Deployment",
			OwnerName:       "analytics-worker",
			EligibleToHeal:  true,
			HealAttempts:    1,
			NextRetryIn:     "20s",
		},
	}

	c.simRollouts = []models.RolloutDeploymentInfo{
		{
			Name:              "order-api",
			Namespace:         "production",
			ReplicasDesired:   5,
			ReplicasUpdated:   5,
			ReplicasReady:     5,
			ReplicasAvailable: 5,
			CurrentRevision:   "4",
			PreviousRevision:  "3",
			IsStuck:           false,
			ProgressingStatus: "True",
			RecoveryStatus:    "Healthy",
			CanRollback:       true,
		},
		{
			Name:              "checkout-v2",
			Namespace:         "production",
			ReplicasDesired:   6,
			ReplicasUpdated:   2,
			ReplicasReady:     2,
			ReplicasAvailable: 2,
			CurrentRevision:   "5 (broken)",
			PreviousRevision:  "4",
			IsStuck:           true,
			StuckReason:       "ProgressDeadlineExceeded",
			ProgressingStatus: "False",
			RecoveryStatus:    "Stuck",
			RollbackCount:     1,
			CanRollback:       true,
		},
	}

	c.simOptimizer = []models.OptimizerRecommendation{
		{
			Key:            "production/auth-gateway/auth-proxy",
			Namespace:      "production",
			Pod:            "auth-gateway",
			Container:      "auth-proxy",
			SamplesCount:   120,
			CurrentCPUReq:  1000,
			CurrentCPULim:  2000,
			CurrentMemReq:  1024 * 1024 * 1024,
			CurrentMemLim:  2048 * 1024 * 1024,
			RecCPUReq:      250,
			RecCPULim:      600,
			RecMemReq:      384 * 1024 * 1024,
			RecMemLim:      768 * 1024 * 1024,
			CPUDrift:       "over",
			MemDrift:       "over",
			EstimatedWaste: "750m CPU / 640MiB RAM (Over-provisioned)",
			UsageP50CPU:    210,
			UsageP90CPU:    540,
			UsageP50Mem:    340 * 1024 * 1024,
			UsageP90Mem:    680 * 1024 * 1024,
		},
		{
			Key:            "data-pipeline/analytics-worker/spark-driver",
			Namespace:      "data-pipeline",
			Pod:            "analytics-worker",
			Container:      "spark-driver",
			SamplesCount:   95,
			CurrentCPUReq:  500,
			CurrentCPULim:  1000,
			CurrentMemReq:  512 * 1024 * 1024,
			CurrentMemLim:  1024 * 1024 * 1024,
			RecCPUReq:      900,
			RecCPULim:      1800,
			RecMemReq:      1536 * 1024 * 1024,
			RecMemLim:      3072 * 1024 * 1024,
			CPUDrift:       "under",
			MemDrift:       "under",
			EstimatedWaste: "Throttling Risk (Under-provisioned)",
			UsageP50CPU:    820,
			UsageP90CPU:    1620,
			UsageP50Mem:    1400 * 1024 * 1024,
			UsageP90Mem:    2800 * 1024 * 1024,
		},
	}

	c.simScalers = []models.ScalerTargetInfo{
		{
			Namespace:              "autoscaler-demo",
			Deployment:             "demo-app",
			CurrentReplicas:        4,
			DesiredReplicas:        6,
			MinReplicas:            1,
			MaxReplicas:            10,
			TargetCPUPercent:       50.0,
			CurrentCPUPercent:      72.4,
			Tolerance:              0.10,
			ScaleUpCooldownSec:     60,
			ScaleDownCooldownSec:   300,
			CooldownActive:         true,
			NextScaleDownAllowedIn: "3m 42s",
			LastScaleDirection:     "up",
			LastScaleTime:          &twoMinAgo,
			ScaleEventsCount:       28,
		},
	}
}
