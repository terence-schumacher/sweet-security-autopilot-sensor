# Sweet Security API Integration

## Overview

The Autopilot Security Sensor now includes full integration with the Sweet Security API. This allows security alerts and events to be automatically forwarded to Sweet Security for centralized monitoring and analysis.

## Implementation Details

### Components Added

1. **Sweet Security Client Package** (`pkg/sweetsecurity/client.go`)
   - HTTP client for communicating with Sweet Security API
   - Supports sending alerts and events
   - Includes health check functionality
   - Handles authentication via Bearer token

2. **Controller Integration** (`cmd/controller/main.go`)
   - Automatically initializes Sweet Security client when configured
   - Sends alerts to Sweet Security when generated
   - Sends high-severity events (CRITICAL/HIGH) to Sweet Security
   - Non-blocking async sending to avoid impacting alert processing

### API Endpoints Used

The integration uses the following Sweet Security API endpoints:

- `POST /api/v1/alerts` - Send security alerts
- `POST /api/v1/events` - Send security events
- `POST /api/v1/events/batch` - Send multiple events in batch (available but not currently used)
- `GET /health` - Health check endpoint

### Authentication

Authentication is done via Bearer token in the `Authorization` header:
```
Authorization: Bearer <API_KEY>
```

## Configuration

### Helm Values

Enable Sweet Security integration in your Helm values:

```yaml
sweetSecurity:
  enabled: true
  apiEndpoint: "https://api.sweet.security"
  apiKeySecret:
    name: sweet-api-key
    key: api-key
```

### Environment Variables

The controller reads the following environment variables:

- `SWEET_SECURITY_ENDPOINT` - API endpoint URL (required)
- `SWEET_SECURITY_API_KEY` - API key for authentication (required)

### Creating the Secret

Before deploying, create a Kubernetes secret with your API key:

```bash
kubectl create secret generic sweet-api-key \
  --from-literal=api-key=YOUR_API_KEY \
  -n apss-system
```

### Deploying with Sweet Security

```bash
# Deploy with Sweet Security enabled
helm upgrade --install apss ./deploy/helm \
  --namespace apss-system \
  --set sweetSecurity.enabled=true \
  --set sweetSecurity.apiEndpoint="https://api.sweet.security" \
  --set sweetSecurity.apiKeySecret.name=sweet-api-key \
  --set sweetSecurity.apiKeySecret.key=api-key
```

## Data Flow

### Alerts

When a detection rule triggers and generates an alert:

1. Alert is logged locally
2. Alert is stored in controller's in-memory buffer
3. Alert is sent to Sweet Security API (if configured)
4. Prometheus metrics are updated

### Events

High-severity events (CRITICAL or HIGH) are automatically forwarded:

1. Event is received from agent
2. Event is queued for processing
3. If severity is CRITICAL or HIGH, event is sent to Sweet Security
4. Event is processed by detection rules

## Alert Format

Alerts sent to Sweet Security include:

```json
{
  "id": "alert-1234567890",
  "timestamp": "2024-01-08T16:00:00Z",
  "severity": "CRITICAL",
  "rule_id": "APSS-001",
  "rule_name": "Potential Reverse Shell",
  "description": "Detected network connection matching reverse shell pattern",
  "pod_name": "my-app-abc123",
  "pod_namespace": "default",
  "mitre_tactic": "Command and Control",
  "mitre_id": "T1059.004",
  "event_ids": ["event-123"],
  "metadata": {
    "source": "apss-autopilot-security-sensor",
    "recommended_actions": ["Investigate pod immediately", "Check for unauthorized processes"]
  }
}
```

## Event Format

Events sent to Sweet Security include:

```json
{
  "id": "event-1234567890",
  "agent_id": "my-app-abc123-default",
  "type": "network_connect",
  "severity": "CRITICAL",
  "timestamp": "2024-01-08T16:00:00Z",
  "pod_name": "my-app-abc123",
  "pod_namespace": "default",
  "network": {
    "protocol": "tcp",
    "dst_ip": "1.2.3.4",
    "dst_port": 4444,
    "state": "ESTABLISHED",
    "is_external": true,
    "is_suspicious_port": true
  },
  "metadata": {}
}
```

## Error Handling

- **Connection Failures**: Errors are logged but do not block alert processing
- **API Errors**: Non-2xx responses are logged with error details
- **Missing Configuration**: Integration gracefully degrades if not configured
- **Health Checks**: Background health check verifies connectivity on startup

## Monitoring

The integration includes logging for:

- Successful API calls (debug level)
- Failed API calls (error level)
- Health check results (info/warn level)
- Configuration status (debug level)

## Testing

### Verify Integration is Working

1. Check controller logs for Sweet Security initialization:
   ```bash
   kubectl logs -f deployment/apss-controller -n apss-system | grep -i "sweet"
   ```

2. Trigger a test alert and verify it's sent:
   ```bash
   # Create a pod that triggers an alert
   kubectl run test-alert --image=nginx --restart=Never
   
   # Check controller logs
   kubectl logs deployment/apss-controller -n apss-system | grep "Sweet Security"
   ```

3. Check Sweet Security dashboard for received alerts

### Health Check

The controller performs a health check on startup. To verify:

```bash
kubectl logs deployment/apss-controller -n apss-system | grep "Sweet Security API connection verified"
```

## Troubleshooting

### Integration Not Working

1. **Check Configuration**:
   ```bash
   kubectl get deployment apss-controller -n apss-system -o yaml | grep -A 5 SWEET
   ```

2. **Verify Secret Exists**:
   ```bash
   kubectl get secret sweet-api-key -n apss-system
   ```

3. **Check Logs**:
   ```bash
   kubectl logs deployment/apss-controller -n apss-system | grep -i "sweet\|error"
   ```

### Common Issues

- **"sweet security client not configured"**: Environment variables not set
- **"unexpected status code: 401"**: Invalid API key
- **"failed to send request"**: Network connectivity issue or wrong endpoint URL

## Future Enhancements

Potential improvements:

1. **Event Batching**: Batch multiple events for efficiency
2. **Retry Logic**: Automatic retry with exponential backoff
3. **Rate Limiting**: Respect API rate limits
4. **Event Filtering**: Configurable filters for which events to send
5. **Metrics**: Prometheus metrics for API call success/failure rates
