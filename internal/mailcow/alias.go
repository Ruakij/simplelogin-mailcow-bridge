package mailcow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
)

type createAliasRequest struct {
	Address       string `json:"address"`
	Goto          string `json:"goto"`
	Active        string `json:"active"`
	SogoVisible   string `json:"sogo_visible"`
	PublicComment string `json:"public_comment"`
}

// CreateAlias creates a new alias in Mailcow
func (c *MailcowClient) CreateAlias(address, gotoAddress string, sogoVisible bool, publicComment string) error {
	requestID := fmt.Sprintf("MCOW-%d", time.Now().UnixNano())
	log := c.logger.WithRequestID(requestID)

	log.Info("Creating new Mailcow alias: %s -> %s", address, gotoAddress)

	requestBody, err := json.Marshal(createAliasRequest{
		Address:       address,
		Goto:          gotoAddress,
		Active:        "1",
		SogoVisible:   booleanToMailcow(sogoVisible),
		PublicComment: publicComment,
	})
	if err != nil {
		log.Error("Failed to marshal request body: %v", err)
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	log.Debug("Preparing HTTP request to: %s", c.apiURL+"/api/v1/add/alias")
	req, err := http.NewRequest("POST", c.apiURL+"/api/v1/add/alias", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Error("Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(mailcowAPIKeyHeader, c.apiKey)
	log.Debug("Request headers set, executing request")

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		log.Error("Failed to execute request (took %s): %v", logger.FormatDuration(requestDuration), err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received response in %s with status code: %d", logger.FormatDuration(requestDuration), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("status %d (body unreadable: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	_, _ = io.Copy(io.Discard, resp.Body)

	log.Info("Successfully created alias in Mailcow")
	return nil
}

func booleanToMailcow(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
