package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockEventsStore struct {
	items      []store.WebhookEventRecord
	total      int64
	lastLimit  int
	lastOffset int
	lastType   string
	lastAction string
}

func (m *mockEventsStore) ListEvents(_ context.Context, limit int, offset int, eventType string, action string) ([]store.WebhookEventRecord, int64, error) {
	m.lastLimit = limit
	m.lastOffset = offset
	m.lastType = eventType
	m.lastAction = action
	return m.items, m.total, nil
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

	h := NewEventsHandler(mockStore)
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
	h := NewEventsHandler(mockStore)
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
