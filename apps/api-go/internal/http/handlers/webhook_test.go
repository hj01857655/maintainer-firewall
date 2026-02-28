package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockWebhookStore struct {
	saved       []store.WebhookEvent
	savedAlerts []store.AlertRecord
}

func (m *mockWebhookStore) SaveEvent(_ context.Context, evt store.WebhookEvent) error {
	m.saved = append(m.saved, evt)
	return nil
}

func (m *mockWebhookStore) SaveAlert(_ context.Context, alert store.AlertRecord) error {
	m.savedAlerts = append(m.savedAlerts, alert)
	return nil
}

func TestWebhookGitHub_SignatureValid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	payload := map[string]any{
		"action": "opened",
		"repository": map[string]any{"full_name": "owner/repo"},
		"sender": map[string]any{"login": "alice"},
		"issue": map[string]any{"title": "urgent duplicate"},
	}
	body, _ := json.Marshal(payload)
	signature := signBody(secret, body)

	mockStore := &mockWebhookStore{}
	h := NewWebhookHandler(secret, mockStore)

	r := gin.New()
	r.POST("/webhook/github", h.GitHub)

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-GitHub-Delivery", "delivery-1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if len(mockStore.saved) != 1 {
		t.Fatalf("expected 1 saved event, got %d", len(mockStore.saved))
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, body=%s", w.Body.String())
	}
	if _, exists := resp["suggested_actions"]; !exists {
		t.Fatalf("expected suggested_actions in response, body=%s", w.Body.String())
	}
	if len(mockStore.savedAlerts) == 0 {
		t.Fatalf("expected at least 1 alert record to be saved")
	}
	if mockStore.savedAlerts[0].EventType != "issues" || mockStore.savedAlerts[0].Action != "opened" {
		t.Fatalf("unexpected alert event/action: %+v", mockStore.savedAlerts[0])
	}
}

func TestWebhookGitHub_SignatureInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	body := []byte(`{"action":"opened"}`)

	mockStore := &mockWebhookStore{}
	h := NewWebhookHandler(secret, mockStore)

	r := gin.New()
	r.POST("/webhook/github", h.GitHub)

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	req.Header.Set("X-GitHub-Event", "issues")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	if len(mockStore.saved) != 0 {
		t.Fatalf("expected 0 saved events, got %d", len(mockStore.saved))
	}
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
