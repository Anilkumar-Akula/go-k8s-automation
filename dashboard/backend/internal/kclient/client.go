// Package kclient builds a read-only Kubernetes API client that works
// both inside a cluster (ServiceAccount token) and on a developer machine
// (kubeconfig). The dashboard never mutates cluster state directly — all
// control actions route through the individual controllers.
package kclient

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

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

func New(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}
