package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IncidentStatus represents the current status of an incident
type IncidentStatus string

const (
	StatusOpen          IncidentStatus = "open"
	StatusInvestigating IncidentStatus = "investigating"
	StatusContained     IncidentStatus = "contained"
	StatusResolved      IncidentStatus = "resolved"
)

// Incident represents a security incident
type Incident struct {
	IncidentID string         `gorm:"primaryKey;type:varchar(36)" json:"incident_id"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Incident details
	Status      IncidentStatus `gorm:"index;type:varchar(20);not null" json:"status"`
	Severity    SeverityLevel  `gorm:"index;type:varchar(20);not null" json:"severity"`
	Category    string         `gorm:"type:varchar(100)" json:"category"`
	Title       string         `gorm:"type:varchar(500);not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`

	// Relationships
	TriggeredByRule string `gorm:"type:varchar(100)" json:"triggered_by_rule"`
	RelatedEvents   string `gorm:"type:text" json:"related_events"` // JSON array of event IDs
	ActionsTaken    string `gorm:"type:text" json:"actions_taken"`  // JSON array of action IDs

	// Assignment
	AssignedTo *string `gorm:"type:varchar(255)" json:"assigned_to"`

	// Additional metadata
	Notes string `gorm:"type:text" json:"notes"`
}

// BeforeCreate hook to generate UUID
func (i *Incident) BeforeCreate(tx *gorm.DB) error {
	if i.IncidentID == "" {
		i.IncidentID = uuid.New().String()
	}
	if i.Status == "" {
		i.Status = StatusOpen
	}
	return nil
}

// TableName specifies the table name for Incident
func (Incident) TableName() string {
	return "incidents"
}
