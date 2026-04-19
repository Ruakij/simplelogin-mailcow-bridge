package mailcow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
)

type editMailboxRequest struct {
	Items []string        `json:"items"`
	Attr  editMailboxAttr `json:"attr"`
}

type editMailboxAttr struct {
	SenderACL []string `json:"sender_acl"`
}

// AllowMailboxToSendAsAlias updates the sender ACL so a mailbox can send as a generated alias.
func (c *MailcowClient) AllowMailboxToSendAsAlias(mailbox, aliasAddress string) error {
	requestID := fmt.Sprintf("MCOW-%d", time.Now().UnixNano())
	log := c.logger.WithRequestID(requestID)

	log.Info("Updating sender ACL for mailbox %s to allow alias %s", mailbox, aliasAddress)

	requestBody, err := json.Marshal(editMailboxRequest{
		Items: []string{mailbox},
		Attr:  editMailboxAttr{SenderACL: []string{"default", aliasAddress}},
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
		body, readErr := readResponseBody(resp.Body)
		if readErr != nil {
			log.Error("Failed to read sender ACL update error response body: %v", readErr)
			return fmt.Errorf("failed to update sender ACL, status code: %d, and could not read response body", resp.StatusCode)
		}

		log.Error("Sender ACL update error response body: %s", string(body))
		return fmt.Errorf("failed to update sender ACL, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	if _, err := readResponseBody(resp.Body); err != nil {
		log.Error("Failed to read sender ACL update success response body: %v", err)
		return fmt.Errorf("failed to read sender ACL update response body: %w", err)
	}

	log.Info("Successfully updated sender ACL for mailbox %s", mailbox)
	return nil
}
