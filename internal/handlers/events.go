package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yourusername/incident-response-mvp/internal/models"
	"github.com/yourusername/incident-response-mvp/internal/services"
)

// EventsHandler handles event-related API endpoints
type EventsHandler struct {
	db              *gorm.DB
	detectionEngine *services.DetectionEngine
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(db *gorm.DB, detectionEngine *services.DetectionEngine) *EventsHandler {
	return &EventsHandler{
		db:              db,
		detectionEngine: detectionEngine,
	}
}

// EventRequest represents the request body for creating an event
type EventRequest struct {
	EventType  string                 `json:"event_type" binding:"required"`
	Source     string                 `json:"source" binding:"required"`
	Severity   string                 `json:"severity"`
	RawData    map[string]interface{} `json:"raw_data"`
	Normalized map[string]interface{} `json:"normalized" binding:"required"`
}

// CreateEvent handles POST /api/v1/events
func (h *EventsHandler) CreateEvent(c *gin.Context) {
	var req EventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default severity
	if req.Severity == "" {
		req.Severity = "info"
	}

	// Convert maps to JSON strings
	normalizedJSON, err := json.Marshal(req.Normalized)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal normalized data"})
		return
	}

	var rawDataJSON string
	if req.RawData != nil {
		rawJSON, err := json.Marshal(req.RawData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal raw data"})
			return
		}
		rawDataJSON = string(rawJSON)
	}

	// Create event
	event := &models.Event{
		Timestamp:  time.Now().UTC(),
		Source:     req.Source,
		EventType:  req.EventType,
		Severity:   models.SeverityLevel(req.Severity),
		RawData:    rawDataJSON,
		Normalized: string(normalizedJSON),
	}

	if err := h.db.Create(event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create event"})
		return
	}

	// Trigger detection engine
	go h.detectionEngine.EvaluateEvent(event)

	c.JSON(http.StatusCreated, event)
}

// ListEvents handles GET /api/v1/events
func (h *EventsHandler) ListEvents(c *gin.Context) {
	var events []models.Event

	query := h.db.Order("timestamp DESC").Limit(100)

	// Filter by event type
	if eventType := c.Query("event_type"); eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	// Filter by severity
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}

	if err := query.Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch events"})
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetEvent handles GET /api/v1/events/:id
func (h *EventsHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("id")

	var event models.Event
	if err := h.db.First(&event, "event_id = ?", eventID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch event"})
		}
		return
	}

	c.JSON(http.StatusOK, event)
}
