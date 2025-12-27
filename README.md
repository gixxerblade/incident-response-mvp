# Incident Response MVP

An automated incident response system that detects security events, evaluates them against detection rules, and executes automated response playbooks.

## Features

- **Event Ingestion**: REST API for ingesting security events
- **Rule-Based Detection**: YAML-based detection rules with time-windowed correlation
- **Automated Response**: Playbook orchestration with variable interpolation
- **Incident Management**: Track and manage security incidents
- **Action System**: Extensible action framework for automated responses

## Architecture

```
Event → Detection Engine → Incident → Playbook → Actions
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- OR Go 1.23+ (for local development)

### Using Docker (Recommended)

1. **Clone the repository**:

   ```bash
   git clone https://github.com/gixxerblade/incident-response-mvp.git
   cd incident-response-mvp
   ```

2. **Start the service**:

   ```bash
   docker-compose up -d
   ```

3. **Verify the service is running**:

   ```bash
   curl http://localhost:8000/health
   ```

4. **Send a test event**:

   ```bash
   curl -X POST http://localhost:8000/api/v1/events \
     -H "Content-Type: application/json" \
     -d '{
       "event_type": "authentication_failed",
       "source": "ssh-server",
       "severity": "medium",
       "normalized": {
         "source_ip": "192.168.1.100",
         "username": "admin"
       }
     }'
   ```

5. **Check for incidents**:

   ```bash
   curl http://localhost:8000/api/v1/incidents
   ```

6. **View logs**:

   ```bash
   docker-compose logs -f api
   ```

### Local Development

1. **Install dependencies**:

   ```bash
   go mod download
   ```

2. **Copy environment file**:

   ```bash
   cp .env.example .env
   ```

3. **Run the server**:

   ```bash
   go run cmd/server/main.go
   ```

## Demo Scenarios

### Scenario 1: Brute Force Attack Detection

Send 5 failed login events from the same IP to trigger the brute force detection rule:

```bash
for i in {1..5}; do
  curl -X POST http://localhost:8000/api/v1/events \
    -H "Content-Type: application/json" \
    -d '{
      "event_type": "authentication_failed",
      "source": "ssh-server",
      "severity": "medium",
      "normalized": {
        "source_ip": "192.168.1.100",
        "username": "admin"
      }
    }'
  sleep 1
done
```

This will:

1. Create an incident
2. Execute the `brute-force-response` playbook
3. Simulate blocking the IP address
4. Log the action
5. Update the incident status to "contained"
6. Send a notification

### Scenario 2: Port Scan Detection

Send events simulating a port scan:

```bash
for port in {22,23,80,443,3306,5432,6379,8080,8443,9200,27017,3000,5000,5001,5002,5003,5004,5005,5006,5007}; do
  curl -X POST http://localhost:8000/api/v1/events \
    -H "Content-Type: application/json" \
    -d "{
      \"event_type\": \"network_connection\",
      \"source\": \"firewall\",
      \"severity\": \"info\",
      \"normalized\": {
        \"source_ip\": \"192.168.1.50\",
        \"destination_port\": $port
      }
    }"
done
```

## API Endpoints

### Events

- `POST /api/v1/events` - Ingest a new event
- `GET /api/v1/events` - List events (supports filtering)
- `GET /api/v1/events/:id` - Get event details

### Incidents

- `GET /api/v1/incidents` - List incidents (supports filtering)
- `GET /api/v1/incidents/:id` - Get incident details
- `PATCH /api/v1/incidents/:id` - Update incident
- `POST /api/v1/incidents/:id/resolve` - Resolve incident

### System

- `GET /health` - Health check
- `GET /api/v1/stats` - System statistics

## Detection Rules

Rules are defined in YAML format in `data/rules/`. The MVP includes 3 sample rules:

### auth-001: Brute Force Detection

- Triggers on 5+ failed login attempts from the same IP within 5 minutes
- Creates high-severity incident
- Executes brute force response playbook

### net-001: Port Scan Detection

- Triggers on 20+ distinct ports accessed from same IP within 1 minute
- Creates high-severity incident
- Executes port scan response playbook

### mal-001: Suspicious Process Detection

- Detects processes with random hex names spawned by cmd.exe/powershell.exe
- Creates high-severity incident
- Sends notification

## Playbooks

Playbooks are defined in YAML format in `data/playbooks/`. The MVP includes 2 sample playbooks:

### brute-force-response

1. Block source IP (simulated)
2. Log blocking action
3. Update incident status to "contained"
4. Send notification

### port-scan-response

1. Log port scan activity
2. Block scanner IP (simulated)
3. Send notification to security team
4. Mark incident as "investigating"

## Actions

The MVP implements 5 actions:

- `create_incident` - Create a new incident
- `notify` - Send notification (console/webhook)
- `block_ip` - Simulate IP blocking (logged, not enforced)
- `log_action` - Log detailed activity
- `update_incident` - Update incident status/metadata

## Configuration

Configuration can be set via environment variables or `.env` file:

```bash
# API
API_HOST=0.0.0.0
API_PORT=8000

# Database
DATABASE_URL=./data/incidents.db

# Detection
RULE_SCAN_INTERVAL=60
CORRELATION_WINDOW=300

# Paths
RULES_DIR=./data/rules
PLAYBOOKS_DIR=./data/playbooks
```

## Project Structure

```
incident-response-mvp/
├── cmd/
│   └── server/          # API server main
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database setup
│   ├── models/          # GORM models
│   ├── handlers/        # HTTP handlers
│   └── services/        # Business logic
│       ├── detection.go    # Detection engine
│       ├── orchestrator.go # Playbook executor
│       └── actions.go      # Action implementations
├── data/
│   ├── rules/           # Detection rules
│   └── playbooks/       # Response playbooks
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o incident-response-server ./cmd/server
```

### Adding New Rules

Create a YAML file in `data/rules/`:

```yaml
rule:
  id: custom-001
  name: "My Custom Rule"
  description: "Description"
  category: custom
  severity: medium
  enabled: true

  conditions:
    - field: event_type
      operator: equals
      value: "my_event_type"

  actions:
    - type: create_incident
      priority: medium
```

### Adding New Playbooks

Create a YAML file in `data/playbooks/`:

```yaml
playbook:
  id: my-response
  name: "My Response Playbook"
  description: "Description"
  version: "1.0"

  inputs:
    - name: incident_id
      required: true

  steps:
    - id: step-1
      name: "Do Something"
      action: notify
      parameters:
        channel: "console"
        message: "Alert!"
```

## Technology Stack

- **Language**: Go 1.23+
- **Web Framework**: Gin
- **Database**: SQLite (GORM)
- **Config**: Viper
- **YAML**: gopkg.in/yaml.v3
- **Containerization**: Docker

## Roadmap

### Post-MVP (Phase 2)

- PostgreSQL support
- Real integrations (Slack, PagerDuty, email)
- CLI tool with Cobra
- Comprehensive test suite

### Future Enhancements

- ML-based anomaly detection
- Web dashboard UI
- SIEM integrations
- Cloud provider integrations
- Advanced correlation engine
- Compliance reporting

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License

## Support

For issues or questions, please open an issue on GitHub.

---

**Built with** [Go](https://golang.org/) | **Powered by** [Gin](https://gin-gonic.com/) | **Database** [GORM](https://gorm.io/)
