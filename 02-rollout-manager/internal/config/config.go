package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Namespace           string
	MaxRollbackAttempts int
	InitialBackoff      time.Duration
	RetryTTL            time.Duration
	MetricsAddr         string
}

func Load() Config {
	return Config{
		Namespace:           getEnv("TARGET_NAMESPACE", "rollout-manager-demo"),
		MaxRollbackAttempts: getEnvInt("MAX_ROLLBACK_ATTEMPTS", 3),
		InitialBackoff:      getEnvDuration("INITIAL_BACKOFF", 2*time.Second),
		RetryTTL:            getEnvDuration("RETRY_TTL", 10*time.Minute),
		MetricsAddr:         getEnv("METRICS_ADDR", ":8080"),
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
