package config

import (
	"log"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Application
	AppName     string `mapstructure:"APP_NAME"`
	AppVersion  string `mapstructure:"APP_VERSION"`
	Environment string `mapstructure:"ENVIRONMENT"`
	Debug       bool   `mapstructure:"DEBUG"`

	// API
	APIPrefix string `mapstructure:"API_PREFIX"`
	APIHost   string `mapstructure:"API_HOST"`
	APIPort   string `mapstructure:"API_PORT"`

	// Database
	DatabaseURL  string `mapstructure:"DATABASE_URL"`
	DatabaseEcho bool   `mapstructure:"DATABASE_ECHO"`

	// Detection
	RuleScanInterval   int `mapstructure:"RULE_SCAN_INTERVAL"`
	CorrelationWindow  int `mapstructure:"CORRELATION_WINDOW"`

	// Orchestration
	PlaybookTimeout    int `mapstructure:"PLAYBOOK_TIMEOUT"`
	MaxPlaybookRetries int `mapstructure:"MAX_PLAYBOOK_RETRIES"`

	// Logging
	LogLevel  string `mapstructure:"LOG_LEVEL"`
	LogFormat string `mapstructure:"LOG_FORMAT"`

	// Paths
	RulesDir     string `mapstructure:"RULES_DIR"`
	PlaybooksDir string `mapstructure:"PLAYBOOKS_DIR"`
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	// Set defaults
	viper.SetDefault("APP_NAME", "Incident Response Agent")
	viper.SetDefault("APP_VERSION", "0.1.0")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("DEBUG", true)

	viper.SetDefault("API_PREFIX", "/api/v1")
	viper.SetDefault("API_HOST", "0.0.0.0")
	viper.SetDefault("API_PORT", "8000")

	viper.SetDefault("DATABASE_URL", "./data/incidents.db")
	viper.SetDefault("DATABASE_ECHO", false)

	viper.SetDefault("RULE_SCAN_INTERVAL", 60)
	viper.SetDefault("CORRELATION_WINDOW", 300)

	viper.SetDefault("PLAYBOOK_TIMEOUT", 3600)
	viper.SetDefault("MAX_PLAYBOOK_RETRIES", 3)

	viper.SetDefault("LOG_LEVEL", "INFO")
	viper.SetDefault("LOG_FORMAT", "json")

	viper.SetDefault("RULES_DIR", "./data/rules")
	viper.SetDefault("PLAYBOOKS_DIR", "./data/playbooks")

	// Read from .env file if it exists
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return nil, err
		}
		// Config file not found; using defaults and environment variables
		log.Println("No .env file found, using defaults and environment variables")
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
