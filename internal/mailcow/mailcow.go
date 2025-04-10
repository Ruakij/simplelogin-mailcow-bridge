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

	return &MailcowClient{
		apiURL:     apiURL,
		apiKey:     apiKey,
		httpClient: client,
	}, nil
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[%s] Failed to read success response body: %v", requestID, err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[%s] Response body: %s", requestID, string(body))

	// Try to parse the response to check for any API errors
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err == nil {
		if msg, ok := response["msg"].(string); ok && msg != "" {
			log.Printf("[%s] Mailcow API message: %s", requestID, msg)
			if msg != "alias_added" && msg != "success" {
				log.Printf("[%s] Mailcow API returned an error message", requestID)
				return fmt.Errorf("Mailcow API returned an error: %s", msg)
			}
		}
	} else {
		log.Printf("[%s] Failed to parse JSON response: %v", requestID, err)
	}

	log.Printf("[%s] Successfully created alias in Mailcow", requestID)
	return nil
}
