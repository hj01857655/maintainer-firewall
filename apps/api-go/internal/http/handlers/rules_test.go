package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockRulesStore struct {
	items      []store.RuleRecord
	total      int64
	createdID  int64
	created    []store.RuleRecord
	lastLimit  int
	lastOffset int
	lastEvent  string
	lastKey    string
	lastActive bool
}

func (m *mockRulesStore) ListRules(_ context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]store.RuleRecord, int64, error) {
	m.lastLimit = limit
	m.lastOffset = offset
	m.lastEvent = eventType
	m.lastKey = keyword
	m.lastActive = activeOnly
	return m.items, m.total, nil
}

func (m *mockRulesStore) CreateRule(_ context.Context, rule store.RuleRecord) (int64, error) {
	m.created = append(m.created, rule)
	if m.createdID == 0 {
		m.createdID = 1
	}
	return m.createdID, nil
}

func TestRulesList_WithFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	mockStore := &mockRulesStore{
		items: []store.RuleRecord{{ID: 1, EventType: "issues", Keyword: "urgent", SuggestionType: "label", SuggestionValue: "P0", Reason: "r", IsActive: true, CreatedAt: now}},
		total: 1,
	}

	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.GET("/rules", h.List)

	req := httptest.NewRequest(http.MethodGet, "/rules?limit=10&offset=20&event_type=issues&keyword=urg&active_only=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if mockStore.lastLimit != 10 || mockStore.lastOffset != 20 || mockStore.lastEvent != "issues" || mockStore.lastKey != "urg" || !mockStore.lastActive {
		t.Fatalf("unexpected list args: %+v", mockStore)
	}

	var resp struct {
		OK    bool  `json:"ok"`
		Total int64 `json:"total"`
		Items []any `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestRulesCreate_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{createdID: 9}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.POST("/rules", h.Create)

	body := `{"event_type":"issues","keyword":"urgent","suggestion_type":"label","suggestion_value":"P0","reason":"urgent rule","is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if len(mockStore.created) != 1 {
		t.Fatalf("expected 1 created rule, got %d", len(mockStore.created))
	}
	if mockStore.created[0].Keyword != "urgent" || mockStore.created[0].SuggestionValue != "P0" {
		t.Fatalf("unexpected created rule: %+v", mockStore.created[0])
	}
}

func TestRulesCreate_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.POST("/rules", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(`{"event_type":"issues"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}
