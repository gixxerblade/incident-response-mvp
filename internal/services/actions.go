package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/gixxerblade/incident-response-mvp/internal/models"
)

// Action interface defines the contract for all actions
type Action interface {
	Execute(params map[string]interface{}) (interface{}, error)
}

// ActionRegistry manages available actions
type ActionRegistry struct {
	db      *gorm.DB
	actions map[string]Action
}

// NewActionRegistry creates a new action registry
func NewActionRegistry(db *gorm.DB) *ActionRegistry {
	registry := &ActionRegistry{
		db:      db,
		actions: make(map[string]Action),
	}

	// Register all MVP actions
	registry.Register("create_incident", &CreateIncidentAction{db: db})
	registry.Register("notify", &NotifyAction{db: db})
	registry.Register("block_ip", &BlockIPAction{db: db})
	registry.Register("log_action", &LogActionAction{db: db})
	registry.Register("update_incident", &UpdateIncidentAction{db: db})

	// Register advanced actions for real-world playbooks
	registry.Register("ssh_command", &SSHCommandAction{db: db})
	registry.Register("grafana_query", &GrafanaQueryAction{db: db})
	registry.Register("prometheus_query", &PrometheusQueryAction{db: db})
	registry.Register("ai_analyze", &AIAnalyzeAction{db: db})

	// Register generic actions that work with ANY service
	registry.Register("http_request", &HTTPRequestAction{db: db})
	registry.Register("shell_script", &ShellScriptAction{db: db})
	registry.Register("webhook", &WebhookAction{db: db})
	registry.Register("python_script", &PythonScriptAction{db: db})

	return registry
}

// Register registers an action
func (ar *ActionRegistry) Register(name string, action Action) {
	ar.actions[name] = action
	log.Printf("Registered action: %s", name)
}

// Execute executes an action by name
func (ar *ActionRegistry) Execute(actionType string, params map[string]interface{}) (interface{}, error) {
	action, ok := ar.actions[actionType]
	if !ok {
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}

	startTime := time.Now()

	// Log action start
	paramsJSON, _ := json.Marshal(params)
	actionLog := &models.ActionLog{
		ActionType: actionType,
		Status:     models.ActionRunning,
		Parameters: string(paramsJSON),
	}
	ar.db.Create(actionLog)

	// Execute action
	result, err := action.Execute(params)

	// Update action log
	executionTime := int(time.Since(startTime).Milliseconds())
	actionLog.ExecutionTime = executionTime
	now := time.Now()
	actionLog.CompletedAt = &now

	if err != nil {
		actionLog.Status = models.ActionFailed
		errMsg := err.Error()
		actionLog.Error = &errMsg
	} else {
		actionLog.Status = models.ActionCompleted
		if result != nil {
			resultJSON, _ := json.Marshal(result)
			resultStr := string(resultJSON)
			actionLog.Result = &resultStr
		}
	}

	ar.db.Save(actionLog)

	return result, err
}

// CreateIncidentAction creates a new incident
type CreateIncidentAction struct {
	db *gorm.DB
}

func (a *CreateIncidentAction) Execute(params map[string]interface{}) (interface{}, error) {
	priority := getStringParam(params, "priority", "medium")
	title := getStringParam(params, "title", "Automated Incident")
	description := getStringParam(params, "description", "")
	category := getStringParam(params, "category", "")

	severity := models.SeverityMedium
	switch priority {
	case "critical":
		severity = models.SeverityCritical
	case "high":
		severity = models.SeverityHigh
	case "low":
		severity = models.SeverityLow
	}

	incident := &models.Incident{
		Status:      models.StatusOpen,
		Severity:    severity,
		Category:    category,
		Title:       title,
		Description: description,
	}

	if err := a.db.Create(incident).Error; err != nil {
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}

	log.Printf("[ACTION] Created incident: %s", incident.IncidentID)
	return map[string]string{"incident_id": incident.IncidentID}, nil
}

// NotifyAction sends a notification
type NotifyAction struct {
	db *gorm.DB
}

func (a *NotifyAction) Execute(params map[string]interface{}) (interface{}, error) {
	channel := getStringParam(params, "channel", "console")
	message := getStringParam(params, "message", "Notification")

	// For MVP, just log the notification
	log.Printf("[ACTION] [NOTIFICATION] [%s] %s", channel, message)

	// In a real implementation, this would send to Slack, email, PagerDuty, etc.
	return map[string]string{
		"channel": channel,
		"message": message,
		"status":  "sent",
	}, nil
}

// BlockIPAction simulates blocking an IP address
type BlockIPAction struct {
	db *gorm.DB
}

func (a *BlockIPAction) Execute(params map[string]interface{}) (interface{}, error) {
	ipAddress := getStringParam(params, "ip_address", "")
	if ipAddress == "" {
		return nil, fmt.Errorf("ip_address parameter is required")
	}

	duration := getIntParam(params, "duration", 3600)

	// For MVP, this is a simulation - log the action
	log.Printf("[ACTION] [BLOCK_IP] Simulating IP block: %s for %d seconds", ipAddress, duration)

	// In a real implementation, this would integrate with firewalls, security groups, etc.
	return map[string]interface{}{
		"ip_address": ipAddress,
		"duration":   duration,
		"action":     "blocked",
		"simulated":  true,
	}, nil
}

// LogActionAction logs detailed activity
type LogActionAction struct {
	db *gorm.DB
}

func (a *LogActionAction) Execute(params map[string]interface{}) (interface{}, error) {
	message := getStringParam(params, "message", "")
	level := getStringParam(params, "level", "info")

	log.Printf("[ACTION] [LOG] [%s] %s", level, message)

	return map[string]string{
		"logged": "true",
		"level":  level,
	}, nil
}

// UpdateIncidentAction updates an incident's status or metadata
type UpdateIncidentAction struct {
	db *gorm.DB
}

func (a *UpdateIncidentAction) Execute(params map[string]interface{}) (interface{}, error) {
	incidentID := getStringParam(params, "incident_id", "")
	if incidentID == "" {
		return nil, fmt.Errorf("incident_id parameter is required")
	}

	var incident models.Incident
	if err := a.db.First(&incident, "incident_id = ?", incidentID).Error; err != nil {
		return nil, fmt.Errorf("incident not found: %w", err)
	}

	// Update status if provided
	if status, ok := params["status"].(string); ok {
		incident.Status = models.IncidentStatus(status)
	}

	// Update notes if provided
	if notes, ok := params["notes"].(string); ok {
		if incident.Notes != "" {
			incident.Notes += "\n" + notes
		} else {
			incident.Notes = notes
		}
	}

	// Update assigned_to if provided
	if assignedTo, ok := params["assigned_to"].(string); ok {
		incident.AssignedTo = &assignedTo
	}

	if err := a.db.Save(&incident).Error; err != nil {
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	log.Printf("[ACTION] Updated incident: %s", incidentID)
	return map[string]string{"incident_id": incidentID, "status": "updated"}, nil
}

// Helper functions to extract parameters

func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultValue
}

func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if val, ok := params[key].(int); ok {
		return val
	}
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

// SSHCommandAction executes SSH commands on remote hosts
type SSHCommandAction struct {
	db *gorm.DB
}

func (a *SSHCommandAction) Execute(params map[string]interface{}) (interface{}, error) {
	host := getStringParam(params, "host", "")
	command := getStringParam(params, "command", "")
	description := getStringParam(params, "description", "")

	if host == "" || command == "" {
		return nil, fmt.Errorf("host and command parameters are required")
	}

	// For MVP, simulate SSH command execution
	log.Printf("[ACTION] [SSH] Would execute on %s: %s", host, command)
	log.Printf("[ACTION] [SSH] Description: %s", description)

	// In production, this would use crypto/ssh to actually execute the command
	// For now, return simulated output
	return map[string]interface{}{
		"host":        host,
		"command":     command,
		"output":      "Simulated command output - implement real SSH client for production",
		"exit_code":   0,
		"simulated":   true,
		"description": description,
	}, nil
}

// GrafanaQueryAction queries Grafana dashboards
type GrafanaQueryAction struct {
	db *gorm.DB
}

func (a *GrafanaQueryAction) Execute(params map[string]interface{}) (interface{}, error) {
	dashboard := getStringParam(params, "dashboard", "")
	environment := getStringParam(params, "environment", "prod")
	metric := getStringParam(params, "metric", "")

	log.Printf("[ACTION] [GRAFANA] Querying dashboard=%s, env=%s, metric=%s", dashboard, environment, metric)

	// In production, this would use Grafana HTTP API
	// For MVP, return simulated metrics
	return map[string]interface{}{
		"dashboard":   dashboard,
		"environment": environment,
		"metric":      metric,
		"value":       42.5,
		"trend":       "stable",
		"simulated":   true,
	}, nil
}

// PrometheusQueryAction queries Prometheus
type PrometheusQueryAction struct {
	db *gorm.DB
}

func (a *PrometheusQueryAction) Execute(params map[string]interface{}) (interface{}, error) {
	host := getStringParam(params, "host", "")
	query := getStringParam(params, "query", "")

	log.Printf("[ACTION] [PROMETHEUS] Query on %s: %s", host, query)

	// In production, would use Prometheus HTTP API
	return map[string]interface{}{
		"host":      host,
		"query":     query,
		"alerts":    []string{},
		"simulated": true,
	}, nil
}

// AIAnalyzeAction uses Claude API for intelligent incident analysis
type AIAnalyzeAction struct {
	db *gorm.DB
}

func (a *AIAnalyzeAction) Execute(params map[string]interface{}) (interface{}, error) {
	incidentID := getStringParam(params, "incident_id", "")
	context := getStringParam(params, "context", "")
	model := getStringParam(params, "model", "claude-sonnet-4")

	if context == "" {
		return nil, fmt.Errorf("context parameter is required for AI analysis")
	}

	log.Printf("[ACTION] [AI_ANALYZE] Analyzing incident %s with %s", incidentID, model)
	log.Printf("[ACTION] [AI_ANALYZE] Context: %s", context)

	// TODO: Integrate with Claude API
	// For MVP, return simulated AI analysis
	// In production, this would call the Anthropic API:
	//
	// import anthropic "github.com/anthropics/anthropic-sdk-go"
	//
	// client := anthropic.NewClient()
	// response, err := client.Messages.New(ctx, anthropic.MessageNewParams{
	//     Model: anthropic.F(model),
	//     Messages: []anthropic.MessageParam{
	//         anthropic.NewUserMessage(anthropic.NewTextBlock(context)),
	//     },
	// })

	return map[string]interface{}{
		"incident_id": incidentID,
		"model":       model,
		"root_cause":  "Worker process crashed due to memory pressure (simulated)",
		"recommendation": "Restart workers and increase memory limits by 100M",
		"confidence":     0.85,
		"reasoning":      "Based on log patterns and resource metrics",
		"simulated":      true,
		"note":           "Implement real Claude API integration for production use",
	}, nil
}
