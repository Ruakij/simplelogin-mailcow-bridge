package auth

import (
	"crypto/tls"
	"fmt"
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
	switch strings.ToUpper(a.method) {
	case "IMAP":
		return a.authenticateIMAP(username, password)
	case "SMTP":
		return a.authenticateSMTP(username, password)
	default:
		return fmt.Errorf("unsupported authentication method: %s", a.method)
	}
}

func (a *AuthModule) authenticateIMAP(username, password string) error {
	// Connect to server with a timeout
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", a.serverAddress, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	// Create a new IMAP client
	c, err := client.New(conn)
	if err != nil {
		return fmt.Errorf("failed to create IMAP client: %w", err)
	}

	// Login
	if err := c.Login(username, password); err != nil {
		return fmt.Errorf("IMAP authentication failed: %w", err)
	}

	// Logout
	if err := c.Logout(); err != nil {
		// Throw error instead of just logging it
		return fmt.Errorf("IMAP logout error: %w", err)
	}

	return nil
}

func (a *AuthModule) authenticateSMTP(username, password string) error {
	host, _, err := net.SplitHostPort(a.serverAddress)
	if err != nil {
		return fmt.Errorf("invalid server address format: %w", err)
	}

	auth := smtp.PlainAuth("", username, password, host)

	// Create a TLS connection
	conn, err := tls.Dial("tcp", a.serverAddress, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create a new SMTP client
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer c.Close()

	// Authenticate
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return nil
}
