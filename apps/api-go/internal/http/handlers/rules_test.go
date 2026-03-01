package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockRulesStore struct {
	items            []store.RuleRecord
	total            int64
	createdID        int64
	created          []store.RuleRecord
	lastLimit        int
	lastOffset       int
	lastEvent        string
	lastKey          string
	lastActive       bool
	updatedID        int64
	updatedIsActive  bool
	updateShouldFail bool
	filterOptions    store.RuleFilterOptions
	filterErr        error
}

func (m *mockRulesStore) ListRules(_ context.Context, limit int, offset int, eventType string, keyword string, activeOnly bool) ([]store.RuleRecord, int64, error) {
	m.lastLimit = limit
	m.lastOffset = offset
	m.lastEvent = eventType
	m.lastKey = keyword
	m.lastActive = activeOnly
	return m.items, m.total, nil
}

func (m *mockRulesStore) ListRuleFilterOptions(_ context.Context) (store.RuleFilterOptions, error) {
	if m.filterErr != nil {
		return store.RuleFilterOptions{}, m.filterErr
	}
	return m.filterOptions, nil
}

func (m *mockRulesStore) CreateRule(_ context.Context, rule store.RuleRecord) (int64, error) {
	m.created = append(m.created, rule)
	if m.createdID == 0 {
		m.createdID = 1
	}
	return m.createdID, nil
}

func (m *mockRulesStore) UpdateRuleActive(_ context.Context, id int64, isActive bool) error {
	if m.updateShouldFail {
		return fmt.Errorf("db failure")
	}
	if id == 404 {
		return fmt.Errorf("rule not found")
	}
	m.updatedID = id
	m.updatedIsActive = isActive
	return nil
}

func (m *mockRulesStore) SaveAuditLog(_ context.Context, _ store.AuditLogRecord) error {
	return nil
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

func TestRulesUpdateActive_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.PATCH("/rules/:id/active", h.UpdateActive)

	req := httptest.NewRequest(http.MethodPatch, "/rules/7/active", strings.NewReader(`{"is_active":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if mockStore.updatedID != 7 || mockStore.updatedIsActive {
		t.Fatalf("unexpected update args: id=%d is_active=%v", mockStore.updatedID, mockStore.updatedIsActive)
	}
}

func TestRulesUpdateActive_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.PATCH("/rules/:id/active", h.UpdateActive)

	req := httptest.NewRequest(http.MethodPatch, "/rules/404/active", strings.NewReader(`{"is_active":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRulesUpdateActive_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.PATCH("/rules/:id/active", h.UpdateActive)

	req := httptest.NewRequest(http.MethodPatch, "/rules/invalid/active", strings.NewReader(`{"is_active":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRulesFilterOptions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{filterOptions: store.RuleFilterOptions{
		EventTypes:      []string{"issues", "pull_request"},
		SuggestionTypes: []string{"label", "comment"},
		ActiveStates:    []string{"active", "inactive"},
	}}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.GET("/rules/filter-options", h.FilterOptions)

	req := httptest.NewRequest(http.MethodGet, "/rules/filter-options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		OK      bool                    `json:"ok"`
		Options store.RuleFilterOptions `json:"options"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || len(resp.Options.EventTypes) != 2 || len(resp.Options.ActiveStates) != 2 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestRulesFilterOptions_StoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRulesHandler(&mockRulesStore{filterErr: errors.New("boom")})
	r := gin.New()
	r.GET("/rules/filter-options", h.FilterOptions)

	req := httptest.NewRequest(http.MethodGet, "/rules/filter-options", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", w.Code, w.Body.String())
	}
}
