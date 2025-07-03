package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	// Auth caching configuration
	AuthCacheTTL int // in seconds, 0 means disabled
	// CORS configuration
	CORSAllowOrigin string
	// Logging configuration
	LogLevel    string
	LogColorize bool
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

	// Auth caching configuration - default to 300 seconds (5 minutes)
	authCacheTTL := 300
	authCacheTTLStr := os.Getenv("AUTH_CACHE_TTL")

	// If explicitly set to 0 or empty string, disable cache
	if authCacheTTLStr == "0" || authCacheTTLStr == "" {
		authCacheTTL = 0
	} else if authCacheTTLStr != "" {
		ttl, err := strconv.Atoi(authCacheTTLStr)
		if err == nil {
			authCacheTTL = ttl
		}
	}

	// Logging configuration
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO" // Default to INFO level if not specified
	}

	// Log colorization, enabled by default, disable with LOG_COLOR=false
	logColorize := true
	logColorStr := os.Getenv("LOG_COLOR")
	if strings.ToLower(logColorStr) == "false" {
		logColorize = false
	}
	
	cfg := &Config{
		Port:                   port,
		MailcowAdminAPIURL:     os.Getenv("MAILCOW_ADMIN_API_URL"),
		MailcowAdminAPIKey:     os.Getenv("MAILCOW_ADMIN_API_KEY"),
		MailcowAuthMethod:      authMethod,
		MailcowServerAddress:   os.Getenv("MAILCOW_SERVER_ADDRESS"),
		AliasValidityPeriod:    aliasValidityPeriod,
		AliasGenerationPattern: os.Getenv("ALIAS_GENERATION_PATTERN"),
		AuthCacheTTL:           authCacheTTL,
		LogLevel:               logLevel,
		LogColorize:            logColorize,
		CORSAllowOrigin:        os.Getenv("CORS_ALLOW_ORIGIN"),
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
		cfg.AliasGenerationPattern = "{firstname}.{lastname}@%s" // Default alias generation pattern
	}

	return cfg, nil
}
