package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/yourusername/incident-response-mvp/internal/config"
	"github.com/yourusername/incident-response-mvp/internal/models"
)

// DB is the global database instance
var DB *gorm.DB

// InitDatabase initializes the database connection and runs migrations
func InitDatabase(cfg *config.Config) error {
	// Create database directory if it doesn't exist
	dbPath := cfg.DatabaseURL
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure GORM logger
	gormConfig := &gorm.Config{}
	if cfg.DatabaseEcho {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run auto-migrations
	if err := db.AutoMigrate(
		&models.Event{},
		&models.Incident{},
		&models.ActionLog{},
	); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	DB = db
	log.Println("Database initialized successfully")
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// CloseDatabase closes the database connection
func CloseDatabase() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
