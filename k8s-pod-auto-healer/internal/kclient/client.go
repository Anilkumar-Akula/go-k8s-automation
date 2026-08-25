// Package kclient builds a Kubernetes clientset that works both inside a
// cluster (ServiceAccount token) and on a developer machine (kubeconfig).
package kclient

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// New tries in-cluster config first; that's what a Pod running inside
// Kubernetes has. It falls back to ~/.kube/config for local development.
func New() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return nil, herr
		}
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(config)
}
