package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/gixxerblade/incident-response-mvp/internal/models"
)

// IncidentsHandler handles incident-related API endpoints
type IncidentsHandler struct {
	db *gorm.DB
}

// NewIncidentsHandler creates a new incidents handler
func NewIncidentsHandler(db *gorm.DB) *IncidentsHandler {
	return &IncidentsHandler{db: db}
}

// ListIncidents handles GET /api/v1/incidents
func (h *IncidentsHandler) ListIncidents(c *gin.Context) {
	var incidents []models.Incident

	query := h.db.Order("created_at DESC")

	// Filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by severity
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}

	if err := query.Find(&incidents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch incidents"})
		return
	}

	c.JSON(http.StatusOK, incidents)
}

// GetIncident handles GET /api/v1/incidents/:id
func (h *IncidentsHandler) GetIncident(c *gin.Context) {
	incidentID := c.Param("id")

	var incident models.Incident
	if err := h.db.First(&incident, "incident_id = ?", incidentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch incident"})
		}
		return
	}

	c.JSON(http.StatusOK, incident)
}

// UpdateIncidentRequest represents the request body for updating an incident
type UpdateIncidentRequest struct {
	Status     *string `json:"status"`
	AssignedTo *string `json:"assigned_to"`
	Notes      *string `json:"notes"`
}

// UpdateIncident handles PATCH /api/v1/incidents/:id
func (h *IncidentsHandler) UpdateIncident(c *gin.Context) {
	incidentID := c.Param("id")

	var incident models.Incident
	if err := h.db.First(&incident, "incident_id = ?", incidentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch incident"})
		}
		return
	}

	var req UpdateIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields if provided
	if req.Status != nil {
		incident.Status = models.IncidentStatus(*req.Status)
	}
	if req.AssignedTo != nil {
		incident.AssignedTo = req.AssignedTo
	}
	if req.Notes != nil {
		if incident.Notes != "" {
			incident.Notes += "\n" + *req.Notes
		} else {
			incident.Notes = *req.Notes
		}
	}

	if err := h.db.Save(&incident).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update incident"})
		return
	}

	c.JSON(http.StatusOK, incident)
}

// ResolveIncident handles POST /api/v1/incidents/:id/resolve
func (h *IncidentsHandler) ResolveIncident(c *gin.Context) {
	incidentID := c.Param("id")

	var incident models.Incident
	if err := h.db.First(&incident, "incident_id = ?", incidentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch incident"})
		}
		return
	}

	incident.Status = models.StatusResolved
	if err := h.db.Save(&incident).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve incident"})
		return
	}

	c.JSON(http.StatusOK, incident)
}
