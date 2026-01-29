// Package config provides shared configuration loading from environment
// and defaults for all APSS components.
package config

import (
	"os"
	"strings"
	"time"
)

// GetEnv returns the value of key from the environment, or defaultValue if unset or empty.
func GetEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return defaultValue
}

// GetEnvDuration returns the duration for key, or defaultValue if unset/invalid.
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	s := os.Getenv(key)
	if s == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue
	}
	return d
}

// AgentConfig holds configuration for the sidecar agent (used by cmd/agent and pkg/monitor).
type AgentConfig struct {
	AgentID             string
	PodName             string
	PodNamespace        string
	NodeName            string
	ControllerEndpoint  string
	ProcScanInterval    time.Duration
	NetScanInterval     time.Duration
	FileScanInterval    time.Duration
	WatchPaths          []string
	SuspiciousProcesses []string
	SuspiciousPorts     []int
}

// ControllerConfig holds configuration for the controller.
type ControllerConfig struct {
	HTTPAddr              string
	ShutdownTimeout       time.Duration
	EventBufferSize       int
	AlertBufferSize       int
	AgentStaleThreshold   time.Duration
	AlertRetentionCount   int
	SweetSecurityEnabled  bool
	SweetSecurityEndpoint string
	SweetSecurityAPIKey   string
	SweetSecurityTimeout  time.Duration
}

// WebhookConfig holds configuration for the mutating webhook.
type WebhookConfig struct {
	SidecarImage       string
	ControllerEndpoint string
	ExcludeNamespaces  []string
	ExcludeLabels      map[string]string
	TLSCertFile        string
	TLSKeyFile         string
	HTTPAddr           string
}

// DefaultAgentConfig returns agent config from environment with defaults.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		AgentID:             GetEnv("AGENT_ID", ""),
		PodName:             GetEnv("POD_NAME", ""),
		PodNamespace:        GetEnv("POD_NAMESPACE", ""),
		NodeName:            GetEnv("NODE_NAME", ""),
		ControllerEndpoint:  GetEnv("CONTROLLER_ENDPOINT", "apss-controller.apss-system.svc.cluster.local:8080"),
		ProcScanInterval:    GetEnvDuration("PROC_SCAN_INTERVAL", 5*time.Second),
		NetScanInterval:     GetEnvDuration("NET_SCAN_INTERVAL", 10*time.Second),
		FileScanInterval:    GetEnvDuration("FILE_SCAN_INTERVAL", 30*time.Second),
		WatchPaths:          defaultWatchPaths(),
		SuspiciousProcesses: defaultSuspiciousProcesses(),
		SuspiciousPorts:     defaultSuspiciousPorts(),
	}
}

func defaultWatchPaths() []string {
	return []string{
		"/etc/passwd", "/etc/shadow", "/etc/sudoers",
		"/root/.ssh", "/etc/crontab", "/var/spool/cron",
	}
}

func defaultSuspiciousProcesses() []string {
	return []string{
		"nc", "ncat", "netcat", "nmap", "masscan",
		"tcpdump", "wireshark",
		"python -c", "perl -e", "ruby -e",
		"bash -i", "sh -i",
		"xmrig", "minerd", "cpuminer",
		"socat", "curl.*|.*sh", "wget.*|.*sh",
	}
}

func defaultSuspiciousPorts() []int {
	return []int{4444, 5555, 6666, 1337, 3389, 5900, 5901, 6379, 27017}
}

// DefaultControllerConfig returns controller config from environment.
func DefaultControllerConfig() ControllerConfig {
	ep := GetEnv("SWEET_SECURITY_ENDPOINT", "")
	key := GetEnv("SWEET_SECURITY_API_KEY", "")
	return ControllerConfig{
		HTTPAddr:              GetEnv("HTTP_ADDR", ":8080"),
		ShutdownTimeout:       GetEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		EventBufferSize:       100000,
		AlertBufferSize:       10000,
		AgentStaleThreshold:   2 * time.Minute,
		AlertRetentionCount:   10000,
		SweetSecurityEnabled:  ep != "" && key != "",
		SweetSecurityEndpoint: ep,
		SweetSecurityAPIKey:   key,
		SweetSecurityTimeout:  GetEnvDuration("SWEET_SECURITY_TIMEOUT", 30*time.Second),
	}
}

// DefaultWebhookConfig returns webhook config from environment.
func DefaultWebhookConfig() WebhookConfig {
	exclude := GetEnv("EXCLUDE_NAMESPACES", "kube-system,kube-public,apss-system")
	namespaces := strings.Split(exclude, ",")
	for i, n := range namespaces {
		namespaces[i] = strings.TrimSpace(n)
	}
	return WebhookConfig{
		SidecarImage:       GetEnv("SIDECAR_IMAGE", "gcr.io/invisible-sre-sandbox/apss-agent:latest"),
		ControllerEndpoint: GetEnv("CONTROLLER_ENDPOINT", "apss-controller.apss-system.svc.cluster.local:8080"),
		ExcludeNamespaces:  namespaces,
		ExcludeLabels:      nil,
		TLSCertFile:        GetEnv("TLS_CERT_FILE", "/etc/webhook/certs/tls.crt"),
		TLSKeyFile:         GetEnv("TLS_KEY_FILE", "/etc/webhook/certs/tls.key"),
		HTTPAddr:           GetEnv("HTTP_ADDR", ":8443"),
	}
}
