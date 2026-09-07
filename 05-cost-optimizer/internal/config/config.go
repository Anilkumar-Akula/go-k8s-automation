// Package config loads cost-optimizer settings from the environment.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Namespace        string
	PollInterval     time.Duration
	WindowSize       int
	MinSamples       int
	WindowTTL        time.Duration
	CPURequestMargin float64
	CPULimitMargin   float64
	MemRequestMargin float64
	MemLimitMargin   float64
	MetricsAddr      string

	CPUCorePerHourRate float64
	MemGiBPerHourRate  float64
	TopN               int // how many wasteful containers to log per poll
}

func Load() Config {
	return Config{
		Namespace:        getEnv("TARGET_NAMESPACE", "cost-optimizer-demo"),
		PollInterval:     getEnvDuration("POLL_INTERVAL", 30*time.Second),
		WindowSize:       getEnvInt("WINDOW_SIZE", 60),
		MinSamples:       getEnvInt("MIN_SAMPLES", 5),
		WindowTTL:        getEnvDuration("WINDOW_TTL", 15*time.Minute),
		CPURequestMargin: getEnvFloat("CPU_REQUEST_MARGIN", 1.1),
		CPULimitMargin:   getEnvFloat("CPU_LIMIT_MARGIN", 1.3),
		MemRequestMargin: getEnvFloat("MEM_REQUEST_MARGIN", 1.1),
		MemLimitMargin:   getEnvFloat("MEM_LIMIT_MARGIN", 1.3),
		MetricsAddr:      getEnv("METRICS_ADDR", ":8080"),

		CPUCorePerHourRate: getEnvFloat("CPU_CORE_HOUR_RATE", 0.033),
		MemGiBPerHourRate:  getEnvFloat("MEM_GIB_HOUR_RATE", 0.0045),
		TopN:               getEnvInt("TOP_N", 5),
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
