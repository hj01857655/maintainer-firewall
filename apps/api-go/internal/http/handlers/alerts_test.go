package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockAlertsStore struct {
	items              []store.AlertRecord
	total              int64
	lastLimit          int
	lastOffset         int
	lastEventType      string
	lastAction         string
	lastSuggestionType string
	filterOptions      store.AlertFilterOptions
	filterErr          error
}

func (m *mockAlertsStore) ListAlerts(_ context.Context, limit int, offset int, eventType string, action string, suggestionType string) ([]store.AlertRecord, int64, error) {
	m.lastLimit = limit
	m.lastOffset = offset
	m.lastEventType = eventType
	m.lastAction = action
	m.lastSuggestionType = suggestionType
	return m.items, m.total, nil
}

func (m *mockAlertsStore) ListAlertFilterOptions(_ context.Context) (store.AlertFilterOptions, error) {
	if m.filterErr != nil {
		return store.AlertFilterOptions{}, m.filterErr
	}
	return m.filterOptions, nil
}

func TestAlertsList_WithFiltersAndTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().UTC()
	mockStore := &mockAlertsStore{
		items: []store.AlertRecord{
			{
				DeliveryID:         "d-1",
				EventType:          "issues",
				Action:             "opened",
				RepositoryFullName: "owner/repo",
				SenderLogin:        "alice",
				RuleMatched:        "urgent",
				SuggestionType:     "label",
				SuggestionValue:    "priority-high",
				Reason:             "contains urgent keyword",
				CreatedAt:          now,
			},
		},
		total: 5,
	}

	h := NewAlertsHandler(mockStore)
	r := gin.New()
	r.GET("/alerts", h.List)

	req := httptest.NewRequest(http.MethodGet, "/alerts?limit=10&offset=20&event_type=issues&action=opened&suggestion_type=label", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if mockStore.lastLimit != 10 || mockStore.lastOffset != 20 {
		t.Fatalf("expected limit=10 offset=20, got limit=%d offset=%d", mockStore.lastLimit, mockStore.lastOffset)
	}
	if mockStore.lastEventType != "issues" || mockStore.lastAction != "opened" || mockStore.lastSuggestionType != "label" {
		t.Fatalf("expected filters issues/opened/label, got %s/%s/%s", mockStore.lastEventType, mockStore.lastAction, mockStore.lastSuggestionType)
	}

	var resp struct {
		OK             bool   `json:"ok"`
		Limit          int    `json:"limit"`
		Offset         int    `json:"offset"`
		Total          int64  `json:"total"`
		Items          []any  `json:"items"`
		EventType      string `json:"event_type"`
		Action         string `json:"action"`
		SuggestionType string `json:"suggestion_type"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.OK || resp.Total != 5 || resp.Limit != 10 || resp.Offset != 20 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
}

func TestAlertsList_InvalidLimitOffsetFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStore := &mockAlertsStore{items: []store.AlertRecord{}, total: 0}
	h := NewAlertsHandler(mockStore)
	r := gin.New()
	r.GET("/alerts", h.List)

	req := httptest.NewRequest(http.MethodGet, "/alerts?limit=abc&offset=-10", nil)
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

func TestAlertsFilterOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockAlertsStore{filterOptions: store.AlertFilterOptions{
		EventTypes:      []string{"issues"},
		Actions:         []string{"opened"},
		SuggestionTypes: []string{"label"},
		Repositories:    []string{"owner/repo"},
		Senders:         []string{"alice"},
	}}
	h := NewAlertsHandler(mockStore)
	r := gin.New()
	r.GET("/alerts/filter-options", h.FilterOptions)

	req := httptest.NewRequest(http.MethodGet, "/alerts/filter-options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		OK      bool                     `json:"ok"`
		Options store.AlertFilterOptions `json:"options"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || len(resp.Options.EventTypes) != 1 || len(resp.Options.SuggestionTypes) != 1 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestAlertsFilterOptions_StoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAlertsHandler(&mockAlertsStore{filterErr: errors.New("boom")})
	r := gin.New()
	r.GET("/alerts/filter-options", h.FilterOptions)

	req := httptest.NewRequest(http.MethodGet, "/alerts/filter-options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", w.Code, w.Body.String())
	}
}
