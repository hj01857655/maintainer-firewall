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
	versionLimit     int
	versionOffset    int
	updatedID        int64
	updatedIsActive  bool
	updateShouldFail bool
	filterOptions    store.RuleFilterOptions
	filterErr        error
	versionItems     []store.RuleVersionRecord
	versionTotal     int64
	publishVersion   int64
	publishCount     int
	publishErr       error
	publishActor     string
	publishSource    int64
	rollbackErr      error
	restoredCount    int
	versionRules     map[int64][]store.RuleRecord
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

func (m *mockRulesStore) CreateRuleVersionSnapshot(_ context.Context, actor string, sourceVersion int64) (int64, int, error) {
	m.publishActor = actor
	m.publishSource = sourceVersion
	if m.publishErr != nil {
		return 0, 0, m.publishErr
	}
	if m.publishVersion == 0 {
		m.publishVersion = 1
	}
	return m.publishVersion, m.publishCount, nil
}

func (m *mockRulesStore) ListRuleVersions(_ context.Context, limit int, offset int) ([]store.RuleVersionRecord, int64, error) {
	m.versionLimit = limit
	m.versionOffset = offset
	return m.versionItems, m.versionTotal, nil
}

func (m *mockRulesStore) GetRulesByVersion(_ context.Context, version int64) ([]store.RuleRecord, error) {
	if m.versionRules == nil {
		return nil, fmt.Errorf("rule version not found")
	}
	items, ok := m.versionRules[version]
	if !ok {
		return nil, fmt.Errorf("rule version not found")
	}
	return items, nil
}

func (m *mockRulesStore) RestoreRulesFromVersion(_ context.Context, _ int64) (int, error) {
	if m.rollbackErr != nil {
		return 0, m.rollbackErr
	}
	return m.restoredCount, nil
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

func TestRulesPublishVersion_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRulesHandler(&mockRulesStore{publishVersion: 3, publishCount: 7})
	r := gin.New()
	r.POST("/rules/publish", h.PublishVersion)

	req := httptest.NewRequest(http.MethodPost, "/rules/publish", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRulesPublishVersion_StoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRulesHandler(&mockRulesStore{publishErr: errors.New("db down")})
	r := gin.New()
	r.POST("/rules/publish", h.PublishVersion)

	req := httptest.NewRequest(http.MethodPost, "/rules/publish", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRulesListVersions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRulesHandler(&mockRulesStore{
		versionItems: []store.RuleVersionRecord{{Version: 2, RuleCount: 5, CreatedBy: "admin"}},
		versionTotal: 1,
	})
	r := gin.New()
	r.GET("/rules/versions", h.ListVersions)

	req := httptest.NewRequest(http.MethodGet, "/rules/versions?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRulesListVersions_LimitOffsetBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.GET("/rules/versions", h.ListVersions)

	req := httptest.NewRequest(http.MethodGet, "/rules/versions?limit=999&offset=-10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if mockStore.versionLimit != 100 || mockStore.versionOffset != 0 {
		t.Fatalf("expected clamped limit/offset = 100/0, got %d/%d", mockStore.versionLimit, mockStore.versionOffset)
	}
}

func TestRulesRollback_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRulesHandler(&mockRulesStore{rollbackErr: fmt.Errorf("rule version not found")})
	r := gin.New()
	r.POST("/rules/rollback", h.Rollback)

	req := httptest.NewRequest(http.MethodPost, "/rules/rollback", strings.NewReader(`{"version":9}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRulesRollback_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{
		restoredCount:  9,
		publishVersion: 12,
	}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.POST("/rules/rollback", h.Rollback)

	req := httptest.NewRequest(http.MethodPost, "/rules/rollback", strings.NewReader(`{"version":5}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK            bool  `json:"ok"`
		FromVersion   int64 `json:"from_version"`
		ToVersion     int64 `json:"to_version"`
		RestoredCount int   `json:"restored_count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || resp.FromVersion != 5 || resp.ToVersion != 12 || resp.RestoredCount != 9 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
	if mockStore.publishSource != 5 {
		t.Fatalf("expected rollback snapshot source version=5, got %d", mockStore.publishSource)
	}
}

func TestRulesReplay_ByVersionSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRulesHandler(&mockRulesStore{
		versionRules: map[int64][]store.RuleRecord{
			2: {
				{
					EventType:       "issues",
					Keyword:         "urgent",
					SuggestionType:  "label",
					SuggestionValue: "P0",
					Reason:          "r",
					IsActive:        true,
				},
			},
		},
	})
	r := gin.New()
	r.POST("/rules/replay", h.Replay)

	body := `{"version":2,"event_type":"issues","payload":{"issue":{"title":"urgent bug","body":"please fix"}}}`
	req := httptest.NewRequest(http.MethodPost, "/rules/replay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRulesReplay_CurrentActiveSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockRulesStore{
		items: []store.RuleRecord{
			{
				EventType:       "issues",
				Keyword:         "urgent",
				SuggestionType:  "label",
				SuggestionValue: "P0",
				Reason:          "r",
				IsActive:        true,
			},
		},
		total: 1,
	}
	h := NewRulesHandler(mockStore)
	r := gin.New()
	r.POST("/rules/replay", h.Replay)

	body := `{"event_type":"issues","payload":{"issue":{"title":"urgent bug"}}}`
	req := httptest.NewRequest(http.MethodPost, "/rules/replay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if mockStore.lastLimit != 1000 || mockStore.lastOffset != 0 || mockStore.lastEvent != "" || mockStore.lastKey != "" || !mockStore.lastActive {
		t.Fatalf("unexpected replay list args: %+v", mockStore)
	}

	var resp struct {
		OK          bool  `json:"ok"`
		Version     int64 `json:"version"`
		RuleCount   int   `json:"rule_count"`
		Suggestions []any `json:"suggestions"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.OK || resp.Version != 0 || resp.RuleCount != 1 || len(resp.Suggestions) == 0 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestRulesReplay_InvalidEventType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRulesHandler(&mockRulesStore{})
	r := gin.New()
	r.POST("/rules/replay", h.Replay)

	req := httptest.NewRequest(http.MethodPost, "/rules/replay", strings.NewReader(`{"event_type":"push","payload":{}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}
