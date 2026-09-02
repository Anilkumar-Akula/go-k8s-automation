// Package config loads autoscaler settings from the environment.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Namespace         string
	Deployment        string
	PollInterval      time.Duration
	TargetCPUPercent  float64
	Tolerance         float64
	MinReplicas       int32
	MaxReplicas       int32
	ScaleUpCooldown   time.Duration
	ScaleDownCooldown time.Duration
	MetricsAddr       string
}

func Load() Config {
	return Config{
		Namespace:         getEnv("TARGET_NAMESPACE", "autoscaler-demo"),
		Deployment:        getEnv("TARGET_DEPLOYMENT", "demo-app"),
		PollInterval:      getEnvDuration("POLL_INTERVAL", 15*time.Second),
		TargetCPUPercent:  getEnvFloat("TARGET_CPU_PERCENT", 50),
		Tolerance:         getEnvFloat("TOLERANCE", 0.1),
		MinReplicas:       int32(getEnvInt("MIN_REPLICAS", 1)),
		MaxReplicas:       int32(getEnvInt("MAX_REPLICAS", 10)),
		ScaleUpCooldown:   getEnvDuration("SCALE_UP_COOLDOWN", 60*time.Second),
		ScaleDownCooldown: getEnvDuration("SCALE_DOWN_COOLDOWN", 300*time.Second),
		MetricsAddr:       getEnv("METRICS_ADDR", ":8080"),
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

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
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
