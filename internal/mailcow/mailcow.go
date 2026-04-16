package mailcow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		body, _ := ioutil.ReadAll(resp.Body)
		log.Error("API check failed with status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("API check failed with status %d", resp.StatusCode)
	}

	log.Info("Mailcow API connectivity check successful")
	return nil
}

// CreateAlias creates a new alias in Mailcow
func (c *MailcowClient) CreateAlias(address, gotoAddress string, sogoVisible bool, publicComment string) error {
	requestID := fmt.Sprintf("MCOW-%d", time.Now().UnixNano())
	log := c.logger.WithRequestID(requestID)

	log.Info("Creating new Mailcow alias: %s -> %s", address, gotoAddress)

	// Prepare request body
	requestBody, err := json.Marshal(map[string]string{
		"address": address,
		"goto":    gotoAddress,
		"active":  "1", // Active by default
		"sogo_visible": booleanToMailcow(sogoVisible),
		"public_comment": publicComment,
	})
	if err != nil {
		log.Error("Failed to marshal request body: %v", requestID, err)
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	log.Debug("Preparing HTTP request to: %s", c.apiURL+"/api/v1/add/alias")
	req, err := http.NewRequest("POST", c.apiURL+"/api/v1/add/alias", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Error("Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(mailcowAPIKeyHeader, c.apiKey)
	log.Debug("Request headers set, executing request")

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		log.Error("Failed to execute request (took %s): %v", logger.FormatDuration(requestDuration), err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received response in %s with status code: %d", logger.FormatDuration(requestDuration), resp.StatusCode)

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		// Read the response body for error details
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("Failed to read error response body: %v", err)
			return fmt.Errorf("failed to create alias, status code: %d, and could not read response body", resp.StatusCode)
		}
		log.Error("Error response body: %s", string(body))
		return fmt.Errorf("failed to create alias, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Validate the response
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read success response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Info("Successfully created alias in Mailcow")
	return nil
}

// AllowMailboxToSendAsAlias updates the sender ACL so a mailbox can send as a generated alias.
func (c *MailcowClient) AllowMailboxToSendAsAlias(mailbox, aliasAddress string) error {
	requestID := fmt.Sprintf("MCOW-%d", time.Now().UnixNano())
	log := c.logger.WithRequestID(requestID)

	log.Info("Updating sender ACL for mailbox %s to allow alias %s", mailbox, aliasAddress)

	requestBody, err := json.Marshal(map[string]interface{}{
		"items": []string{mailbox},
		"attr": map[string]interface{}{
			"sender_acl": []string{"default", aliasAddress},
		},
	})
	if err != nil {
		log.Error("Failed to marshal sender ACL update request body: %v", err)
		return fmt.Errorf("failed to marshal sender ACL update request body: %w", err)
	}

	log.Debug("Preparing HTTP request to: %s", c.apiURL+"/api/v1/edit/mailbox")
	req, err := http.NewRequest("POST", c.apiURL+"/api/v1/edit/mailbox", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Error("Failed to create sender ACL update request: %v", err)
		return fmt.Errorf("failed to create sender ACL update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(mailcowAPIKeyHeader, c.apiKey)

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)
	if err != nil {
		log.Error("Failed to execute sender ACL update request (took %s): %v", logger.FormatDuration(requestDuration), err)
		return fmt.Errorf("failed to execute sender ACL update request: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received sender ACL update response in %s with status code: %d", logger.FormatDuration(requestDuration), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			log.Error("Failed to read sender ACL update error response body: %v", readErr)
			return fmt.Errorf("failed to update sender ACL, status code: %d, and could not read response body", resp.StatusCode)
		}

		log.Error("Sender ACL update error response body: %s", string(body))
		return fmt.Errorf("failed to update sender ACL, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		log.Error("Failed to read sender ACL update success response body: %v", err)
		return fmt.Errorf("failed to read sender ACL update response body: %w", err)
	}

	log.Info("Successfully updated sender ACL for mailbox %s", mailbox)
	return nil
}

func booleanToMailcow(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
