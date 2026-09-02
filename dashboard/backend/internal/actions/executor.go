// Package actions' Executor performs the actual Kubernetes mutation for
// each control action. It never accepts arbitrary resource specs — each
// method's signature is as narrow as the operation it performs.
package actions

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type Executor struct {
	clientset kubernetes.Interface
}

func NewExecutor(clientset kubernetes.Interface) *Executor {
	return &Executor{clientset: clientset}
}

// RestartPod deletes the pod; its owning controller (Deployment,
// StatefulSet, ...) recreates it — the same mechanism project 01's
// auto-healer uses.
func (e *Executor) RestartPod(ctx context.Context, namespace, pod string) error {
	return e.clientset.CoreV1().Pods(namespace).Delete(ctx, pod, metav1.DeleteOptions{})
}

const revisionAnnotation = "deployment.kubernetes.io/revision"

// RollbackDeployment restores deployment's pod template to the
// ReplicaSet from the revision immediately before its current one — the
// same revision history `kubectl rollout undo` reads, reused here
// instead of tracking our own (see project 02).
func (e *Executor) RollbackDeployment(ctx context.Context, namespace, deployment string) (oldValue, newValue string, err error) {
	dep, err := e.clientset.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("get deployment: %w", err)
	}

	previous, err := previousReplicaSet(ctx, e.clientset, dep)
	if err != nil {
		return "", "", err
	}
	if previous == nil {
		return "", "", fmt.Errorf("no previous revision to roll back to")
	}

	oldValue = fmt.Sprintf("revision %s", dep.Annotations[revisionAnnotation])
	newValue = fmt.Sprintf("revision %s (replicaset %s)", previous.Annotations[revisionAnnotation], previous.Name)

	dep.Spec.Template = previous.Spec.Template
	if _, err := e.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{}); err != nil {
		return "", "", fmt.Errorf("update deployment: %w", err)
	}
	return oldValue, newValue, nil
}

func previousReplicaSet(ctx context.Context, clientset kubernetes.Interface, dep *appsv1.Deployment) (*appsv1.ReplicaSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("build selector: %w", err)
	}
	rsList, err := clientset.AppsV1().ReplicaSets(dep.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, fmt.Errorf("list replicasets: %w", err)
	}

	var owned []appsv1.ReplicaSet
	for _, rs := range rsList.Items {
		if ownedByDeployment(rs.OwnerReferences, dep.UID) {
			owned = append(owned, rs)
		}
	}
	sort.Slice(owned, func(i, j int) bool {
		return revisionOf(owned[i].Annotations) > revisionOf(owned[j].Annotations)
	})

	currentRev := revisionOf(dep.Annotations)
	for i := range owned {
		if rev := revisionOf(owned[i].Annotations); rev != 0 && rev < currentRev {
			return &owned[i], nil
		}
	}
	return nil, nil
}

func revisionOf(annotations map[string]string) int64 {
	rev, _ := strconv.ParseInt(annotations[revisionAnnotation], 10, 64)
	return rev
}

func ownedByDeployment(refs []metav1.OwnerReference, uid types.UID) bool {
	for _, r := range refs {
		if r.UID == uid {
			return true
		}
	}
	return false
}

// RecommendedResources is the subset of a resource-optimizer
// recommendation ApplyRecommendation needs.
type RecommendedResources struct {
	ReqCPUMilli, LimCPUMilli int64
	ReqMemBytes, LimMemBytes int64
}

// ApplyRecommendation resolves pod's owning Deployment (Pod -> ReplicaSet
// -> Deployment, the same owner-chain project 01 walks) and patches that
// container's resources.requests/limits in the Deployment's pod template
// to rec — applying to the template, not the live pod, since a running
// Pod's resources are immutable pre-in-place-resize and all replicas of
// a Deployment share one template anyway.
func (e *Executor) ApplyRecommendation(ctx context.Context, namespace, pod, container string, rec RecommendedResources) (oldValue, newValue string, err error) {
	p, err := e.clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("get pod: %w", err)
	}
	deployment, err := resolveOwningDeployment(ctx, e.clientset, p)
	if err != nil {
		return "", "", err
	}
	if deployment == "" {
		return "", "", fmt.Errorf("pod %s is not owned by a Deployment", pod)
	}

	dep, err := e.clientset.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("get deployment: %w", err)
	}

	containers := dep.Spec.Template.Spec.Containers
	idx := -1
	for i := range containers {
		if containers[i].Name == container {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "", "", fmt.Errorf("container %s not found in deployment %s", container, deployment)
	}

	oldValue = resourceListString(containers[idx].Resources.Requests, containers[idx].Resources.Limits)

	containers[idx].Resources.Requests = corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(rec.ReqCPUMilli, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(rec.ReqMemBytes, resource.BinarySI),
	}
	containers[idx].Resources.Limits = corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(rec.LimCPUMilli, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(rec.LimMemBytes, resource.BinarySI),
	}
	newValue = resourceListString(containers[idx].Resources.Requests, containers[idx].Resources.Limits)

	if _, err := e.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{}); err != nil {
		return "", "", fmt.Errorf("update deployment: %w", err)
	}
	return oldValue, newValue, nil
}

func resourceListString(requests, limits corev1.ResourceList) string {
	return fmt.Sprintf("req cpu=%s mem=%s, lim cpu=%s mem=%s",
		requests.Cpu().String(), requests.Memory().String(), limits.Cpu().String(), limits.Memory().String())
}

func resolveOwningDeployment(ctx context.Context, clientset kubernetes.Interface, pod *corev1.Pod) (string, error) {
	if len(pod.OwnerReferences) == 0 {
		return "", nil
	}
	owner := pod.OwnerReferences[0]
	if owner.Kind != "ReplicaSet" {
		return "", nil
	}
	rs, err := clientset.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get replicaset: %w", err)
	}
	for _, rsOwner := range rs.OwnerReferences {
		if rsOwner.Kind == "Deployment" {
			return rsOwner.Name, nil
		}
	}
	return "", nil
}

// ScaleDeployment patches only the scale subresource — the same narrow
// write project 04's autoscaler performs, needing deployments/scale
// RBAC rather than write access to the whole Deployment.
func (e *Executor) ScaleDeployment(ctx context.Context, namespace, deployment string, replicas int32) (oldValue, newValue string, err error) {
	current, err := e.clientset.AppsV1().Deployments(namespace).GetScale(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("get scale: %w", err)
	}
	oldValue = strconv.Itoa(int(current.Spec.Replicas))
	newValue = strconv.Itoa(int(replicas))

	current.Spec.Replicas = replicas
	if _, err := e.clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, deployment, current, metav1.UpdateOptions{}); err != nil {
		return "", "", fmt.Errorf("update scale: %w", err)
	}
	return oldValue, newValue, nil
}
