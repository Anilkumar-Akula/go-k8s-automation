package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	c := Load()
	if c.Namespace != "cost-optimizer-demo" {
		t.Errorf("Namespace = %q, want default", c.Namespace)
	}
	if c.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want 30s", c.PollInterval)
	}
	if c.WindowSize != 60 {
		t.Errorf("WindowSize = %d, want 60", c.WindowSize)
	}
	if c.MetricsAddr != ":8080" {
		t.Errorf("MetricsAddr = %q, want :8080", c.MetricsAddr)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("TARGET_NAMESPACE", "prod")
	t.Setenv("POLL_INTERVAL", "10s")
	t.Setenv("WINDOW_SIZE", "120")
	t.Setenv("MIN_SAMPLES", "3")
	t.Setenv("WINDOW_TTL", "5m")
	t.Setenv("CPU_REQUEST_MARGIN", "1.5")
	t.Setenv("CPU_CORE_HOUR_RATE", "0.05")
	t.Setenv("METRICS_ADDR", ":9090")

	c := Load()
	if c.Namespace != "prod" {
		t.Errorf("Namespace = %q, want prod", c.Namespace)
	}
	if c.PollInterval != 10*time.Second {
		t.Errorf("PollInterval = %v, want 10s", c.PollInterval)
	}
	if c.WindowSize != 120 {
		t.Errorf("WindowSize = %d, want 120", c.WindowSize)
	}
	if c.MinSamples != 3 {
		t.Errorf("MinSamples = %d, want 3", c.MinSamples)
	}
	if c.WindowTTL != 5*time.Minute {
		t.Errorf("WindowTTL = %v, want 5m", c.WindowTTL)
	}
	if c.CPURequestMargin != 1.5 {
		t.Errorf("CPURequestMargin = %v, want 1.5", c.CPURequestMargin)
	}
	if c.CPUCorePerHourRate != 0.05 {
		t.Errorf("CPUCorePerHourRate = %v, want 0.05", c.CPUCorePerHourRate)
	}
	if c.MetricsAddr != ":9090" {
		t.Errorf("MetricsAddr = %q, want :9090", c.MetricsAddr)
	}
}

func TestLoadInvalidEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("WINDOW_SIZE", "not-a-number")
	t.Setenv("CPU_REQUEST_MARGIN", "not-a-float")
	t.Setenv("POLL_INTERVAL", "not-a-duration")

	c := Load()
	if c.WindowSize != 60 {
		t.Errorf("WindowSize = %d, want default 60 on invalid input", c.WindowSize)
	}
	if c.CPURequestMargin != 1.1 {
		t.Errorf("CPURequestMargin = %v, want default 1.1 on invalid input", c.CPURequestMargin)
	}
	if c.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want default 30s on invalid input", c.PollInterval)
	}
}
