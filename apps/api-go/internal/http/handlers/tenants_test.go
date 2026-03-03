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

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
)

type mockTenantStore struct {
	items           []store.TenantRecord
	listErr         error
	createErr       error
	lastCreateID    string
	lastCreateName  string
	updateErr       error
	lastUpdateID    string
	lastUpdateState bool
}

func (m *mockTenantStore) ListTenants(_ context.Context) ([]store.TenantRecord, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *mockTenantStore) CreateTenant(_ context.Context, id string, name string) error {
	m.lastCreateID = id
	m.lastCreateName = name
	return m.createErr
}

func (m *mockTenantStore) UpdateTenantActive(_ context.Context, id string, isActive bool) error {
	m.lastUpdateID = id
	m.lastUpdateState = isActive
	return m.updateErr
}

func TestTenantsList_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockTenantStore{
		items: []store.TenantRecord{{
			ID:        "default",
			Name:      "Default Tenant",
			IsActive:  true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}},
	}
	h := NewTenantsHandler(mockStore)
	r := gin.New()
	r.GET("/api/tenants", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/tenants", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		OK    bool  `json:"ok"`
		Items []any `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if !body.OK || len(body.Items) != 1 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestTenantsCreate_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTenantsHandler(&mockTenantStore{})
	r := gin.New()
	r.POST("/api/tenants", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/tenants", strings.NewReader(`{"id":"!bad","name":"Bad"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTenantsCreate_Duplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTenantsHandler(&mockTenantStore{createErr: errors.New("duplicate key value violates unique constraint")})
	r := gin.New()
	r.POST("/api/tenants", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/tenants", strings.NewReader(`{"id":"team-a","name":"Team A"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTenantsUpdateActive_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockTenantStore{}
	h := NewTenantsHandler(mockStore)
	r := gin.New()
	r.PATCH("/api/tenants/:id/active", h.UpdateActive)

	req := httptest.NewRequest(http.MethodPatch, "/api/tenants/team-a/active", strings.NewReader(`{"is_active":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if mockStore.lastUpdateID != "team-a" || mockStore.lastUpdateState {
		t.Fatalf("unexpected update args: id=%s isActive=%v", mockStore.lastUpdateID, mockStore.lastUpdateState)
	}
}
