package sweetsecurity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Client handles communication with Sweet Security API
type Client struct {
	apiEndpoint string
	apiKey      string
	httpClient  *http.Client
	log         *logrus.Logger
}

// Config for Sweet Security client
type Config struct {
	APIEndpoint string
	APIKey      string
	Timeout     time.Duration
}

// NewClient creates a new Sweet Security API client
func NewClient(cfg Config, log *logrus.Logger) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		apiEndpoint: cfg.APIEndpoint,
		apiKey:      cfg.APIKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		log: log,
	}
}

// Alert represents a security alert to send to Sweet Security
type Alert struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    string                 `json:"severity"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Description string                 `json:"description"`
	PodName     string                 `json:"pod_name"`
	PodNamespace string                `json:"pod_namespace"`
	ClusterName string                 `json:"cluster_name,omitempty"`
	MitreTactic string                 `json:"mitre_tactic,omitempty"`
	MitreID     string                 `json:"mitre_id,omitempty"`
	EventIDs    []string               `json:"event_ids,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Event represents a security event to send to Sweet Security
type Event struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	Type         string                 `json:"type"`
	Severity     string                 `json:"severity"`
	Timestamp    time.Time              `json:"timestamp"`
	PodName      string                 `json:"pod_name"`
	PodNamespace string                 `json:"pod_namespace"`
	Process      map[string]interface{} `json:"process,omitempty"`
	Network      map[string]interface{} `json:"network,omitempty"`
	File         map[string]interface{} `json:"file,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SendAlert sends a security alert to Sweet Security API
func (c *Client) SendAlert(ctx context.Context, alert *Alert) error {
	if c.apiEndpoint == "" || c.apiKey == "" {
		return fmt.Errorf("sweet security client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/alerts", c.apiEndpoint)
	return c.sendJSON(ctx, url, alert)
}

// SendEvent sends a security event to Sweet Security API
func (c *Client) SendEvent(ctx context.Context, event *Event) error {
	if c.apiEndpoint == "" || c.apiKey == "" {
		return fmt.Errorf("sweet security client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/events", c.apiEndpoint)
	return c.sendJSON(ctx, url, event)
}

// SendBatchEvents sends multiple events in a batch
func (c *Client) SendBatchEvents(ctx context.Context, events []*Event) error {
	if c.apiEndpoint == "" || c.apiKey == "" {
		return fmt.Errorf("sweet security client not configured")
	}

	url := fmt.Sprintf("%s/api/v1/events/batch", c.apiEndpoint)
	payload := map[string]interface{}{
		"events": events,
	}
	return c.sendJSON(ctx, url, payload)
}

// sendJSON sends a JSON payload to the API
func (c *Client) sendJSON(ctx context.Context, url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("User-Agent", "apss-autopilot-security-sensor/0.1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.log.WithFields(logrus.Fields{
		"url":    url,
		"status": resp.StatusCode,
	}).Debug("Successfully sent to Sweet Security API")

	return nil
}

// HealthCheck checks if the Sweet Security API is reachable
func (c *Client) HealthCheck(ctx context.Context) error {
	if c.apiEndpoint == "" || c.apiKey == "" {
		return fmt.Errorf("sweet security client not configured")
	}

	url := fmt.Sprintf("%s/health", c.apiEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}
