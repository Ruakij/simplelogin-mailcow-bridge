package auth

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

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
			log.Printf("[%s] Using cached authentication for user %s (valid until %s)",
				requestID, maskedUser, cacheEntry.Expiry.Format(time.RFC3339))
			return nil // Cached authentication is valid
		}

		if found {
			log.Printf("[%s] Cached authentication for user %s has expired, re-authenticating",
				requestID, maskedUser)
		}
	}

	log.Printf("[%s] Starting %s authentication for user %s to server %s",
		requestID, strings.ToUpper(a.method), maskedUser, a.serverAddress)

	var err error
	startTime := time.Now()

	switch strings.ToUpper(a.method) {
	case "IMAP":
		err = a.authenticateIMAP(username, password, requestID)
	case "SMTP":
		err = a.authenticateSMTP(username, password, requestID)
	default:
		err = fmt.Errorf("unsupported authentication method: %s", a.method)
		log.Printf("[%s] %v", requestID, err)
		return err
	}

	duration := time.Since(startTime)
	if err != nil {
		log.Printf("[%s] Authentication failed after %s: %v", requestID, duration, err)
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

		log.Printf("[%s] Cached authentication for user %s (valid until %s)",
			requestID, maskedUser, expiry.Format(time.RFC3339))
	}

	log.Printf("[%s] Authentication successful for user %s (took %s)", requestID, maskedUser, duration)
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
		log.Printf("Cleaned up %d expired cache entries", removed)
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
	log.Printf("[%s] Establishing TLS connection to IMAP server", requestID)

	// Connect to server with a timeout
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", a.serverAddress, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		log.Printf("[%s] Failed to connect to IMAP server: %v", requestID, err)
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	log.Printf("[%s] TLS connection established", requestID)

	// Create a new IMAP client
	log.Printf("[%s] Creating IMAP client", requestID)
	c, err := client.New(conn)
	if err != nil {
		log.Printf("[%s] Failed to create IMAP client: %v", requestID, err)
		return fmt.Errorf("failed to create IMAP client: %w", err)
	}

	// Login
	log.Printf("[%s] Attempting IMAP login", requestID)
	if err := c.Login(username, password); err != nil {
		log.Printf("[%s] IMAP login failed: %v", requestID, err)
		return fmt.Errorf("IMAP authentication failed: %w", err)
	}
	log.Printf("[%s] IMAP login successful", requestID)

	// Logout
	log.Printf("[%s] Performing IMAP logout", requestID)
	if err := c.Logout(); err != nil {
		log.Printf("[%s] IMAP logout error: %v", requestID, err)
		return fmt.Errorf("IMAP logout error: %w", err)
	}
	log.Printf("[%s] IMAP logout completed", requestID)

	return nil
}

func (a *AuthModule) authenticateSMTP(username, password, requestID string) error {
	log.Printf("[%s] Preparing SMTP authentication", requestID)

	host, _, err := net.SplitHostPort(a.serverAddress)
	if err != nil {
		log.Printf("[%s] Invalid server address format: %v", requestID, err)
		return fmt.Errorf("invalid server address format: %w", err)
	}

	auth := smtp.PlainAuth("", username, password, host)
	log.Printf("[%s] SMTP auth prepared for host: %s", requestID, host)

	// Create a TLS connection
	log.Printf("[%s] Establishing TLS connection to SMTP server", requestID)
	conn, err := tls.Dial("tcp", a.serverAddress, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		log.Printf("[%s] Failed to connect to SMTP server: %v", requestID, err)
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()
	log.Printf("[%s] TLS connection established", requestID)

	// Create a new SMTP client
	log.Printf("[%s] Creating SMTP client", requestID)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Printf("[%s] Failed to create SMTP client: %v", requestID, err)
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer c.Close()
	log.Printf("[%s] SMTP client created", requestID)

	// Authenticate
	log.Printf("[%s] Attempting SMTP authentication", requestID)
	if err := c.Auth(auth); err != nil {
		log.Printf("[%s] SMTP authentication failed: %v", requestID, err)
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	log.Printf("[%s] SMTP authentication successful", requestID)

	return nil
}
