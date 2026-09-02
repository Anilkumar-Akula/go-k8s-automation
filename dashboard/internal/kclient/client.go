package kclient

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Clients holds both standard Kubernetes and metrics-server clients.
type Clients struct {
	Kube    kubernetes.Interface
	Metrics metricsv1beta1.Interface
	Config  *rest.Config
	Live    bool
}

// New creates Kubernetes clients, trying in-cluster first, then kubeconfig.
// If both fail, it returns an empty client marked as Live=false without erroring out.
func New() (*Clients, error) {
	cfg, err := rest.InClusterConfig()
	if err == nil {
		kube, kErr := kubernetes.NewForConfig(cfg)
		metrics, mErr := metricsv1beta1.NewForConfig(cfg)
		if kErr == nil {
			return &Clients{
				Kube:    kube,
				Metrics: metrics,
				Config:  cfg,
				Live:    mErr == nil || kube != nil,
			}, nil
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
				return &Clients{
					Kube:    kube,
					Metrics: metrics,
					Config:  cfg,
					Live:    true,
				}, nil
			}
		}
	}

	return &Clients{Live: false}, fmt.Errorf("no live Kubernetes cluster reachable")
}
