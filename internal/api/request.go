package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type randomAliasRequest struct {
	Note        string `json:"note"`
	EmailDomain string `json:"email_domain"`
}

// emailDomainOverride holds parsed localpart/domain overrides from the request.
// Empty string means no override for that component.
type emailDomainOverride struct {
	Localpart string
	Domain    string
}

// effectivePattern returns the generation pattern with any email_domain override applied.
func (a *API) effectivePattern(req *randomAliasRequest) (string, error) {
	pattern := a.config.AliasGenerationPattern
	if req.EmailDomain == "" {
		return pattern, nil
	}
	if a.emailDomainRegex != nil && !a.emailDomainRegex.MatchString(req.EmailDomain) {
		return "", fmt.Errorf("email_domain override does not match configured pre-generation regex")
	}
	override, err := parseEmailDomainOverride(req.EmailDomain)
	if err != nil {
		return "", err
	}
	return applyPatternOverride(pattern, override), nil
}

func badRequest(w http.ResponseWriter, err error) {
	http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
}

func decodeAliasRequest(r *http.Request) (*randomAliasRequest, error) {
	var req randomAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			return &req, nil
		}
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}
	return &req, nil
}

// parseEmailDomainOverride parses the raw email_domain field into override components.
// Returns nil if rawValue is empty.
//
// Parsing rules (applied to raw value before any alias generation):
//   - No "@"             → domain-only  (e.g. "example.org")
//   - "@domain"          → domain-only  (e.g. "@example.org")
//   - "localpart@"       → localpart-only
//   - "localpart@domain" → both
func parseEmailDomainOverride(rawValue string) (*emailDomainOverride, error) {
	if rawValue == "" {
		return nil, nil
	}

	atIdx := strings.Index(rawValue, "@")
	if atIdx == -1 {
		return &emailDomainOverride{Domain: rawValue}, nil
	}

	localpart := rawValue[:atIdx]
	domain := rawValue[atIdx+1:]

	if localpart == "" && domain == "" {
		return nil, fmt.Errorf("email_domain override is invalid")
	}

	return &emailDomainOverride{Localpart: localpart, Domain: domain}, nil
}

// applyPatternOverride returns a modified generation pattern with the override applied.
// Splits on the last "@" in the pattern to separate the localpart template from the
// domain template, then substitutes whichever components are overridden.
func applyPatternOverride(pattern string, override *emailDomainOverride) string {
	if override == nil {
		return pattern
	}

	atIdx := strings.LastIndex(pattern, "@")

	var localpartPart, domainPart string
	if atIdx == -1 {
		localpartPart = pattern
	} else {
		localpartPart = pattern[:atIdx]
		domainPart = pattern[atIdx+1:]
	}

	if override.Localpart != "" {
		localpartPart = override.Localpart
	}
	if override.Domain != "" {
		domainPart = override.Domain
	}

	if domainPart == "" {
		return localpartPart
	}
	return localpartPart + "@" + domainPart
}
