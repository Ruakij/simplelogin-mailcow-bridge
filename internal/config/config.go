package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config stores the application configuration
type Config struct {
	Port                      int
	MailcowAdminAPIURL        string
	MailcowAdminAPIKey        string
	MailcowAuthMethod         string
	MailcowServerAddress      string
	MailcowAliasSOGoVisible   bool
	MailcowAliasAllowSendAs   bool
	MailcowAliasPublicComment string
	AliasValidityPeriod       int
	AliasGenerationPattern    string
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
	if err := checkRequiredEnvKeys(
		"MAILCOW_ADMIN_API_URL",
		"MAILCOW_ADMIN_API_KEY",
		"MAILCOW_SERVER_ADDRESS",
	); err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:                      getEnvInt("PORT", 8080),
		MailcowAdminAPIURL:        getEnvString("MAILCOW_ADMIN_API_URL", ""),
		MailcowAdminAPIKey:        getEnvString("MAILCOW_ADMIN_API_KEY", ""),
		MailcowAuthMethod:         getEnvString("MAILCOW_AUTH_METHOD", "IMAP"),
		MailcowServerAddress:      getEnvString("MAILCOW_SERVER_ADDRESS", ""),
		MailcowAliasSOGoVisible:   getEnvBool("MAILCOW_ALIAS_SOGO_VISIBLE", true),
		MailcowAliasAllowSendAs:   getEnvBool("MAILCOW_ALIAS_ALLOW_SEND_AS", true),
		MailcowAliasPublicComment: getEnvString("MAILCOW_ALIAS_PUBLIC_COMMENT", "simplelogin-mailcow-bridge"),
		AliasValidityPeriod:       getEnvInt("ALIAS_VALIDITY_PERIOD", 10),
		AliasGenerationPattern:    getEnvString("ALIAS_GENERATION_PATTERN", "{firstname}.{lastname}@%d"),
		AuthCacheTTL:              getEnvInt("AUTH_CACHE_TTL", 300),
		LogLevel:                  getEnvString("LOG_LEVEL", "INFO"),
		LogColorize:               getEnvBool("LOG_COLOR", true),
		CORSAllowOrigin:           getEnvString("CORS_ALLOW_ORIGIN", ""),
	}

	return cfg, nil
}

// Helper functions
func getEnvString(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func checkRequiredEnvKeys(keys ...string) error {
	missingKeys := make([]string, 0)

	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("required environment variables not set: %s", strings.Join(missingKeys, ", "))
	}

	return nil
}
