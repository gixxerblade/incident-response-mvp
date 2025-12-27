package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// Playbook represents a response playbook loaded from YAML
type Playbook struct {
	Playbook struct {
		ID          string          `yaml:"id"`
		Name        string          `yaml:"name"`
		Description string          `yaml:"description"`
		Version     string          `yaml:"version"`
		Inputs      []PlaybookInput `yaml:"inputs"`
		Steps       []PlaybookStep  `yaml:"steps"`
	} `yaml:"playbook"`
}

// PlaybookInput represents a playbook input parameter
type PlaybookInput struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
}

// PlaybookStep represents a step in a playbook
type PlaybookStep struct {
	ID         string                 `yaml:"id"`
	Name       string                 `yaml:"name"`
	Action     string                 `yaml:"action"`
	Parameters map[string]interface{} `yaml:"parameters"`
	OnFailure  string                 `yaml:"on_failure"`
	Condition  string                 `yaml:"condition"`
}

// Orchestrator handles playbook execution
type Orchestrator struct {
	db        *gorm.DB
	playbooks map[string]Playbook
	actions   *ActionRegistry
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(db *gorm.DB, actions *ActionRegistry) *Orchestrator {
	return &Orchestrator{
		db:        db,
		playbooks: make(map[string]Playbook),
		actions:   actions,
	}
}

// LoadPlaybooks loads all YAML playbooks from the specified directory
func (o *Orchestrator) LoadPlaybooks(playbooksDir string) error {
	files, err := filepath.Glob(filepath.Join(playbooksDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob playbooks: %w", err)
	}

	files2, err := filepath.Glob(filepath.Join(playbooksDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to glob playbooks: %w", err)
	}
	files = append(files, files2...)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Warning: failed to read playbook file %s: %v", file, err)
			continue
		}

		var playbook Playbook
		if err := yaml.Unmarshal(data, &playbook); err != nil {
			log.Printf("Warning: failed to parse playbook file %s: %v", file, err)
			continue
		}

		o.playbooks[playbook.Playbook.ID] = playbook
		log.Printf("Loaded playbook: %s (%s)", playbook.Playbook.ID, playbook.Playbook.Name)
	}

	log.Printf("Loaded %d playbooks", len(o.playbooks))
	return nil
}

// ExecutePlaybook executes a playbook with the given inputs
func (o *Orchestrator) ExecutePlaybook(playbookID string, inputs map[string]interface{}) error {
	playbook, ok := o.playbooks[playbookID]
	if !ok {
		return fmt.Errorf("playbook not found: %s", playbookID)
	}

	log.Printf("Executing playbook: %s (%s)", playbookID, playbook.Playbook.Name)

	// Validate required inputs
	for _, input := range playbook.Playbook.Inputs {
		if input.Required {
			if _, ok := inputs[input.Name]; !ok {
				return fmt.Errorf("missing required input: %s", input.Name)
			}
		}
	}

	// Execution context holds inputs and step outputs
	context := make(map[string]interface{})
	context["inputs"] = inputs

	// Execute steps sequentially
	for _, step := range playbook.Playbook.Steps {
		log.Printf("Executing step: %s - %s", step.ID, step.Name)

		// Interpolate variables in parameters
		interpolatedParams := o.interpolateParameters(step.Parameters, context)

		// Execute the action
		result, err := o.actions.Execute(step.Action, interpolatedParams)
		if err != nil {
			log.Printf("Step %s failed: %v", step.ID, err)

			// Handle failure based on on_failure policy
			if step.OnFailure == "abort" || step.OnFailure == "" {
				return fmt.Errorf("step %s failed: %w", step.ID, err)
			} else if step.OnFailure == "continue" {
				log.Printf("Continuing after failure in step %s", step.ID)
			}
		}

		// Store step result in context
		if context["steps"] == nil {
			context["steps"] = make(map[string]interface{})
		}
		context["steps"].(map[string]interface{})[step.ID] = map[string]interface{}{
			"output": result,
			"error":  err,
		}

		log.Printf("Step %s completed", step.ID)
	}

	log.Printf("Playbook %s execution completed", playbookID)
	return nil
}

// interpolateParameters replaces template variables in parameters
func (o *Orchestrator) interpolateParameters(params map[string]interface{}, context map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range params {
		switch v := value.(type) {
		case string:
			result[key] = o.interpolateString(v, context)
		case map[string]interface{}:
			result[key] = o.interpolateParameters(v, context)
		default:
			result[key] = value
		}
	}

	return result
}

// interpolateString replaces {{ variable }} patterns in a string
func (o *Orchestrator) interpolateString(s string, context map[string]interface{}) string {
	// Simple template interpolation for {{ inputs.field }} and {{ steps.step-1.output }}
	result := s

	// Find all {{ ... }} patterns
	start := strings.Index(result, "{{")
	for start != -1 {
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		end += start

		// Extract variable path
		varPath := strings.TrimSpace(result[start+2 : end])
		value := o.resolveVariable(varPath, context)

		// Replace in string
		result = result[:start] + fmt.Sprintf("%v", value) + result[end+2:]

		// Find next occurrence
		start = strings.Index(result, "{{")
	}

	return result
}

// resolveVariable resolves a variable path like "inputs.incident_id" or "steps.step-1.output"
func (o *Orchestrator) resolveVariable(path string, context map[string]interface{}) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = context

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return path // Return the original path if not found
		}
	}

	return current
}
