package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"maintainer-firewall/api-go/internal/service"
	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockEventsStore struct {
	items       []store.WebhookEventRecord
	total       int64
	lastLimit   int
	lastOffset  int
	lastType    string
	lastAction  string
	savedEvents []store.WebhookEvent
	saveErr     error
}

func (m *mockEventsStore) ListEvents(_ context.Context, limit int, offset int, eventType string, action string) ([]store.WebhookEventRecord, int64, error) {
	m.lastLimit = limit
	m.lastOffset = offset
	m.lastType = eventType
	m.lastAction = action
	return m.items, m.total, nil
}

func (m *mockEventsStore) SaveEvent(_ context.Context, evt store.WebhookEvent) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedEvents = append(m.savedEvents, evt)
	return nil
}

type mockGitHubEventTypesProvider struct {
	items  []string
	events []service.GitHubUserEvent
	err    error
	calls  int
}

func (m *mockGitHubEventTypesProvider) ListRecentEventTypes(_ context.Context) ([]string, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

func (m *mockGitHubEventTypesProvider) ListRecentEvents(_ context.Context) ([]service.GitHubUserEvent, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func TestEventsList_WithFiltersAndTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().UTC()
	mockStore := &mockEventsStore{
		items: []store.WebhookEventRecord{
			{
				ID:                 1,
				DeliveryID:         "d1",
				EventType:          "issues",
				Action:             "opened",
				RepositoryFullName: "owner/repo",
				SenderLogin:        "alice",
				ReceivedAt:         now,
			},
		},
		total: 33,
	}

	h := NewEventsHandler(mockStore, nil)
	r := gin.New()
	r.GET("/events", h.List)

	req := httptest.NewRequest(http.MethodGet, "/events?limit=10&offset=20&event_type=issues&action=opened", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if mockStore.lastLimit != 10 || mockStore.lastOffset != 20 {
		t.Fatalf("expected limit=10 offset=20, got limit=%d offset=%d", mockStore.lastLimit, mockStore.lastOffset)
	}
	if mockStore.lastType != "issues" || mockStore.lastAction != "opened" {
		t.Fatalf("expected filters issues/opened, got %s/%s", mockStore.lastType, mockStore.lastAction)
	}

	var resp struct {
		OK     bool   `json:"ok"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
		Total  int64  `json:"total"`
		Items  []any  `json:"items"`
		Event  string `json:"event_type"`
		Action string `json:"action"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.OK || resp.Total != 33 || resp.Limit != 10 || resp.Offset != 20 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
}

func TestEventsList_InvalidLimitOffsetFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStore := &mockEventsStore{items: []store.WebhookEventRecord{}, total: 0}
	h := NewEventsHandler(mockStore, nil)
	r := gin.New()
	r.GET("/events", h.List)

	req := httptest.NewRequest(http.MethodGet, "/events?limit=abc&offset=-10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if mockStore.lastLimit != 20 {
		t.Fatalf("expected default limit=20, got %d", mockStore.lastLimit)
	}
	if mockStore.lastOffset != 0 {
		t.Fatalf("expected sanitized offset=0, got %d", mockStore.lastOffset)
	}
}

func TestEventsList_SourceGitHub_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStore := &mockEventsStore{}
	githubProvider := &mockGitHubEventTypesProvider{items: []string{"CreateEvent", "IssuesEvent"}}
	h := NewEventsHandler(mockStore, githubProvider)
	r := gin.New()
	r.GET("/events", h.List)

	req := httptest.NewRequest(http.MethodGet, "/events?source=github", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if githubProvider.calls != 1 {
		t.Fatalf("expected github provider called once, got %d", githubProvider.calls)
	}
	if mockStore.lastLimit != 0 {
		t.Fatalf("expected db store not called, lastLimit=%d", mockStore.lastLimit)
	}

	var resp struct {
		OK         bool     `json:"ok"`
		Source     string   `json:"source"`
		EventTypes []string `json:"event_types"`
		Total      int      `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || resp.Source != "github" || resp.Total != 2 || len(resp.EventTypes) != 2 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestEventsList_SourceGitHub_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewEventsHandler(&mockEventsStore{}, nil)
	r := gin.New()
	r.GET("/events", h.List)

	req := httptest.NewRequest(http.MethodGet, "/events?source=github", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestEventsList_SourceGitHub_ProviderError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	githubProvider := &mockGitHubEventTypesProvider{err: errors.New("github api status: 401")}
	h := NewEventsHandler(&mockEventsStore{}, githubProvider)
	r := gin.New()
	r.GET("/events", h.List)

	req := httptest.NewRequest(http.MethodGet, "/events?source=github", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestEventsList_SourceGitHub_SyncTrue_SavesEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStore := &mockEventsStore{}
	githubProvider := &mockGitHubEventTypesProvider{events: []service.GitHubUserEvent{{
		DeliveryID:         "gh-1001",
		EventType:          "IssuesEvent",
		Action:             "opened",
		RepositoryFullName: "owner/repo",
		SenderLogin:        "alice",
		PayloadJSON:        []byte(`{"action":"opened"}`),
	}}}
	h := NewEventsHandler(mockStore, githubProvider)
	r := gin.New()
	r.GET("/events", h.List)

	req := httptest.NewRequest(http.MethodGet, "/events?source=github&sync=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if len(mockStore.savedEvents) != 1 {
		t.Fatalf("expected 1 saved event, got %d", len(mockStore.savedEvents))
	}
	if mockStore.savedEvents[0].DeliveryID != "gh-1001" {
		t.Fatalf("unexpected saved event: %+v", mockStore.savedEvents[0])
	}

	var resp struct {
		OK    bool `json:"ok"`
		Sync  bool `json:"sync"`
		Saved int  `json:"saved"`
		Total int  `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || !resp.Sync || resp.Saved != 1 || resp.Total != 1 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestEventsList_SourceGitHub_SyncTrue_StoreNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	githubProvider := &mockGitHubEventTypesProvider{events: []service.GitHubUserEvent{}}
	h := NewEventsHandler(nil, githubProvider)
	r := gin.New()
	r.GET("/events", h.List)

	req := httptest.NewRequest(http.MethodGet, "/events?source=github&sync=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestEventsSyncGitHubEvents_Success(t *testing.T) {
	mockStore := &mockEventsStore{}
	githubProvider := &mockGitHubEventTypesProvider{events: []service.GitHubUserEvent{
		{DeliveryID: "gh-1", EventType: "IssuesEvent", Action: "opened", RepositoryFullName: "a/b", SenderLogin: "alice", PayloadJSON: []byte(`{"id":"1"}`)},
		{DeliveryID: "gh-2", EventType: "PushEvent", Action: "unknown", RepositoryFullName: "a/b", SenderLogin: "alice", PayloadJSON: []byte(`{"id":"2"}`)},
	}}
	h := NewEventsHandler(mockStore, githubProvider)

	saved, total, err := h.SyncGitHubEvents(context.Background())
	if err != nil {
		t.Fatalf("sync github events failed: %v", err)
	}
	if total != 2 || saved != 2 {
		t.Fatalf("expected total=2 saved=2, got total=%d saved=%d", total, saved)
	}
}

func TestEventsSyncGitHubEvents_ProviderError(t *testing.T) {
	h := NewEventsHandler(&mockEventsStore{}, &mockGitHubEventTypesProvider{err: errors.New("boom")})
	_, _, err := h.SyncGitHubEvents(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEventsSyncGitHubEvents_StoreNotConfigured(t *testing.T) {
	h := NewEventsHandler(nil, &mockGitHubEventTypesProvider{events: []service.GitHubUserEvent{}})
	_, _, err := h.SyncGitHubEvents(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEventsSyncGitHubEvents_AlreadyRunning(t *testing.T) {
	h := NewEventsHandler(&mockEventsStore{}, &mockGitHubEventTypesProvider{events: []service.GitHubUserEvent{}})
	h.syncStatus.Running = true
	_, _, err := h.SyncGitHubEvents(context.Background())
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("expected already running error, got %v", err)
	}
}

func TestEventsGitHubSyncStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewEventsHandler(&mockEventsStore{}, &mockGitHubEventTypesProvider{})
	r := gin.New()
	r.GET("/events/sync-status", h.GitHubSyncStatus)

	req := httptest.NewRequest(http.MethodGet, "/events/sync-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, body=%s", w.Body.String())
	}
}
