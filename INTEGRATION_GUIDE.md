# Integration Guide: Connect to ANY Service

## The Problem with Custom Actions

**Don't do this:**

```go
// Bad: Need a new action for every service
registry.Register("slack_action", &SlackAction{})
registry.Register("jira_action", &JiraAction{})
registry.Register("pagerduty_action", &PagerDutyAction{})
registry.Register("datadog_action", &DatadogAction{})
// ... 100 more integrations?
```

## The Solution: 4 Generic Actions

We provide **4 powerful generic actions** that can integrate with **any service**:

### 1. `http_request` - Call Any HTTP API

Works with: Slack, Jira, PagerDuty, Datadog, Grafana, Prometheus, GitHub, etc.

```yaml
- id: notify-slack
  action: http_request
  parameters:
    method: "POST"
    url: "https://hooks.slack.com/services/YOUR/WEBHOOK"
    headers:
      Content-Type: "application/json"
    body:
      text: "Alert: {{ inputs.message }}"
      channel: "#incidents"
    timeout: 30
```

**Real Examples:**

| Service | How to Use |
|---------|-----------|
| **Slack** | POST to webhook URL |
| **Jira** | POST to /rest/api/3/issue |
| **PagerDuty** | POST to /v2/enqueue |
| **Datadog** | POST to /api/v1/events |
| **Grafana** | POST to /api/ds/query |
| **Prometheus** | GET /api/v1/alerts |
| **GitHub** | POST to /repos/owner/repo/issues |
| **Zendesk** | POST to /api/v2/tickets.json |

### 2. `shell_script` - Run Any Shell Command

Works with: SSH, Ansible, kubectl, docker, custom CLIs

```yaml
- id: restart-service
  action: shell_script
  parameters:
    script: |
      #!/bin/bash
      ssh admin@server "systemctl restart nginx"
      kubectl rollout restart deployment/api
      docker-compose restart worker
    timeout: 300
    workdir: "/opt/scripts"
```

**Use Cases:**

- SSH to remote servers
- Run Ansible playbooks
- Execute kubectl commands
- Docker operations
- Custom maintenance scripts
- Database queries
- File operations

### 3. `webhook` - Send to Any Webhook

Simplified POST to webhooks (wrapper around http_request)

```yaml
- id: notify-teams
  action: webhook
  parameters:
    url: "https://outlook.office.com/webhook/YOUR_WEBHOOK"
    payload:
      title: "Incident Alert"
      text: "{{ inputs.message }}"
      themeColor: "FF0000"
```

### 4. `python_script` - Run Python Code

Works with: Complex integrations, data processing, ML models

```yaml
- id: analyze-logs
  action: python_script
  parameters:
    script: "./scripts/analyze.py"
    python: "python3"
    args:
      - "{{ inputs.incident_id }}"
      - "--model"
      - "ml-v2"
```

**When to Use:**

- Complex API integrations
- Data processing/analysis
- Machine learning inference
- Custom business logic
- Multi-step workflows

## Real-World Example: Your Async Worker Playbook

Here's how the generic actions work for your actual use case:

```yaml
steps:
  # Check batch logs via SSH
  - action: shell_script
    parameters:
      script: |
        ssh fe-01 "tail -n 50 /var/log/bycore/batch-monitor_rq-bycore_batch.log"

  # Query Grafana
  - action: http_request
    parameters:
      method: "POST"
      url: "https://grafana.example.com/api/ds/query"
      headers:
        Authorization: "Bearer {{ env.GRAFANA_TOKEN }}"
      body:
        queries:
          - expr: "rq_job_duration_seconds"
            range: true

  # Check worker status via Ansible
  - action: shell_script
    parameters:
      script: |
        ansible async_worker_matrix -i hosts.yaml \
          -a "sudo docker-compose ps"

  # Query Prometheus
  - action: http_request
    parameters:
      url: "http://monitor-01:9090/api/v1/query"
      params:
        query: "ALERTS{alertname=~'async.*is_down'}"

  # Page on-call via PagerDuty
  - action: http_request
    parameters:
      method: "POST"
      url: "https://events.pagerduty.com/v2/enqueue"
      body:
        routing_key: "{{ env.PAGERDUTY_KEY }}"
        event_action: "trigger"
```

## Benefits

✅ **No Code Changes** - Add new integrations via YAML
✅ **Infinite Flexibility** - Anything with an API or CLI
✅ **Maintainable** - Update playbooks, not code
✅ **Extensible** - Custom Python scripts for complex logic
✅ **Secure** - Credentials via environment variables

## Environment Variables

Store credentials securely:

```bash
# .env
SLACK_WEBHOOK=https://hooks.slack.com/services/...
JIRA_TOKEN=Basic ABC123...
PAGERDUTY_KEY=R0ABC...
GRAFANA_TOKEN=Bearer XYZ...
GITHUB_TOKEN=ghp_ABC...
```

Reference in playbooks:

```yaml
headers:
  Authorization: "{{ env.GRAFANA_TOKEN }}"
```

## Advanced Patterns

### Pattern 1: Conditional Execution

```yaml
- id: escalate
  action: http_request
  parameters:
    url: "{{ inputs.severity == 'critical' ? env.PAGERDUTY_URL : env.SLACK_URL }}"
  condition: "{{ inputs.confidence < 0.8 }}"
```

### Pattern 2: Chained Requests

```yaml
- id: get-runbook
  action: http_request
  parameters:
    url: "https://wiki.company.com/api/runbooks/{{ inputs.incident_type }}"

- id: execute-runbook
  action: shell_script
  parameters:
    script: "{{ steps.get-runbook.output.body.script }}"
```

### Pattern 3: Error Handling

```yaml
- id: try-restart
  action: shell_script
  parameters:
    script: "systemctl restart app"
  on_failure: continue  # or abort

- id: fallback
  action: http_request
  parameters:
    url: "{{ env.PAGERDUTY_URL }}"
  condition: "{{ steps.try-restart.error }}"
```

## Security Best Practices

1. **Never hardcode credentials** - Use environment variables
2. **Use secrets management** - Vault, AWS Secrets Manager, etc.
3. **Validate inputs** - Sanitize user-provided data
4. **Audit logs** - Every action is logged with parameters
5. **Principle of least privilege** - Limit script permissions

## FAQ

**Q: Can I integrate with service X?**
A: If it has an HTTP API, SSH access, or CLI → Yes!

**Q: What if the service doesn't have an API?**
A: Use `shell_script` to run their CLI or `python_script` for custom code

**Q: How do I handle authentication?**
A: Pass tokens via headers or environment variables

**Q: Can I run complex logic?**
A: Yes! Use `python_script` or `shell_script` for multi-step workflows

**Q: Is this secure?**
A: Yes - credentials via env vars, all actions logged, input validation

## Examples Repository

See `data/playbooks/generic-integration-examples.yaml` for complete examples:

- Slack notifications
- Jira ticket creation
- PagerDuty pages
- Datadog events
- GitHub issues
- Email via SendGrid
- Custom Python scripts
- Shell automation

## Next Steps

1. Copy `generic-integration-examples.yaml`
2. Replace placeholder URLs/tokens with your real values
3. Test with a simple integration (Slack webhook)
4. Build out your custom playbooks
5. No code changes needed!

---

**Remember**: With these 4 generic actions, you can integrate with **any service** without writing a single line of code!
