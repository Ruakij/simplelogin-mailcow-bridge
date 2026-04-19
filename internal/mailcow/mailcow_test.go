package mailcow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"git.ruekov.eu/ruakij/simplelogin-mailcow-bridge/internal/logger"
)

const (
	testMailbox   = "user@example.com"
	testLegacyACL = "legacy@example.com"
	testOldAlias  = "old@example.com"
	testNewAlias  = "new@example.com"
)

func TestBuildSenderACLPreservesExistingEntries(t *testing.T) {
	existing := []string{"default", testLegacyACL, "*", "OLD@example.com"}

	acl := buildSenderACL(existing, testNewAlias)
	expected := []string{"default", testLegacyACL, "*", "OLD@example.com", testNewAlias}

	if !reflect.DeepEqual(acl, expected) {
		t.Fatalf("expected merged ACL %v, got %v", expected, acl)
	}
}

func TestBuildSenderACLAppendsWithoutDeduplication(t *testing.T) {
	existing := []string{"default", testNewAlias}

	acl := buildSenderACL(existing, testNewAlias)
	expected := []string{"default", testNewAlias, testNewAlias}

	if !reflect.DeepEqual(acl, expected) {
		t.Fatalf("expected appended ACL %v, got %v", expected, acl)
	}
}

func TestParseSenderACLFromMailboxResponse(t *testing.T) {
	t.Run("top-level sender_acl", func(t *testing.T) {
		response := []byte(fmt.Sprintf(`[{"sender_acl":["default","%s","*"]}]`, testOldAlias))

		acl, err := parseSenderACLFromMailboxResponse(response)
		if err != nil {
			t.Fatalf("expected sender ACL parsing to succeed, got error: %v", err)
		}

		expected := []string{"default", testOldAlias, "*"}
		if !reflect.DeepEqual(acl, expected) {
			t.Fatalf("expected sender ACL %v, got %v", expected, acl)
		}
	})

	t.Run("sender_acl in attributes", func(t *testing.T) {
		response := []byte(fmt.Sprintf(`[{"attributes":{"sender_acl":"default, %s"}}]`, testOldAlias))

		acl, err := parseSenderACLFromMailboxResponse(response)
		if err != nil {
			t.Fatalf("expected sender ACL parsing to succeed, got error: %v", err)
		}

		expected := []string{"default", testOldAlias}
		if !reflect.DeepEqual(acl, expected) {
			t.Fatalf("expected sender ACL %v, got %v", expected, acl)
		}
	})
}

func TestAllowMailboxToSendAsAliasMergesExistingACLAndAliases(t *testing.T) {
	var capturedRequest editMailboxRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/get/mailbox/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`[{"sender_acl":["default","%s","*"]}]`, testLegacyACL)))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/edit/mailbox":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read edit mailbox request body: %v", err)
			}

			if err := json.Unmarshal(body, &capturedRequest); err != nil {
				t.Fatalf("failed to parse edit mailbox request body: %v", err)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"type":"success"}]`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := &MailcowClient{
		apiURL:     server.URL,
		apiKey:     "test-api-key",
		httpClient: server.Client(),
		logger:     logger.WithComponent("MailcowTest"),
	}

	if err := client.AllowMailboxToSendAsAlias(testMailbox, testNewAlias); err != nil {
		t.Fatalf("expected sender ACL update to succeed, got error: %v", err)
	}

	expectedItems := []string{testMailbox}
	if !reflect.DeepEqual(capturedRequest.Items, expectedItems) {
		t.Fatalf("expected items %v, got %v", expectedItems, capturedRequest.Items)
	}

	expectedACL := []string{"default", testLegacyACL, "*", testNewAlias}
	if !reflect.DeepEqual(capturedRequest.Attr.SenderACL, expectedACL) {
		t.Fatalf("expected sender_acl %v, got %v", expectedACL, capturedRequest.Attr.SenderACL)
	}
}
