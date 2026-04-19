package mailcow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
)

type mailboxEntry struct {
	SenderACL  stringList        `json:"sender_acl"`
	Attributes mailboxAttributes `json:"attributes"`
}

type mailboxAttributes struct {
	SenderACL stringList `json:"sender_acl"`
}

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

	existingSenderACL, err := c.getMailboxSenderACL(mailbox)
	if err != nil {
		log.Warn("Failed to fetch existing sender ACL for mailbox %s: %v", mailbox, err)
	}

	senderACL := buildSenderACL(existingSenderACL, aliasAddress)
	log.Debug("Computed merged sender ACL with %d entries for mailbox %s", len(senderACL), mailbox)

	requestBody, err := json.Marshal(editMailboxRequest{
		Items: []string{mailbox},
		Attr:  editMailboxAttr{SenderACL: senderACL},
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

func (c *MailcowClient) getMailboxSenderACL(mailbox string) ([]string, error) {
	mailboxEndpoint := fmt.Sprintf("%s/api/v1/get/mailbox/%s", c.apiURL, url.PathEscape(mailbox))

	req, err := http.NewRequest("GET", mailboxEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create mailbox query request: %w", err)
	}
	req.Header.Set(mailcowAPIKeyHeader, c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query mailbox sender ACL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := readResponseBody(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to query mailbox sender ACL, status code: %d, and could not read response body", resp.StatusCode)
		}

		return nil, fmt.Errorf("failed to query mailbox sender ACL, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read mailbox sender ACL response body: %w", err)
	}

	senderACL, err := parseSenderACLFromMailboxResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mailbox sender ACL response: %w", err)
	}

	return senderACL, nil
}

func parseSenderACLFromMailboxResponse(body []byte) ([]string, error) {
	var entries []mailboxEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mailbox response: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	entry := entries[0]
	if len(entry.SenderACL) > 0 {
		return entry.SenderACL, nil
	}
	if len(entry.Attributes.SenderACL) > 0 {
		return entry.Attributes.SenderACL, nil
	}

	return nil, nil
}

func buildSenderACL(existingSenderACL []string, newAlias string) []string {
	merged := make([]string, 0, len(existingSenderACL)+1)
	merged = append(merged, existingSenderACL...)
	merged = append(merged, newAlias)
	return merged
}
