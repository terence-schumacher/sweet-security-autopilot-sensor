package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EventType represents the type of security event
type EventType int

const (
	EventTypeUnknown EventType = iota
	EventTypeProcessStart
	EventTypeProcessExit
	EventTypeNetworkConnect
	EventTypeNetworkListen
	EventTypeFileCreate
	EventTypeFileModify
	EventTypeFileDelete
	EventTypeFileAccess
	EventTypeResourceAnomaly
	EventTypeDNSQuery
	EventTypeK8sAudit
	EventTypeSuspiciousActivity
)

// Severity levels for events
type Severity int

const (
	SeverityUnknown Severity = iota
	SeverityInfo
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// SecurityEvent is the internal event representation
type SecurityEvent struct {
	ID        string
	Type      EventType
	Severity  Severity
	Timestamp time.Time

	// Source context (filled by collector)
	PodName       string
	PodNamespace  string
	ContainerID   string
	ContainerName string

	// Event payloads (only one is set)
	Process  *ProcessEvent
	Network  *NetworkEvent
	File     *FileEvent
	Resource *ResourceEvent
	DNS      *DNSEvent
	Audit    *AuditEvent

	// Additional context
	Metadata map[string]string
	Tags     []string
}

// ProcessEvent contains process-related event data
type ProcessEvent struct {
	PID                  int
	PPID                 int
	Name                 string
	ExePath              string
	Cmdline              []string
	User                 string
	UID                  int
	StartTime            time.Time
	ExitCode             int
	SuspiciousIndicators []string
}

// NetworkEvent contains network-related event data
type NetworkEvent struct {
	Protocol         string
	SrcIP            string
	SrcPort          int
	DstIP            string
	DstPort          int
	State            string
	PID              int
	ProcessName      string
	IsExternal       bool
	IsSuspiciousPort bool
	GeoLocation      string
}

// FileEvent contains file-related event data
type FileEvent struct {
	Path        string
	Operation   string
	PID         int
	ProcessName string
	OldHash     string
	NewHash     string
	SizeBytes   int64
	Permissions string
}

// ResourceEvent contains resource usage event data
type ResourceEvent struct {
	CPUPercent       float64
	MemoryBytes      int64
	MemoryLimitBytes int64
	MemoryPercent    float64
	DiskReadBytes    int64
	DiskWriteBytes   int64
	NetworkRxBytes   int64
	NetworkTxBytes   int64
	AnomalyType      string
	AnomalyScore     float64
}

// DNSEvent contains DNS query event data
type DNSEvent struct {
	QueryName       string
	QueryType       string
	Answers         []string
	PID             int
	ProcessName     string
	IsSuspiciousTLD bool
	PossibleTunnel  bool
	EntropyScore    float64
}

// AuditEvent contains Kubernetes audit event data
type AuditEvent struct {
	Verb             string
	Resource         string
	Name             string
	Namespace        string
	User             string
	Groups           []string
	SourceIP         string
	UserAgent        string
	ResponseCode     int
	PolicyViolations []string
}

// Config for the event collector
type Config struct {
	ControllerEndpoint string
	AgentID            string
	PodName            string
	PodNamespace       string
	BufferSize         int
}

// EventCollector collects and sends events to the controller
type EventCollector struct {
	cfg Config
	log *logrus.Logger

	// Event channel for incoming events
	eventChan chan SecurityEvent

	// HTTP client for controller
	httpClient *http.Client
	mu         sync.RWMutex

	// Stats
	eventsSent    int64
	eventsDropped int64
}

// New creates a new EventCollector
func New(cfg Config, log *logrus.Logger) (*EventCollector, error) {
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 10000
	}

	return &EventCollector{
		cfg: cfg,
		log: log,
		eventChan: make(chan SecurityEvent, cfg.BufferSize),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// EventChannel returns the channel for sending events
func (ec *EventCollector) EventChannel() chan<- SecurityEvent {
	return ec.eventChan
}

// Start begins the event collection and streaming
func (ec *EventCollector) Start(ctx context.Context) error {
	ec.log.WithField("endpoint", ec.cfg.ControllerEndpoint).Info("Starting event collector")

	// Process events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event := <-ec.eventChan:
			ec.processEvent(ctx, event)
		}
	}
}

// processEvent handles an incoming security event
func (ec *EventCollector) processEvent(ctx context.Context, event SecurityEvent) {
	// Enrich event with pod context
	event.PodName = ec.cfg.PodName
	event.PodNamespace = ec.cfg.PodNamespace

	// Generate event ID if not set
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s-%d", ec.cfg.AgentID, time.Now().UnixNano())
	}

	// Log event locally (always)
	ec.logEvent(event)

	// Send to controller if connected
	if err := ec.sendEvent(ctx, event); err != nil {
		ec.eventsDropped++
		ec.log.WithError(err).Debug("Failed to send event")
	} else {
		ec.eventsSent++
	}
}

// logEvent logs the event locally
func (ec *EventCollector) logEvent(event SecurityEvent) {
	fields := logrus.Fields{
		"event_id":      event.ID,
		"event_type":    event.Type,
		"severity":      event.Severity,
		"pod_name":      event.PodName,
		"pod_namespace": event.PodNamespace,
	}

	// Add event-specific fields
	switch {
	case event.Process != nil:
		fields["process_name"] = event.Process.Name
		fields["process_pid"] = event.Process.PID
		fields["process_cmdline"] = event.Process.Cmdline
		if len(event.Process.SuspiciousIndicators) > 0 {
			fields["suspicious_indicators"] = event.Process.SuspiciousIndicators
		}

	case event.Network != nil:
		fields["protocol"] = event.Network.Protocol
		fields["dst_ip"] = event.Network.DstIP
		fields["dst_port"] = event.Network.DstPort
		fields["state"] = event.Network.State
		fields["is_external"] = event.Network.IsExternal

	case event.File != nil:
		fields["file_path"] = event.File.Path
		fields["file_operation"] = event.File.Operation
		if event.File.OldHash != "" && event.File.NewHash != "" {
			fields["hash_changed"] = event.File.OldHash != event.File.NewHash
		}

	case event.DNS != nil:
		fields["dns_query"] = event.DNS.QueryName
		fields["dns_type"] = event.DNS.QueryType
	}

	// Log at appropriate level based on severity
	switch event.Severity {
	case SeverityCritical:
		ec.log.WithFields(fields).Error("CRITICAL: Security event detected")
	case SeverityHigh:
		ec.log.WithFields(fields).Warn("HIGH: Security event detected")
	case SeverityMedium:
		ec.log.WithFields(fields).Warn("MEDIUM: Security event detected")
	case SeverityLow:
		ec.log.WithFields(fields).Info("LOW: Security event detected")
	default:
		ec.log.WithFields(fields).Debug("Security event")
	}
}

// sendEvent sends an event to the controller via HTTP
func (ec *EventCollector) sendEvent(ctx context.Context, event SecurityEvent) error {
	if ec.cfg.ControllerEndpoint == "" {
		return fmt.Errorf("controller endpoint not configured")
	}

	// Convert event to JSON for HTTP API
	eventJSON, err := ec.eventToJSON(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("http://%s/api/v1/events", ec.cfg.ControllerEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(eventJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := ec.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// eventToJSON converts SecurityEvent to JSON format expected by controller
func (ec *EventCollector) eventToJSON(event SecurityEvent) ([]byte, error) {
	// Map internal event types to controller's expected format
	type ControllerEvent struct {
		ID           string                 `json:"id"`
		AgentID      string                 `json:"agent_id"`
		Type         string                 `json:"type"`
		Severity     string                 `json:"severity"`
		Timestamp    time.Time              `json:"timestamp"`
		PodName      string                 `json:"pod_name"`
		PodNamespace string                 `json:"pod_namespace"`
		Process      interface{}            `json:"process,omitempty"`
		Network      interface{}            `json:"network,omitempty"`
		File         interface{}            `json:"file,omitempty"`
		Metadata     map[string]interface{} `json:"metadata,omitempty"`
	}

	ce := ControllerEvent{
		ID:           event.ID,
		AgentID:      ec.cfg.AgentID,
		Type:         eventTypeToString(event.Type),
		Severity:     severityToString(event.Severity),
		Timestamp:    event.Timestamp,
		PodName:      event.PodName,
		PodNamespace: event.PodNamespace,
		Metadata:     make(map[string]interface{}),
	}

	// Convert metadata
	for k, v := range event.Metadata {
		ce.Metadata[k] = v
	}

	// Add event-specific data
	if event.Process != nil {
		ce.Process = map[string]interface{}{
			"pid":                   event.Process.PID,
			"ppid":                  event.Process.PPID,
			"name":                  event.Process.Name,
			"cmdline":               event.Process.Cmdline,
			"suspicious_indicators": event.Process.SuspiciousIndicators,
		}
	}

	if event.Network != nil {
		ce.Network = map[string]interface{}{
			"protocol":          event.Network.Protocol,
			"dst_ip":            event.Network.DstIP,
			"dst_port":           event.Network.DstPort,
			"state":             event.Network.State,
			"is_external":        event.Network.IsExternal,
			"is_suspicious_port": event.Network.IsSuspiciousPort,
		}
	}

	if event.File != nil {
		ce.File = map[string]interface{}{
			"path":      event.File.Path,
			"operation": event.File.Operation,
			"old_hash":  event.File.OldHash,
			"new_hash":  event.File.NewHash,
		}
	}

	return json.Marshal(ce)
}

// eventTypeToString converts EventType to string
func eventTypeToString(et EventType) string {
	switch et {
	case EventTypeProcessStart:
		return "process_start"
	case EventTypeProcessExit:
		return "process_exit"
	case EventTypeNetworkConnect:
		return "network_connect"
	case EventTypeNetworkListen:
		return "network_listen"
	case EventTypeFileCreate:
		return "file_create"
	case EventTypeFileModify:
		return "file_modify"
	case EventTypeFileDelete:
		return "file_delete"
	case EventTypeFileAccess:
		return "file_access"
	default:
		return "unknown"
	}
}

// severityToString converts Severity to string
func severityToString(s Severity) string {
	switch s {
	case SeverityCritical:
		return "CRITICAL"
	case SeverityHigh:
		return "HIGH"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityLow:
		return "LOW"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// GetStats returns collector statistics
func (ec *EventCollector) GetStats() (sent, dropped int64) {
	return ec.eventsSent, ec.eventsDropped
}
