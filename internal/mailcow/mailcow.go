package mailcow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	return &MailcowClient{
		apiURL:     apiURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}, nil
}

// CreateAlias creates a new alias in Mailcow
func (c *MailcowClient) CreateAlias(address, gotoAddress string) error {
	// Prepare request body
	requestBody, err := json.Marshal(map[string]string{
		"address": address,
		"goto":    gotoAddress,
		"active":  "1", // Active by default
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.apiURL+"/api/v1/add/alias", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		// Read the response body for error details
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to create alias, status code: %d, and could not read response body", resp.StatusCode)
		}
		return fmt.Errorf("failed to create alias, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Validate the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to parse the response to check for any API errors
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err == nil {
		if msg, ok := response["msg"].(string); ok && msg != "" {
			if msg != "alias_added" && msg != "success" {
				return fmt.Errorf("Mailcow API returned an error: %s", msg)
			}
		}
	}

	return nil
}
