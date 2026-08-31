// Package kclient builds Kubernetes API clients that work both inside a
// cluster (ServiceAccount token) and on a developer machine (kubeconfig).
package kclient

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Config resolves in-cluster config first, falling back to ~/.kube/config
// for local development.
func Config() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}
	home, herr := os.UserHomeDir()
	if herr != nil {
		return nil, herr
	}
	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}

// New builds a core Kubernetes clientset (Pods, container specs).
func New(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

// NewMetrics builds a metrics.k8s.io clientset (metrics-server) for
// live Pod/container CPU and memory usage.
func NewMetrics(config *rest.Config) (*metricsv.Clientset, error) {
	return metricsv.NewForConfig(config)
}
