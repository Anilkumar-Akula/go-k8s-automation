// Package config loads dashboard-api settings from the environment: where
// to listen, how often to poll, and where each controller's /metrics
// endpoint lives.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr   string
	PollInterval time.Duration
	EventBuffer  int

	AutoHealer        ControllerTarget
	RolloutManager    ControllerTarget
	ResourceOptimizer ControllerTarget
	Autoscaler        AutoscalerTarget
}

// ControllerTarget is where to scrape one controller's Prometheus metrics
// and which namespace it's watching (for display/labeling).
type ControllerTarget struct {
	Name       string
	Namespace  string
	MetricsURL string
}

// AutoscalerTarget also names the Deployment it scales, since its metrics
// are labeled per-deployment rather than cluster-wide.
type AutoscalerTarget struct {
	ControllerTarget
	Deployment string
}

func Load() Config {
	return Config{
		ListenAddr:   getEnv("LISTEN_ADDR", ":8090"),
		PollInterval: getEnvDuration("POLL_INTERVAL", 15*time.Second),
		EventBuffer:  getEnvInt("EVENT_BUFFER_SIZE", 200),

		AutoHealer: ControllerTarget{
			Name:       "auto-healer",
			Namespace:  getEnv("AUTO_HEALER_NAMESPACE", "auto-healer-demo"),
			MetricsURL: getEnv("AUTO_HEALER_METRICS_URL", "http://auto-healer.auto-healer-demo.svc.cluster.local:8080/metrics"),
		},
		RolloutManager: ControllerTarget{
			Name:       "rollout-manager",
			Namespace:  getEnv("ROLLOUT_MANAGER_NAMESPACE", "rollout-manager-demo"),
			MetricsURL: getEnv("ROLLOUT_MANAGER_METRICS_URL", "http://rollout-manager.rollout-manager-demo.svc.cluster.local:8080/metrics"),
		},
		ResourceOptimizer: ControllerTarget{
			Name:       "resource-optimizer",
			Namespace:  getEnv("RESOURCE_OPTIMIZER_NAMESPACE", "resource-optimizer-demo"),
			MetricsURL: getEnv("RESOURCE_OPTIMIZER_METRICS_URL", "http://resource-optimizer.resource-optimizer-demo.svc.cluster.local:8080/metrics"),
		},
		Autoscaler: AutoscalerTarget{
			ControllerTarget: ControllerTarget{
				Name:       "autoscaler",
				Namespace:  getEnv("AUTOSCALER_NAMESPACE", "autoscaler-demo"),
				MetricsURL: getEnv("AUTOSCALER_METRICS_URL", "http://autoscaler.autoscaler-demo.svc.cluster.local:8080/metrics"),
			},
			Deployment: getEnv("AUTOSCALER_DEPLOYMENT", "demo-app"),
		},
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
