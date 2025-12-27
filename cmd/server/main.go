package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/yourusername/incident-response-mvp/internal/config"
	"github.com/yourusername/incident-response-mvp/internal/database"
	"github.com/yourusername/incident-response-mvp/internal/handlers"
	"github.com/yourusername/incident-response-mvp/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	db := database.GetDB()

	// Initialize services
	detectionEngine := services.NewDetectionEngine(db)
	if err := detectionEngine.LoadRules(cfg.RulesDir); err != nil {
		log.Printf("Warning: Failed to load rules: %v", err)
	}

	actionRegistry := services.NewActionRegistry(db)
	orchestrator := services.NewOrchestrator(db, actionRegistry)
	if err := orchestrator.LoadPlaybooks(cfg.PlaybooksDir); err != nil {
		log.Printf("Warning: Failed to load playbooks: %v", err)
	}

	// Initialize handlers
	eventsHandler := handlers.NewEventsHandler(db, detectionEngine)
	incidentsHandler := handlers.NewIncidentsHandler(db)

	// Set up Gin router
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": cfg.AppName,
			"version": cfg.AppVersion,
		})
	})

	// API v1 routes
	v1 := router.Group(cfg.APIPrefix)
	{
		// Events
		events := v1.Group("/events")
		{
			events.POST("", eventsHandler.CreateEvent)
			events.GET("", eventsHandler.ListEvents)
			events.GET("/:id", eventsHandler.GetEvent)
		}

		// Incidents
		incidents := v1.Group("/incidents")
		{
			incidents.GET("", incidentsHandler.ListIncidents)
			incidents.GET("/:id", incidentsHandler.GetIncident)
			incidents.PATCH("/:id", incidentsHandler.UpdateIncident)
			incidents.POST("/:id/resolve", incidentsHandler.ResolveIncident)
		}

		// Stats endpoint
		v1.GET("/stats", func(c *gin.Context) {
			var eventCount, incidentCount, actionCount int64
			db.Table("events").Count(&eventCount)
			db.Table("incidents").Count(&incidentCount)
			db.Table("action_logs").Count(&actionCount)

			c.JSON(200, gin.H{
				"events":    eventCount,
				"incidents": incidentCount,
				"actions":   actionCount,
			})
		})
	}

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.APIHost, cfg.APIPort)
	log.Printf("Starting %s v%s on %s", cfg.AppName, cfg.AppVersion, addr)
	log.Printf("Swagger UI available at http://%s/swagger/index.html (when implemented)", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
