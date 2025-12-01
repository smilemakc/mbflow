# MBFlow Trigger Examples

This directory contains complete workflow examples demonstrating each trigger type.

## Examples

### 1. Cron Trigger - Daily Sales Report

**File:** `cron_daily_report.json`

A comprehensive example of a scheduled workflow that:
- Runs daily at 9:00 AM EST
- Fetches sales data from an API
- Transforms the data into a report format
- Sends the report via email

**Key Features:**
- Timezone-aware scheduling
- Multiple sequential nodes
- Template-based data transformation
- Third-party API integration (SendGrid)

**Setup:**
```bash
# Create workflow
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @cron_daily_report.json

# The cron trigger will start automatically if enabled
```

---

### 2. Webhook Trigger - GitHub Integration

**File:** `webhook_github_integration.json`

Processes GitHub push events and triggers CI/CD pipeline:
- Receives webhook from GitHub
- Validates push event data
- Checks if push is to main branch
- Triggers CI/CD build
- Notifies team on Slack

**Security:**
- HMAC signature verification
- IP whitelist (GitHub's webhook IPs)
- Conditional execution based on branch

**Setup:**
```bash
# 1. Create workflow and trigger
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @webhook_github_integration.json

# 2. Get trigger ID from response
TRIGGER_ID="<trigger-id>"

# 3. Configure GitHub webhook
# URL: https://your-domain.com/api/v1/webhooks/$TRIGGER_ID
# Content type: application/json
# Secret: your-github-webhook-secret
# Events: Just the push event
```

---

### 3. Event Trigger - User Onboarding

**File:** `event_user_onboarding.json`

Automated user onboarding triggered by user creation events:
- Listens for `user.created` events via Redis pub/sub
- Creates user profile
- Sends welcome email
- Creates onboarding tasks
- Tracks event in analytics

**Features:**
- Event filtering (only active users from API source)
- Parallel execution (profile + tasks)
- Multiple third-party integrations
- Analytics tracking

**Setup:**
```bash
# 1. Create workflow and trigger
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @event_user_onboarding.json

# 2. Publish events from your application
curl -X POST http://localhost:8181/api/v1/events/publish \
  -H "Content-Type: application/json" \
  -d '{
    "type": "user.created",
    "source": "api",
    "data": {
      "user_id": "123",
      "email": "user@example.com",
      "name": "John Doe",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z"
    }
  }'
```

---

### 4. Interval Trigger - Health Check

**File:** `interval_health_check.json`

Continuous service monitoring that:
- Checks health every 30 seconds
- Monitors API, database, and Redis
- Aggregates health status
- Sends alerts to PagerDuty on failures
- Logs all health check results

**Features:**
- Parallel health checks
- Retry logic with timeout
- Conditional alerting
- Comprehensive logging

**Setup:**
```bash
# Create workflow and trigger
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @interval_health_check.json

# Monitor execution logs
curl http://localhost:8181/api/v1/executions?limit=10
```

---

### 5. Manual Trigger - Data Export

**File:** `manual_data_export.json`

On-demand data export with flexible parameters:
- User-initiated execution
- Flexible date range filtering
- Multiple format support (CSV, JSON, XLSX)
- Cloud storage upload
- Email notification with signed download link

**Features:**
- Input schema validation
- Default parameter values
- Long-running workflow support
- Secure temporary download links

**Setup:**
```bash
# 1. Create workflow and trigger
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @manual_data_export.json

# 2. Get trigger ID from response
TRIGGER_ID="<trigger-id>"

# 3. Execute with custom parameters
curl -X POST http://localhost:8181/api/v1/triggers/$TRIGGER_ID/execute \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "start_date": "2024-01-01",
      "end_date": "2024-01-31",
      "format": "csv",
      "include_deleted": false,
      "filters": {
        "status": "active",
        "category": "sales"
      },
      "user_email": "user@example.com"
    }
  }'
```

---

## Common Configuration

All examples require the following environment variables:

```bash
# API Server
PORT=8181
HOST=0.0.0.0

# Database
DATABASE_URL=postgres://mbflow:mbflow@localhost:5432/mbflow?sslmode=disable

# Redis (required for triggers)
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## Testing Examples

### Unit Test Individual Triggers

```bash
# Test cron expression parsing
go test ./internal/application/trigger/... -run TestParseCronSchedule -v

# Test event filtering
go test ./internal/application/trigger/... -run TestEventListener_MatchesFilter -v

# Test webhook signature
go test ./internal/application/trigger/... -run TestWebhookRegistry_ValidateSignature -v
```

### Integration Testing

```bash
# Start dependencies
docker-compose up -d postgres redis

# Run full integration tests
go test ./internal/application/trigger/... -v

# Test with actual Redis
go test ./internal/application/trigger/... -v -tags=integration
```

### End-to-End Testing

```bash
# 1. Start server
go run cmd/server/main.go

# 2. Create workflow and trigger
./scripts/test-trigger-cron.sh

# 3. Monitor execution
watch -n 1 'curl -s http://localhost:8181/api/v1/executions?limit=5 | jq'
```

## Customization

### Modifying Schedules

Cron expressions:
```json
{
  "schedule": "0 0 9 * * *",     // Every day at 9 AM
  "schedule": "0 */15 * * * *",  // Every 15 minutes
  "schedule": "0 0 0 * * 1",     // Every Monday at midnight
  "schedule": "0 0 12 1 * *"     // First day of month at noon
}
```

Intervals:
```json
{
  "interval": "30s",   // 30 seconds
  "interval": "5m",    // 5 minutes
  "interval": "1h",    // 1 hour
  "interval": 60       // 60 seconds (integer)
}
```

### Adding Security

Webhook HMAC:
```json
{
  "config": {
    "secret": "your-webhook-secret-min-32-chars",
    "ip_whitelist": [
      "192.168.1.0/24",
      "10.0.0.1"
    ]
  }
}
```

### Event Filtering

```json
{
  "config": {
    "event_type": "user.created",
    "filter": {
      "source": "api",
      "status": "active",
      "role": "admin"
    }
  }
}
```

## Monitoring

### Check Trigger State

```bash
# Get trigger details
curl http://localhost:8181/api/v1/triggers/{trigger_id}

# Check Redis state
redis-cli GET "trigger:{trigger_id}:state"
```

### View Executions

```bash
# Recent executions
curl http://localhost:8181/api/v1/executions?limit=10

# Executions for specific workflow
curl http://localhost:8181/api/v1/executions?workflow_id={workflow_id}

# Failed executions
curl http://localhost:8181/api/v1/executions?status=failed
```

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug go run cmd/server/main.go

# Monitor Redis pub/sub
redis-cli PSUBSCRIBE "mbflow:events:*"

# Check cron scheduler
curl http://localhost:8181/api/v1/health
```

## Best Practices

1. **Cron Triggers**
   - Use timezone-aware schedules for user-facing features
   - Ensure workflows complete before next trigger
   - Monitor execution duration
   - Set reasonable retry limits

2. **Webhook Triggers**
   - Always configure HMAC secrets in production
   - Use IP whitelisting for known sources
   - Implement idempotency in workflows
   - Return 202 quickly, process asynchronously

3. **Event Triggers**
   - Use specific event types
   - Implement filters to reduce processing
   - Keep event payloads small
   - Handle missing data gracefully

4. **Interval Triggers**
   - Choose intervals longer than workflow duration
   - Monitor execution times
   - Implement circuit breakers for failures
   - Use for simple periodic tasks

5. **Manual Triggers**
   - Validate user permissions
   - Set reasonable timeout limits
   - Implement rate limiting per user
   - Log all execution requests

## Troubleshooting

### Trigger Not Firing

1. Check trigger is enabled:
   ```bash
   curl http://localhost:8181/api/v1/triggers/{id}
   ```

2. Verify workflow exists and is active:
   ```bash
   curl http://localhost:8181/api/v1/workflows/{workflow_id}
   ```

3. Check server logs:
   ```bash
   docker logs mbflow-api | grep trigger
   ```

4. For cron triggers, verify expression:
   ```bash
   # Use online cron parser or run unit test
   go test ./internal/application/trigger/... -run TestParseCronSchedule
   ```

### Webhook Receiving 401

1. Verify HMAC signature calculation
2. Check secret matches configuration
3. Ensure header name is `X-Webhook-Signature`
4. Review webhook handler logs

### Event Not Triggering Workflow

1. Check Redis connection
2. Verify event type matches trigger config
3. Review event filters
4. Test with direct event publish:
   ```bash
   curl -X POST http://localhost:8181/api/v1/events/publish \
     -H "Content-Type: application/json" \
     -d '{"type": "test.event", "data": {}}'
   ```

## Additional Resources

- [Trigger System Documentation](../../docs/TRIGGERS.md)
- [API Documentation](../../docs/API.md)
- [Architecture Overview](../../docs/ARCHITECTURE_DIAGRAMS.md)
- [GitHub Repository](https://github.com/smilemakc/mbflow)

## Contributing

To add a new example:

1. Create a JSON file with workflow and trigger configuration
2. Test the example thoroughly
3. Add documentation to this README
4. Submit a pull request

## License

These examples are part of the MBFlow project and are subject to the same license.
