package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActionStatus represents the status of an action execution
type ActionStatus string

const (
	ActionPending   ActionStatus = "pending"
	ActionRunning   ActionStatus = "running"
	ActionCompleted ActionStatus = "completed"
	ActionFailed    ActionStatus = "failed"
)

// ActionLog represents a log entry for executed actions
type ActionLog struct {
	ActionID string       `gorm:"primaryKey;type:varchar(36)" json:"action_id"`
	CreatedAt time.Time   `gorm:"autoCreateTime" json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`

	// Action details
	ActionType string       `gorm:"type:varchar(100);not null" json:"action_type"`
	Status     ActionStatus `gorm:"type:varchar(20);not null" json:"status"`

	// Context
	IncidentID  *string `gorm:"type:varchar(36)" json:"incident_id"`
	PlaybookID  *string `gorm:"type:varchar(100)" json:"playbook_id"`
	StepID      *string `gorm:"type:varchar(100)" json:"step_id"`

	// Execution details
	Parameters string  `gorm:"type:text" json:"parameters"` // JSON parameters
	Result     *string `gorm:"type:text" json:"result"`     // JSON result
	Error      *string `gorm:"type:text" json:"error"`      // Error message if failed

	// Metadata
	ExecutionTime int    `json:"execution_time"` // in milliseconds
	Notes         string `gorm:"type:text" json:"notes"`
}

// BeforeCreate hook to generate UUID and set defaults
func (a *ActionLog) BeforeCreate(tx *gorm.DB) error {
	if a.ActionID == "" {
		a.ActionID = uuid.New().String()
	}
	if a.Status == "" {
		a.Status = ActionPending
	}
	return nil
}

// TableName specifies the table name for ActionLog
func (ActionLog) TableName() string {
	return "action_logs"
}
