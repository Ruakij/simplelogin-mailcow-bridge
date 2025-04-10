package mailcow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// MailcowClient is a client for the Mailcow Admin API
type MailcowClient struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
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
	log.Printf("[%s] Checking Mailcow API connectivity at %s", requestID, c.apiURL)

	// Create request to the mailq endpoint
	req, err := http.NewRequest("GET", c.apiURL+"/api/v1/get/mailq/all", nil)
	if err != nil {
		log.Printf("[%s] Failed to create API check request: %v", requestID, err)
		return fmt.Errorf("failed to create API check request: %w", err)
	}

	// Set headers
	req.Header.Set("X-API-Key", c.apiKey)

	// Execute request with timeout
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		log.Printf("[%s] API connectivity check failed (took %s): %v", requestID, requestDuration, err)
		return fmt.Errorf("API connectivity check failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[%s] Received API check response in %s with status code: %d", requestID, requestDuration, resp.StatusCode)

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		// Read the response body for error details
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("[%s] API check failed with status %d: %s", requestID, resp.StatusCode, string(body))
		return fmt.Errorf("API check failed with status %d", resp.StatusCode)
	}

	log.Printf("[%s] Mailcow API connectivity check successful", requestID)
	return nil
}

// CreateAlias creates a new alias in Mailcow
func (c *MailcowClient) CreateAlias(address, gotoAddress string) error {
	requestID := fmt.Sprintf("MCOW-%d", time.Now().UnixNano())
	log.Printf("[%s] Creating new Mailcow alias: %s -> %s", requestID, address, gotoAddress)

	// Prepare request body
	requestBody, err := json.Marshal(map[string]string{
		"address": address,
		"goto":    gotoAddress,
		"active":  "1", // Active by default
	})
	if err != nil {
		log.Printf("[%s] Failed to marshal request body: %v", requestID, err)
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	log.Printf("[%s] Preparing HTTP request to: %s", requestID, c.apiURL+"/api/v1/add/alias")
	req, err := http.NewRequest("POST", c.apiURL+"/api/v1/add/alias", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("[%s] Failed to create request: %v", requestID, err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	log.Printf("[%s] Request headers set, executing request", requestID)

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		log.Printf("[%s] Failed to execute request (took %s): %v", requestID, requestDuration, err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[%s] Received response in %s with status code: %d", requestID, requestDuration, resp.StatusCode)

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		// Read the response body for error details
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[%s] Failed to read error response body: %v", requestID, err)
			return fmt.Errorf("failed to create alias, status code: %d, and could not read response body", resp.StatusCode)
		}
		log.Printf("[%s] Error response body: %s", requestID, string(body))
		return fmt.Errorf("failed to create alias, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Validate the response
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[%s] Failed to read success response body: %v", requestID, err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[%s] Successfully created alias in Mailcow", requestID)
	return nil
}
