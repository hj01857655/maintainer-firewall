package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockWebhookStore struct {
	saved            []store.WebhookEvent
	savedAlerts      []store.AlertRecord
	savedActionFails []store.ActionExecutionFailure
	rules            []store.RuleRecord
}

type mockWebhookExecutor struct {
	labels          []string
	comments        []string
	labelFailTimes  int
	commentFailTimes int
	labelCalls      int
	commentCalls    int
}

func (m *mockWebhookExecutor) AddLabel(_ context.Context, _ string, _ int, label string) error {
	m.labelCalls++
	if m.labelFailTimes > 0 {
		m.labelFailTimes--
		return errors.New("label fail")
	}
	m.labels = append(m.labels, label)
	return nil
}

func (m *mockWebhookExecutor) AddComment(_ context.Context, _ string, _ int, body string) error {
	m.commentCalls++
	if m.commentFailTimes > 0 {
		m.commentFailTimes--
		return errors.New("comment fail")
	}
	m.comments = append(m.comments, body)
	return nil
}

func (m *mockWebhookStore) SaveEvent(_ context.Context, evt store.WebhookEvent) error {
	m.saved = append(m.saved, evt)
	return nil
}

func (m *mockWebhookStore) SaveAlert(_ context.Context, alert store.AlertRecord) error {
	m.savedAlerts = append(m.savedAlerts, alert)
	return nil
}

func (m *mockWebhookStore) SaveActionExecutionFailure(_ context.Context, item store.ActionExecutionFailure) error {
	m.savedActionFails = append(m.savedActionFails, item)
	return nil
}

func (m *mockWebhookStore) ListRules(_ context.Context, _ int, _ int, _ string, _ string, _ bool) ([]store.RuleRecord, int64, error) {
	return m.rules, int64(len(m.rules)), nil
}

func TestWebhookGitHub_SignatureValid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	payload := map[string]any{
		"action": "opened",
		"repository": map[string]any{"full_name": "owner/repo"},
		"sender": map[string]any{"login": "alice"},
		"issue": map[string]any{"title": "urgent duplicate", "number": 12},
	}
	body, _ := json.Marshal(payload)
	signature := signBody(secret, body)

	mockStore := &mockWebhookStore{
		rules: []store.RuleRecord{
			{EventType: "issues", Keyword: "urgent", SuggestionType: "label", SuggestionValue: "P0", Reason: "urgent rule"},
		},
	}
	h := NewWebhookHandler(secret, mockStore)
	exec := &mockWebhookExecutor{}
	h.ActionExecutor = exec

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
	if mockStore.savedAlerts[0].SuggestionValue != "P0" {
		t.Fatalf("expected rule suggestion value P0, got %s", mockStore.savedAlerts[0].SuggestionValue)
	}
	if len(exec.labels) != 1 || exec.labels[0] != "P0" {
		t.Fatalf("expected executor label P0, got %+v", exec.labels)
	}
}

func TestWebhookGitHub_ExecutorFailureDoesNotBlockWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	payload := map[string]any{
		"action": "opened",
		"repository": map[string]any{"full_name": "owner/repo"},
		"sender": map[string]any{"login": "alice"},
		"issue": map[string]any{"title": "urgent duplicate", "number": 12},
	}
	body, _ := json.Marshal(payload)
	signature := signBody(secret, body)

	mockStore := &mockWebhookStore{
		rules: []store.RuleRecord{{EventType: "issues", Keyword: "urgent", SuggestionType: "label", SuggestionValue: "P0", Reason: "urgent rule"}},
	}
	h := NewWebhookHandler(secret, mockStore)
	exec := &mockWebhookExecutor{labelFailTimes: 5}
	h.ActionExecutor = exec

	r := gin.New()
	r.POST("/webhook/github", h.GitHub)

	req := httptest.NewRequest(http.MethodPost, "/webhook/github", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-GitHub-Delivery", "delivery-fail")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 even when action execution failed, got %d, body=%s", w.Code, w.Body.String())
	}
	if len(mockStore.saved) != 1 || len(mockStore.savedAlerts) == 0 {
		t.Fatalf("event/alert should still persist, events=%d alerts=%d", len(mockStore.saved), len(mockStore.savedAlerts))
	}
	if exec.labelCalls < 3 {
		t.Fatalf("expected retry attempts >=3, got %d", exec.labelCalls)
	}
	if len(mockStore.savedActionFails) == 0 {
		t.Fatalf("expected action execution failure to be persisted")
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
