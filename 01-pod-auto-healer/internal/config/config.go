package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Namespace              string
	MaxRemediationAttempts int
	InitialBackoff         time.Duration
	RestartThreshold       int32
	RetryTTL               time.Duration
	MetricsAddr            string
}

func Load() Config {
	return Config{
		Namespace:              getEnv("TARGET_NAMESPACE", "auto-healer-demo"),
		MaxRemediationAttempts: getEnvInt("MAX_REMEDIATION_ATTEMPTS", 3),
		InitialBackoff:         getEnvDuration("INITIAL_BACKOFF", 2*time.Second),
		RestartThreshold:       int32(getEnvInt("RESTART_THRESHOLD", 3)),
		RetryTTL:               getEnvDuration("RETRY_TTL", 10*time.Minute),
		MetricsAddr:            getEnv("METRICS_ADDR", ":8080"),
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
