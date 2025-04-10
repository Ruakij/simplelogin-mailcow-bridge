package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config stores the application configuration
type Config struct {
	Port                   int
	MailcowAdminAPIURL     string
	MailcowAdminAPIKey     string
	MailcowAuthMethod      string
	MailcowServerAddress   string
	AliasValidityPeriod    int
	AliasGenerationPattern string
}

// LoadConfig loads the configuration from environment variables
func LoadConfig() (*Config, error) {
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		port = 8080 // Default port
	}

	aliasValidityPeriod, err := strconv.Atoi(os.Getenv("ALIAS_VALIDITY_PERIOD"))
	if err != nil {
		aliasValidityPeriod = 10 // Default validity period (years)
	}

	// Get authentication method with IMAP as default
	authMethod := os.Getenv("MAILCOW_AUTH_METHOD")
	if authMethod == "" {
		authMethod = "IMAP" // Default to IMAP if not specified
	}

	cfg := &Config{
		Port:                   port,
		MailcowAdminAPIURL:     os.Getenv("MAILCOW_ADMIN_API_URL"),
		MailcowAdminAPIKey:     os.Getenv("MAILCOW_ADMIN_API_KEY"),
		MailcowAuthMethod:      authMethod,
		MailcowServerAddress:   os.Getenv("MAILCOW_SERVER_ADDRESS"),
		AliasValidityPeriod:    aliasValidityPeriod,
		AliasGenerationPattern: os.Getenv("ALIAS_GENERATION_PATTERN"),
	}

	// Check if required environment variables are set
	if cfg.MailcowAdminAPIURL == "" {
		return nil, fmt.Errorf("MAILCOW_ADMIN_API_URL environment variable not set")
	}
	if cfg.MailcowAdminAPIKey == "" {
		return nil, fmt.Errorf("MAILCOW_ADMIN_API_KEY environment variable not set")
	}
	if cfg.MailcowServerAddress == "" {
		return nil, fmt.Errorf("MAILCOW_SERVER_ADDRESS environment variable not set")
	}
	if cfg.AliasGenerationPattern == "" {
		cfg.AliasGenerationPattern = "squirrel.fenneck@%s" // Default alias generation pattern
	}

	return cfg, nil
}
