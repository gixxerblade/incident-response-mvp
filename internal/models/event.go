package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeverityLevel represents event severity
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// Event represents a security event in the system
type Event struct {
	EventID  string        `gorm:"primaryKey;type:varchar(36)" json:"event_id"`
	Timestamp time.Time    `gorm:"index;not null" json:"timestamp"`

	// Event metadata
	Source    string        `gorm:"index;type:varchar(255);not null" json:"source"`
	EventType string        `gorm:"index;type:varchar(100);not null" json:"event_type"`
	Severity  SeverityLevel `gorm:"index;type:varchar(20);not null" json:"severity"`

	// Event data (stored as JSON in SQLite)
	RawData    string `gorm:"type:text" json:"raw_data"`
	Normalized string `gorm:"type:text;not null" json:"normalized"`

	// Timestamps
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}

// BeforeCreate hook to generate UUID
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.EventID == "" {
		e.EventID = uuid.New().String()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	return nil
}

// TableName specifies the table name for Event
func (Event) TableName() string {
	return "events"
}
