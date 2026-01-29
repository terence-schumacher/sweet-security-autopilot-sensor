// Package types defines shared API types for events, alerts, and detection
// used by the controller HTTP API and internal processing.
package types

import "time"

// SecurityEvent is the HTTP/API representation of a security event from agents.
type SecurityEvent struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	Type         string                 `json:"type"`
	Severity     string                 `json:"severity"`
	Timestamp    time.Time              `json:"timestamp"`
	PodName      string                 `json:"pod_name"`
	PodNamespace string                 `json:"pod_namespace"`
	Process      *ProcessEventData      `json:"process,omitempty"`
	Network      *NetworkEventData      `json:"network,omitempty"`
	File         *FileEventData         `json:"file,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessEventData is process-related payload in a security event.
type ProcessEventData struct {
	PID                  int      `json:"pid"`
	PPID                 int      `json:"ppid"`
	Name                 string   `json:"name"`
	Cmdline              []string `json:"cmdline"`
	SuspiciousIndicators []string `json:"suspicious_indicators,omitempty"`
}

// NetworkEventData is network-related payload in a security event.
type NetworkEventData struct {
	Protocol         string `json:"protocol"`
	DstIP            string `json:"dst_ip"`
	DstPort          int    `json:"dst_port"`
	State            string `json:"state"`
	IsExternal       bool   `json:"is_external"`
	IsSuspiciousPort bool   `json:"is_suspicious_port"`
}

// FileEventData is file-related payload in a security event.
type FileEventData struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	OldHash   string `json:"old_hash,omitempty"`
	NewHash   string `json:"new_hash,omitempty"`
}
