package mailcow

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
)

const mailcowAPIKeyHeader = "X-API-Key"

// MailcowClient is a client for the Mailcow Admin API
type MailcowClient struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
	logger     *logger.Logger
}

// NewMailcowClient creates a new MailcowClient
func NewMailcowClient(apiURL, apiKey string) (*MailcowClient, error) {
	if apiURL == "" || apiKey == "" {
		return nil, fmt.Errorf("apiURL and apiKey must be set")
	}

	// Create HTTP client with reasonable timeouts
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	mc := &MailcowClient{
		apiURL:     apiURL,
		apiKey:     apiKey,
		httpClient: client,
		logger:     logger.WithComponent("Mailcow"),
	}

	// Check API connectivity at startup
	if err := mc.CheckAPIConnectivity(); err != nil {
		return nil, fmt.Errorf("Mailcow API connectivity check failed: %w", err)
	}

	return mc, nil
}

// CheckAPIConnectivity verifies the Mailcow API is accessible
func (c *MailcowClient) CheckAPIConnectivity() error {
	requestID := fmt.Sprintf("MCOW-INIT-%d", time.Now().UnixNano())
	log := c.logger.WithRequestID(requestID)

	log.Info("Checking Mailcow API connectivity at %s", c.apiURL)

	// Create request to the mailq endpoint
	req, err := http.NewRequest("GET", c.apiURL+"/api/v1/get/mailq/all", nil)
	if err != nil {
		log.Error("Failed to create API check request: %v", err)
		return fmt.Errorf("failed to create API check request: %w", err)
	}

	// Set headers
	req.Header.Set(mailcowAPIKeyHeader, c.apiKey)

	// Execute request with timeout
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		log.Error("API connectivity check failed (took %s): %v", logger.FormatDuration(requestDuration), err)
		return fmt.Errorf("API connectivity check failed: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received API check response in %s with status code: %d", logger.FormatDuration(requestDuration), resp.StatusCode)

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		// Read the response body for error details
		body, _ := io.ReadAll(resp.Body)
		log.Error("API check failed with status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("API check failed with status %d", resp.StatusCode)
	}

	log.Info("Mailcow API connectivity check successful")
	return nil
}
