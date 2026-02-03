package config

import (
	"os"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns default when unset", func(t *testing.T) {
		os.Unsetenv("APSS_TEST_GETENV_UNSET")
		got := GetEnv("APSS_TEST_GETENV_UNSET", "default")
		if got != "default" {
			t.Errorf("GetEnv(unset) = %q, want %q", got, "default")
		}
	})

	t.Run("returns value when set", func(t *testing.T) {
		os.Setenv("APSS_TEST_GETENV_SET", "myvalue")
		defer os.Unsetenv("APSS_TEST_GETENV_SET")
		got := GetEnv("APSS_TEST_GETENV_SET", "default")
		if got != "myvalue" {
			t.Errorf("GetEnv(set) = %q, want %q", got, "myvalue")
		}
	})

	t.Run("returns default when empty", func(t *testing.T) {
		os.Setenv("APSS_TEST_GETENV_EMPTY", "")
		defer os.Unsetenv("APSS_TEST_GETENV_EMPTY")
		got := GetEnv("APSS_TEST_GETENV_EMPTY", "default")
		if got != "default" {
			t.Errorf("GetEnv(empty) = %q, want %q", got, "default")
		}
	})

	t.Run("trims space", func(t *testing.T) {
		os.Setenv("APSS_TEST_GETENV_TRIM", "  trimmed  ")
		defer os.Unsetenv("APSS_TEST_GETENV_TRIM")
		got := GetEnv("APSS_TEST_GETENV_TRIM", "default")
		if got != "trimmed" {
			t.Errorf("GetEnv(trim) = %q, want %q", got, "trimmed")
		}
	})
}

func TestGetEnvDuration(t *testing.T) {
	t.Run("returns default when unset", func(t *testing.T) {
		os.Unsetenv("APSS_TEST_DURATION_UNSET")
		got := GetEnvDuration("APSS_TEST_DURATION_UNSET", 5*time.Second)
		if got != 5*time.Second {
			t.Errorf("GetEnvDuration(unset) = %v, want 5s", got)
		}
	})

	t.Run("returns default when empty", func(t *testing.T) {
		os.Setenv("APSS_TEST_DURATION_EMPTY", "")
		defer os.Unsetenv("APSS_TEST_DURATION_EMPTY")
		got := GetEnvDuration("APSS_TEST_DURATION_EMPTY", 10*time.Second)
		if got != 10*time.Second {
			t.Errorf("GetEnvDuration(empty) = %v, want 10s", got)
		}
	})

	t.Run("parses valid duration", func(t *testing.T) {
		os.Setenv("APSS_TEST_DURATION_VALID", "30s")
		defer os.Unsetenv("APSS_TEST_DURATION_VALID")
		got := GetEnvDuration("APSS_TEST_DURATION_VALID", time.Second)
		if got != 30*time.Second {
			t.Errorf("GetEnvDuration(30s) = %v, want 30s", got)
		}
	})

	t.Run("returns default on invalid duration", func(t *testing.T) {
		os.Setenv("APSS_TEST_DURATION_INVALID", "not-a-duration")
		defer os.Unsetenv("APSS_TEST_DURATION_INVALID")
		got := GetEnvDuration("APSS_TEST_DURATION_INVALID", 7*time.Second)
		if got != 7*time.Second {
			t.Errorf("GetEnvDuration(invalid) = %v, want 7s", got)
		}
	})
}

func TestDefaultAgentConfig(t *testing.T) {
	cfg := DefaultAgentConfig()
	if cfg.ControllerEndpoint != "apss-controller.apss-system.svc.cluster.local:8080" {
		t.Errorf("ControllerEndpoint = %q", cfg.ControllerEndpoint)
	}
	if cfg.ProcScanInterval != 5*time.Second {
		t.Errorf("ProcScanInterval = %v", cfg.ProcScanInterval)
	}
	if len(cfg.WatchPaths) == 0 {
		t.Error("WatchPaths should be non-empty")
	}
	if len(cfg.SuspiciousProcesses) == 0 {
		t.Error("SuspiciousProcesses should be non-empty")
	}
	if len(cfg.SuspiciousPorts) == 0 {
		t.Error("SuspiciousPorts should be non-empty")
	}
}

func TestDefaultControllerConfig(t *testing.T) {
	os.Unsetenv("SWEET_SECURITY_ENDPOINT")
	os.Unsetenv("SWEET_SECURITY_API_KEY")
	cfg := DefaultControllerConfig()
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.SweetSecurityEnabled {
		t.Error("SweetSecurityEnabled should be false when env unset")
	}
	if cfg.EventBufferSize != 100000 {
		t.Errorf("EventBufferSize = %d", cfg.EventBufferSize)
	}
}

func TestDefaultWebhookConfig(t *testing.T) {
	cfg := DefaultWebhookConfig()
	if cfg.SidecarImage == "" {
		t.Error("SidecarImage should be set")
	}
	if len(cfg.ExcludeNamespaces) == 0 {
		t.Error("ExcludeNamespaces should be non-empty")
	}
	for _, ns := range cfg.ExcludeNamespaces {
		if ns == "" {
			t.Error("ExcludeNamespaces should not contain empty strings")
		}
	}
}
