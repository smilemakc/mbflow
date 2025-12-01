# Trigger System Quick Start Guide

Get up and running with MBFlow triggers in 5 minutes.

## Prerequisites

```bash
# Start required services
docker compose up -d postgres redis

# Or manually:
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=mbflow postgres:15
docker run -d -p 6379:6379 redis:7
```

## 1. Start MBFlow Server

```bash
# Set environment variables
export DATABASE_URL="postgres://mbflow:mbflow@localhost:5432/mbflow?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export PORT=8181

# Build and run server
go build -o mbflow-server ./cmd/server
./mbflow-server
```

## 2. Create Your First Workflow

Create a simple workflow that will be triggered:

```bash
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Hello World Workflow",
    "description": "Simple test workflow",
    "status": "active",
    "nodes": [
      {
        "id": "hello",
        "name": "Say Hello",
        "type": "transform",
        "config": {
          "template": {
            "message": "Hello from trigger! Input: {{.input}}"
          }
        }
      }
    ],
    "edges": []
  }'
```

Save the workflow ID from the response.

## 3. Create Triggers

### Option A: Manual Trigger (Simplest)

```bash
WORKFLOW_ID="<your-workflow-id>"

curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"Manual Trigger\",
    \"type\": \"manual\",
    \"enabled\": true,
    \"config\": {}
  }"
```

Save the trigger ID, then execute:

```bash
TRIGGER_ID="<your-trigger-id>"

curl -X POST http://localhost:8181/api/v1/triggers/$TRIGGER_ID/execute \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "test": "Hello from manual trigger!"
    }
  }'
```

### Option B: Interval Trigger (Every 30 seconds)

```bash
curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"Every 30 Seconds\",
    \"type\": \"interval\",
    \"enabled\": true,
    \"config\": {
      \"interval\": \"30s\"
    }
  }"
```

Watch it execute every 30 seconds:

```bash
watch -n 5 'curl -s http://localhost:8181/api/v1/executions?limit=5 | jq'
```

### Option C: Cron Trigger (Every Minute)

```bash
curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"Every Minute\",
    \"type\": \"cron\",
    \"enabled\": true,
    \"config\": {
      \"schedule\": \"0 * * * * *\",
      \"timezone\": \"UTC\"
    }
  }"
```

### Option D: Webhook Trigger

```bash
curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"Webhook Trigger\",
    \"type\": \"webhook\",
    \"enabled\": true,
    \"config\": {
      \"secret\": \"my-webhook-secret-min-32-characters-long\"
    }
  }"
```

Get the trigger ID and test the webhook:

```bash
TRIGGER_ID="<your-trigger-id>"

curl -X POST http://localhost:8181/api/v1/webhooks/$TRIGGER_ID \
  -H "Content-Type: application/json" \
  -d '{
    "data": "Hello from webhook!"
  }'
```

### Option E: Event Trigger

```bash
curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"User Created Event\",
    \"type\": \"event\",
    \"enabled\": true,
    \"config\": {
      \"event_type\": \"user.created\"
    }
  }"
```

Publish an event to trigger it:

```bash
# Using Redis CLI
redis-cli PUBLISH "mbflow:events:user.created" '{
  "type": "user.created",
  "source": "api",
  "data": {"user_id": "123", "email": "test@example.com"},
  "timestamp": "2024-01-01T00:00:00Z"
}'

# Or create an event publishing endpoint in your app
```

## 4. Monitor Executions

```bash
# List recent executions
curl http://localhost:8181/api/v1/executions?limit=10 | jq

# Get specific execution
curl http://localhost:8181/api/v1/executions/<execution-id> | jq

# Filter by workflow
curl http://localhost:8181/api/v1/executions?workflow_id=<workflow-id> | jq

# Filter by status
curl http://localhost:8181/api/v1/executions?status=failed | jq
```

## 5. Manage Triggers

```bash
# List all triggers
curl http://localhost:8181/api/v1/triggers | jq

# Get trigger details
curl http://localhost:8181/api/v1/triggers/<trigger-id> | jq

# Disable trigger
curl -X POST http://localhost:8181/api/v1/triggers/<trigger-id>/disable

# Enable trigger
curl -X POST http://localhost:8181/api/v1/triggers/<trigger-id>/enable

# Update trigger
curl -X PUT http://localhost:8181/api/v1/triggers/<trigger-id> \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "schedule": "0 */5 * * * *"
    }
  }'

# Delete trigger
curl -X DELETE http://localhost:8181/api/v1/triggers/<trigger-id>
```

## Common Patterns

### Daily Report at 9 AM

```bash
curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"Daily Report\",
    \"type\": \"cron\",
    \"enabled\": true,
    \"config\": {
      \"schedule\": \"0 0 9 * * *\",
      \"timezone\": \"America/New_York\"
    }
  }"
```

### Health Check Every 30 Seconds

```bash
curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"Health Check\",
    \"type\": \"interval\",
    \"enabled\": true,
    \"config\": {
      \"interval\": \"30s\"
    }
  }"
```

### Webhook with Security

```bash
curl -X POST http://localhost:8181/api/v1/triggers \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"name\": \"Secure Webhook\",
    \"type\": \"webhook\",
    \"enabled\": true,
    \"config\": {
      \"secret\": \"your-secret-min-32-characters-long\",
      \"ip_whitelist\": [\"192.168.1.0/24\"]
    }
  }"
```

## Troubleshooting

### Trigger Not Firing

```bash
# Check trigger is enabled
curl http://localhost:8181/api/v1/triggers/<trigger-id> | jq '.enabled'

# Check workflow is active
curl http://localhost:8181/api/v1/workflows/<workflow-id> | jq '.status'

# Check server logs
docker logs mbflow-api

# For cron triggers, verify expression
# Use https://crontab.guru/ or our test:
go test ./internal/application/trigger/... -run TestParseCronSchedule -v
```

### Webhook Returns Error

```bash
# Check trigger exists
curl http://localhost:8181/api/v1/webhooks/<trigger-id>

# Test without signature (if no secret configured)
curl -X POST http://localhost:8181/api/v1/webhooks/<trigger-id> \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# Check if IP is whitelisted
curl http://localhost:8181/api/v1/triggers/<trigger-id> | jq '.config.ip_whitelist'
```

### Event Not Triggering

```bash
# Check Redis connection
redis-cli PING

# Verify event type
curl http://localhost:8181/api/v1/triggers?type=event | jq '.[].config.event_type'

# Monitor Redis pub/sub
redis-cli PSUBSCRIBE "mbflow:events:*"

# Publish test event
redis-cli PUBLISH "mbflow:events:test" '{"type":"test","data":{}}'
```

## Next Steps

1. Explore [complete examples](./README.md)
2. Read [full documentation](../../docs/internal/TRIGGERS.md)
3. Set up monitoring and metrics
4. Configure production security settings
5. Implement error handling in workflows

## Tips

- Start with manual triggers for testing
- Use interval triggers for simple periodic tasks
- Use cron triggers for specific schedules
- Always configure webhook secrets in production
- Monitor execution history regularly
- Set up alerts for failed executions

## Resources

- [API Documentation](../../docs/API.md)
- [Trigger Documentation](../../docs/internal/TRIGGERS.md)
- [Examples](./README.md)
- [Architecture](../../docs/internal/ARCHITECTURE_DIAGRAMS.md)

## Getting Help

- Check [troubleshooting guide](../../docs/internal/TRIGGERS.md#troubleshooting)
- Review [examples](./README.md)
- Open GitHub issue
- Join community Discord/Slack

---

Happy triggering! 🚀
