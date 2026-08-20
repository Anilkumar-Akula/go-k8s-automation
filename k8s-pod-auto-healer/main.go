package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	homedir, error := os.UserHomeDir()
	if error != nil {
		panic(error)

	}
	kubeconfig := filepath.Join(homedir, ".kube", "config")

	//load kubeconfig file
	config, error := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if error != nil {
		panic(error)
	}

	//create clientset
	clientset, error := kubernetes.NewForConfig(config)
	if error != nil {
		panic(error)
	}
	fmt.Println("Connected to Kubernetes!")
	fmt.Println("Watching Pods...")
	fmt.Println("--------------------------------")

	//get all pods in all namespaces
	pods, error := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if error != nil {
		panic(error)
	}

	for _, pod := range pods.Items {
		fmt.Printf("Namespace: %-20s pod: %-50s status: %s\n", pod.Namespace, pod.Name, pod.Status.Phase)
	}

	// Watch all Pods in all namespaces.
	watcher, err := clientset.CoreV1().
		Pods("").
		Watch(context.Background(), metav1.ListOptions{})

	if err != nil {
		log.Fatalf("failed to watch pods: %v", err)
	}

	defer watcher.Stop()

	// Receive events continuously.
	for event := range watcher.ResultChan() {

		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		switch event.Type {

		case "ADDED":
			fmt.Printf(
				"[ADDED] %s/%s - %s\n",
				pod.Namespace,
				pod.Name,
				pod.Status.Phase,
			)
			if checkPodHealth(pod) {
				remediatePod(clientset, pod)
			}

		case "MODIFIED":
			fmt.Printf(
				"[MODIFIED] %s/%s - %s\n",
				pod.Namespace,
				pod.Name,
				pod.Status.Phase,
			)
			if checkPodHealth(pod) {
				remediatePod(clientset, pod)
			}
		case "DELETED":
			fmt.Printf(
				"[DELETED] %s/%s - %s\n",
				pod.Namespace,
				pod.Name,
				pod.Status.Phase,
			)
			if checkPodHealth(pod) {
				remediatePod(clientset, pod)
			}

		case "ERROR":
			fmt.Printf(
				"[ERROR] %s/%s\n",
				pod.Namespace,
				pod.Name,
			)
			if checkPodHealth(pod) {
				remediatePod(clientset, pod)
			}
		}
	}

}
func checkPodHealth(pod *corev1.Pod) bool {

	// Pod failed
	if pod.Status.Phase == corev1.PodFailed {
		fmt.Printf(
			"🚨 UNHEALTHY POD: %s/%s - Phase=Failed\n",
			pod.Namespace,
			pod.Name,
		)
		return true
	}

	// Check containers
	for _, containerStatus := range pod.Status.ContainerStatuses {

		// Too many restarts
		if containerStatus.RestartCount >= 3 {
			fmt.Printf(
				"🚨 HIGH RESTART COUNT: %s/%s - Container=%s Restarts=%d\n",
				pod.Namespace,
				pod.Name,
				containerStatus.Name,
				containerStatus.RestartCount,
			)
			return true
		}

		// CrashLoopBackOff
		if containerStatus.State.Waiting != nil {

			reason := containerStatus.State.Waiting.Reason

			if strings.Contains(reason, "CrashLoopBackOff") {
				fmt.Printf(
					"🚨 CRASH LOOP: %s/%s - Container=%s\n",
					pod.Namespace,
					pod.Name,
					containerStatus.Name,
				)
			}

			if strings.Contains(reason, "ImagePullBackOff") {
				fmt.Printf(
					"🚨 IMAGE PULL ERROR: %s/%s - Container=%s\n",
					pod.Namespace,
					pod.Name,
					containerStatus.Name,
				)
			}
		}

		// OOMKilled
		if containerStatus.LastTerminationState.Terminated != nil {

			reason := containerStatus.LastTerminationState.Terminated.Reason

			if reason == "OOMKilled" {
				fmt.Printf(
					"🚨 OOM KILLED: %s/%s - Container=%s\n",
					pod.Namespace,
					pod.Name,
					containerStatus.Name,
				)
			}
		}
	}
	return false
}
func remediatePod(clientset *kubernetes.Clientset, pod *corev1.Pod) {

	if pod.Namespace != "auto-healer-demo" {
		return
	}

	fmt.Printf(
		"🔧 REMEDIATING: %s/%s\n",
		pod.Namespace,
		pod.Name,
	)

	err := clientset.CoreV1().
		Pods(pod.Namespace).
		Delete(
			context.Background(),
			pod.Name,
			metav1.DeleteOptions{},
		)

	if err != nil {
		fmt.Printf(
			"❌ REMEDIATION FAILED: %s/%s - %v\n",
			pod.Namespace,
			pod.Name,
			err,
		)
		return
	}

	fmt.Printf(
		"✅ POD DELETED: %s/%s\n",
		pod.Namespace,
		pod.Name,
	)
}
