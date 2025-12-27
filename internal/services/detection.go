package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/gixxerblade/incident-response-mvp/internal/models"
)

// Rule represents a detection rule loaded from YAML
type Rule struct {
	Rule struct {
		ID          string   `yaml:"id"`
		Name        string   `yaml:"name"`
		Description string   `yaml:"description"`
		Category    string   `yaml:"category"`
		Severity    string   `yaml:"severity"`
		Enabled     bool     `yaml:"enabled"`
		Conditions  []Condition `yaml:"conditions"`
		Actions     []RuleAction `yaml:"actions"`
	} `yaml:"rule"`
}

// Condition represents a rule condition
type Condition struct {
	Field      string      `yaml:"field"`
	Operator   string      `yaml:"operator"`
	Value      interface{} `yaml:"value"`
	Values     []string    `yaml:"values"`
	Pattern    string      `yaml:"pattern"`
	Threshold  int         `yaml:"threshold"`
	TimeWindow int         `yaml:"timewindow"`
	CountField string      `yaml:"count_field"`
}

// RuleAction represents an action to take when a rule matches
type RuleAction struct {
	Type      string      `yaml:"type"`
	Priority  string      `yaml:"priority"`
	Playbook  string      `yaml:"playbook"`
	Channel   string      `yaml:"channel"`
	Channels  []string    `yaml:"channels"`
	Message   string      `yaml:"message"`
	Duration  interface{} `yaml:"duration"`
}

// DetectionEngine handles rule evaluation and detection
type DetectionEngine struct {
	db    *gorm.DB
	rules []Rule
}

// NewDetectionEngine creates a new detection engine
func NewDetectionEngine(db *gorm.DB) *DetectionEngine {
	return &DetectionEngine{
		db:    db,
		rules: []Rule{},
	}
}

// LoadRules loads all YAML rules from the specified directory
func (de *DetectionEngine) LoadRules(rulesDir string) error {
	files, err := filepath.Glob(filepath.Join(rulesDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob rules: %w", err)
	}

	files2, err := filepath.Glob(filepath.Join(rulesDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to glob rules: %w", err)
	}
	files = append(files, files2...)

	de.rules = []Rule{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Warning: failed to read rule file %s: %v", file, err)
			continue
		}

		var rule Rule
		if err := yaml.Unmarshal(data, &rule); err != nil {
			log.Printf("Warning: failed to parse rule file %s: %v", file, err)
			continue
		}

		if rule.Rule.Enabled {
			de.rules = append(de.rules, rule)
			log.Printf("Loaded rule: %s (%s)", rule.Rule.ID, rule.Rule.Name)
		}
	}

	log.Printf("Loaded %d enabled rules", len(de.rules))
	return nil
}

// EvaluateEvent evaluates an event against all loaded rules
func (de *DetectionEngine) EvaluateEvent(event *models.Event) error {
	log.Printf("Evaluating event %s against %d rules", event.EventID, len(de.rules))

	// Parse normalized data
	var normalized map[string]interface{}
	if err := json.Unmarshal([]byte(event.Normalized), &normalized); err != nil {
		return fmt.Errorf("failed to parse normalized data: %w", err)
	}

	for _, rule := range de.rules {
		if de.matchesRule(event, normalized, rule) {
			log.Printf("Event %s matched rule %s", event.EventID, rule.Rule.ID)
			if err := de.executeRuleActions(event, rule); err != nil {
				log.Printf("Error executing rule actions: %v", err)
			}
		}
	}

	// Mark event as processed
	now := time.Now()
	event.ProcessedAt = &now
	de.db.Save(event)

	return nil
}

// matchesRule checks if an event matches a rule's conditions
func (de *DetectionEngine) matchesRule(event *models.Event, normalized map[string]interface{}, rule Rule) bool {
	for _, condition := range rule.Rule.Conditions {
		if !de.evaluateCondition(event, normalized, condition) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (de *DetectionEngine) evaluateCondition(event *models.Event, normalized map[string]interface{}, cond Condition) bool {
	// Get the field value
	var fieldValue interface{}
	switch cond.Field {
	case "event_type":
		fieldValue = event.EventType
	case "source":
		fieldValue = event.Source
	case "severity":
		fieldValue = string(event.Severity)
	default:
		// Try to get from normalized data
		fieldValue = getNestedField(normalized, cond.Field)
	}

	switch cond.Operator {
	case "equals":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", cond.Value)

	case "in":
		strValue := fmt.Sprintf("%v", fieldValue)
		for _, v := range cond.Values {
			if strValue == v {
				return true
			}
		}
		return false

	case "greater_than":
		// Simple numeric comparison
		if num, ok := fieldValue.(float64); ok {
			if threshold, ok := cond.Value.(float64); ok {
				return num > threshold
			}
		}
		return false

	case "regex":
		strValue := fmt.Sprintf("%v", fieldValue)
		matched, err := regexp.MatchString(cond.Pattern, strValue)
		if err != nil {
			log.Printf("Regex error: %v", err)
			return false
		}
		return matched

	case "count", "count_distinct":
		return de.evaluateCountCondition(event, cond)

	default:
		log.Printf("Unknown operator: %s", cond.Operator)
		return false
	}
}

// evaluateCountCondition evaluates time-windowed count conditions
func (de *DetectionEngine) evaluateCountCondition(event *models.Event, cond Condition) bool {
	// Calculate time window
	windowStart := time.Now().Add(-time.Duration(cond.TimeWindow) * time.Second)

	// Query events in the time window
	var count int64
	query := de.db.Model(&models.Event{}).
		Where("timestamp >= ?", windowStart).
		Where(cond.Field+" = ?", event.EventType)

	if cond.Operator == "count_distinct" && cond.CountField != "" {
		query = query.Distinct(cond.CountField)
	}

	query.Count(&count)

	return int(count) >= cond.Threshold
}

// executeRuleActions executes the actions specified by a rule
func (de *DetectionEngine) executeRuleActions(event *models.Event, rule Rule) error {
	for _, action := range rule.Rule.Actions {
		switch action.Type {
		case "create_incident":
			if err := de.createIncident(event, rule, action); err != nil {
				log.Printf("Failed to create incident: %v", err)
			}

		case "execute_playbook":
			log.Printf("Triggering playbook: %s for event %s", action.Playbook, event.EventID)
			// Playbook execution will be handled by orchestrator

		case "notify":
			de.sendNotification(event, rule, action)

		default:
			log.Printf("Unknown action type: %s", action.Type)
		}
	}
	return nil
}

// createIncident creates an incident from a rule match
func (de *DetectionEngine) createIncident(event *models.Event, rule Rule, action RuleAction) error {
	severity := models.SeverityMedium
	switch strings.ToLower(rule.Rule.Severity) {
	case "critical":
		severity = models.SeverityCritical
	case "high":
		severity = models.SeverityHigh
	case "medium":
		severity = models.SeverityMedium
	case "low":
		severity = models.SeverityLow
	}

	incident := &models.Incident{
		Status:          models.StatusOpen,
		Severity:        severity,
		Category:        rule.Rule.Category,
		Title:           rule.Rule.Name,
		Description:     fmt.Sprintf("%s\nTriggered by event: %s", rule.Rule.Description, event.EventID),
		TriggeredByRule: rule.Rule.ID,
		RelatedEvents:   fmt.Sprintf("[\"%s\"]", event.EventID),
	}

	if err := de.db.Create(incident).Error; err != nil {
		return fmt.Errorf("failed to create incident: %w", err)
	}

	log.Printf("Created incident %s for rule %s", incident.IncidentID, rule.Rule.ID)
	return nil
}

// sendNotification sends a notification
func (de *DetectionEngine) sendNotification(event *models.Event, rule Rule, action RuleAction) {
	message := action.Message
	if message == "" {
		message = fmt.Sprintf("Rule '%s' triggered by event %s", rule.Rule.Name, event.EventID)
	}

	// For MVP, just log the notification
	channel := action.Channel
	if channel == "" && len(action.Channels) > 0 {
		channel = action.Channels[0]
	}

	log.Printf("[NOTIFICATION] [%s] %s", channel, message)
}

// getNestedField retrieves a nested field from a map using dot notation
func getNestedField(data map[string]interface{}, field string) interface{} {
	parts := strings.Split(field, ".")
	var current interface{} = data

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}
