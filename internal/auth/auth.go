package auth

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
	"github.com/emersion/go-imap/client"
)

// AuthCache represents a cached authentication
type AuthCache struct {
	Expiry time.Time
}

// AuthModule is a module for authenticating users against Mailcow
type AuthModule struct {
	method        string
	serverAddress string
	cacheTTL      time.Duration
	cache         map[string]AuthCache
	cacheMutex    sync.RWMutex
	logger        *logger.Logger
}

// NewAuthModule creates a new AuthModule
func NewAuthModule(method, serverAddress string, cacheTTL int) (*AuthModule, error) {
	if method == "" || serverAddress == "" {
		return nil, fmt.Errorf("method and serverAddress must be set")
	}

	// Default to IMAP if method is not specified
	if method == "" {
		method = "IMAP"
	}

	// Convert TTL from seconds to duration
	cacheDuration := time.Duration(cacheTTL) * time.Second

	return &AuthModule{
		method:        method,
		serverAddress: serverAddress,
		cacheTTL:      cacheDuration,
		cache:         make(map[string]AuthCache),
		logger:        logger.WithComponent("Auth"),
	}, nil
}

// IsCacheEnabled returns true if caching is enabled (TTL > 0)
func (a *AuthModule) IsCacheEnabled() bool {
	return a.cacheTTL > 0
}

// hashCredentials creates a secure hash for the credential cache
func hashCredentials(username, password string) string {
	// Combine username and password, then hash
	combined := username + ":" + password
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash)
}

// Authenticate authenticates a user against Mailcow
func (a *AuthModule) Authenticate(username, password string) error {
	// Generate a request ID for logging
	requestID := fmt.Sprintf("AUTH-%d", time.Now().UnixNano())
	log := a.logger.WithRequestID(requestID)

	// Mask username for logging
	maskedUser := username
	if len(maskedUser) > 3 {
		maskedUser = maskedUser[:3] + "***"
	}

	// Check cache if enabled (TTL > 0)
	if a.IsCacheEnabled() {
		credHash := hashCredentials(username, password)
		a.cacheMutex.RLock()
		cacheEntry, found := a.cache[credHash]
		a.cacheMutex.RUnlock()

		if found && time.Now().Before(cacheEntry.Expiry) {
			log.Debug("Using cached authentication for user %s (valid until %s)",
				maskedUser, cacheEntry.Expiry.Format(time.RFC3339))
			return nil // Cached authentication is valid
		}

		if found {
			log.Debug("Cached authentication for user %s has expired, re-authenticating",
				maskedUser)
		}
	}

	log.Info("Starting %s authentication for user %s to server %s",
		strings.ToUpper(a.method), maskedUser, a.serverAddress)

	var err error
	startTime := time.Now()

	switch strings.ToUpper(a.method) {
	case "IMAP":
		err = a.authenticateIMAP(username, password, requestID)
	case "SMTP":
		err = a.authenticateSMTP(username, password, requestID)
	default:
		err = fmt.Errorf("unsupported authentication method: %s", a.method)
		log.Error("%v", err)
		return err
	}

	duration := time.Since(startTime)
	if err != nil {
		log.Error("Authentication failed after %s: %v", logger.FormatDuration(duration), err)
		return err
	}

	// Cache successful authentication if caching is enabled
	if a.IsCacheEnabled() {
		credHash := hashCredentials(username, password)
		expiry := time.Now().Add(a.cacheTTL)

		a.cacheMutex.Lock()
		a.cache[credHash] = AuthCache{
			Expiry: expiry,
		}
		a.cacheMutex.Unlock()

		log.Debug("Cached authentication for user %s (valid until %s)",
			maskedUser, expiry.Format(time.RFC3339))
	}

	log.Info("Authentication successful for user %s (took %s)", maskedUser, logger.FormatDuration(duration))
	return nil
}

// CleanupCache removes expired entries from the cache
func (a *AuthModule) CleanupCache() int {
	if !a.IsCacheEnabled() {
		return 0
	}

	now := time.Now()
	removed := 0

	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	for key, entry := range a.cache {
		if now.After(entry.Expiry) {
			delete(a.cache, key)
			removed++
		}
	}

	if removed > 0 {
		a.logger.Debug("Cleaned up %d expired cache entries", removed)
	}

	return removed
}

// CacheStats returns statistics about the cache
func (a *AuthModule) CacheStats() (int, int) {
	if !a.IsCacheEnabled() {
		return 0, 0
	}

	now := time.Now()
	total := 0
	valid := 0

	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()

	for _, entry := range a.cache {
		total++
		if now.Before(entry.Expiry) {
			valid++
		}
	}

	return total, valid
}

// authenticateIMAP authenticates a user against the IMAP server
func (a *AuthModule) authenticateIMAP(username, password, requestID string) error {
	log := a.logger.WithRequestID(requestID)
	log.Debug("Establishing TLS connection to IMAP server")

	// Connect to server with a timeout
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", a.serverAddress, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		log.Error("Failed to connect to IMAP server: %v", err)
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	log.Debug("TLS connection established")

	// Create a new IMAP client
	log.Debug("Creating IMAP client")
	c, err := client.New(conn)
	if err != nil {
		log.Error("Failed to create IMAP client: %v", err)
		return fmt.Errorf("failed to create IMAP client: %w", err)
	}

	// Login
	log.Debug("Attempting IMAP login")
	if err := c.Login(username, password); err != nil {
		log.Error("IMAP login failed: %v", err)
		return fmt.Errorf("IMAP authentication failed: %w", err)
	}
	log.Debug("IMAP login successful")

	// Logout
	log.Debug("Performing IMAP logout")
	if err := c.Logout(); err != nil {
		log.Error("IMAP logout error: %v", err)
		return fmt.Errorf("IMAP logout error: %w", err)
	}
	log.Debug("IMAP logout completed")

	return nil
}

func (a *AuthModule) authenticateSMTP(username, password, requestID string) error {
	log := a.logger.WithRequestID(requestID)
	log.Debug("Preparing SMTP authentication")

	host, _, err := net.SplitHostPort(a.serverAddress)
	if err != nil {
		log.Error("Invalid server address format: %v", err)
		return fmt.Errorf("invalid server address format: %w", err)
	}

	auth := smtp.PlainAuth("", username, password, host)
	log.Debug("SMTP auth prepared for host: %s", host)

	// Create a TLS connection
	log.Debug("Establishing TLS connection to SMTP server")
	conn, err := tls.Dial("tcp", a.serverAddress, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		log.Error("Failed to connect to SMTP server: %v", err)
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()
	log.Debug("TLS connection established")

	// Create a new SMTP client
	log.Debug("Creating SMTP client")
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Error("Failed to create SMTP client: %v", err)
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer c.Close()
	log.Debug("SMTP client created")

	// Authenticate
	log.Debug("Attempting SMTP authentication")
	if err := c.Auth(auth); err != nil {
		log.Error("SMTP authentication failed: %v", err)
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	log.Debug("SMTP authentication successful")

	return nil
}
