package auth

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/emersion/go-imap/client"
)

// AuthModule is a module for authenticating users against Mailcow
type AuthModule struct {
	method        string
	serverAddress string
}

// NewAuthModule creates a new AuthModule
func NewAuthModule(method, serverAddress string) (*AuthModule, error) {
	if method == "" || serverAddress == "" {
		return nil, fmt.Errorf("method and serverAddress must be set")
	}

	// Default to IMAP if method is not specified
	if method == "" {
		method = "IMAP"
	}

	return &AuthModule{
		method:        method,
		serverAddress: serverAddress,
	}, nil
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

	log.Printf("[%s] Authentication successful for user %s (took %s)", requestID, maskedUser, duration)
	return nil
}

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
