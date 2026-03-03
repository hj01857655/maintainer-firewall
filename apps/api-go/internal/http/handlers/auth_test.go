package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"maintainer-firewall/api-go/internal/store"
	"maintainer-firewall/api-go/internal/tenantctx"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type mockAuthStore struct {
	userToReturn    *store.AdminUser
	errToReturn     error
	updatedUserID   int64
	updatedLoginAt  time.Time
	updateCallCount int
}

func (m *mockAuthStore) GetAdminUserByUsername(_ context.Context, _ string) (store.AdminUser, error) {
	if m.errToReturn != nil {
		return store.AdminUser{}, m.errToReturn
	}
	if m.userToReturn == nil {
		return store.AdminUser{}, fmt.Errorf("admin user not found")
	}
	return *m.userToReturn, nil
}

func (m *mockAuthStore) UpdateAdminUserLastLogin(_ context.Context, id int64, at time.Time) error {
	m.updatedUserID = id
	m.updatedLoginAt = at
	m.updateCallCount++
	return nil
}

func TestAuthLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler("admin", "pass123", "jwt-secret", time.Hour)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"pass123","tenant_id":"team-a"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	token, _ := body["token"].(string)
	if strings.TrimSpace(token) == "" {
		t.Fatalf("expected non-empty jwt token, got %s", w.Body.String())
	}

	parsed, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		return []byte("jwt-secret"), nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		t.Fatalf("expected valid jwt token, err=%v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("expected map claims")
	}
	if claims["tenant_id"] != "team-a" {
		t.Fatalf("expected tenant_id=team-a, got %#v", claims["tenant_id"])
	}
	if claims["role"] != "admin" {
		t.Fatalf("expected role=admin, got %#v", claims["role"])
	}
	perms, ok := claims["permissions"].([]any)
	if !ok || len(perms) != 3 {
		t.Fatalf("expected permissions length=3, got %#v", claims["permissions"])
	}
}

func TestAuthLogin_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler("admin", "pass123", "jwt-secret", time.Hour)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthLogin_UsesDatabaseUserWhenAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hash, err := bcrypt.GenerateFromPassword([]byte("db-pass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}

	mockStore := &mockAuthStore{userToReturn: &store.AdminUser{ID: 7, Username: "admin", PasswordHash: string(hash), IsActive: true, Role: "admin", Permissions: []string{"read", "write", "admin"}}}
	h := NewAuthHandlerWithStore(mockStore, "admin", "env-pass", "jwt-secret", time.Hour, true)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"db-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if mockStore.updateCallCount != 1 || mockStore.updatedUserID != 7 {
		t.Fatalf("expected update last login called once for user 7, got calls=%d user=%d", mockStore.updateCallCount, mockStore.updatedUserID)
	}
}

func TestAuthLogin_DatabaseUserDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hash, err := bcrypt.GenerateFromPassword([]byte("db-pass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}

	mockStore := &mockAuthStore{userToReturn: &store.AdminUser{ID: 8, Username: "admin", PasswordHash: string(hash), IsActive: false}}
	h := NewAuthHandlerWithStore(mockStore, "admin", "env-pass", "jwt-secret", time.Hour, true)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"db-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAuthLogin_NoDbUserAndFallbackDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockStore := &mockAuthStore{errToReturn: fmt.Errorf("admin user not found")}
	h := NewAuthHandlerWithStore(mockStore, "admin", "env-pass", "jwt-secret", time.Hour, false)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"env-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAuthLogin_LockoutAfterTooManyFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler("admin", "pass123", "jwt-secret", time.Hour)
	h.MaxFailedAttempts = 3
	h.RateLimitWindow = time.Hour
	h.LockoutDuration = 5 * time.Minute

	r := gin.New()
	r.POST("/auth/login", h.Login)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d expected 401, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after lockout, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthLogin_SuccessClearsFailureWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler("admin", "pass123", "jwt-secret", time.Hour)
	h.MaxFailedAttempts = 2
	h.RateLimitWindow = time.Hour
	h.LockoutDuration = 5 * time.Minute

	r := gin.New()
	r.POST("/auth/login", h.Login)

	bad := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	bad.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, bad)
	if w1.Code != http.StatusUnauthorized {
		t.Fatalf("expected first bad login 401, got %d", w1.Code)
	}

	good := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"pass123"}`))
	good.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, good)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected good login 200, got %d body=%s", w2.Code, w2.Body.String())
	}

	bad2 := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	bad2.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, bad2)
	if w3.Code != http.StatusUnauthorized {
		t.Fatalf("expected bad login after success to be 401, got %d", w3.Code)
	}
}

func TestAuthMiddleware_RequiresToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware("jwt-secret"))
	r.GET("/protected", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token, err := issueJWT("admin", "team-a", "editor", []string{"read", "write"}, "jwt-secret", time.Hour)
	if err != nil {
		t.Fatalf("issue jwt: %v", err)
	}

	r := gin.New()
	r.Use(AuthMiddleware("jwt-secret"))
	r.GET("/protected", func(c *gin.Context) {
		ctxTenantID, ok := tenantctx.FromContext(c.Request.Context())
		c.JSON(200, gin.H{
			"ok":            true,
			"tenant_id":     c.GetString("tenant_id"),
			"role":          c.GetString("role"),
			"permissions":   c.GetStringSlice("permissions"),
			"ctx_ok":        ok,
			"ctx_tenant_id": ctxTenantID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["tenant_id"] != "team-a" {
		t.Fatalf("expected middleware tenant_id=team-a, got %#v", body["tenant_id"])
	}
	if body["role"] != "editor" {
		t.Fatalf("expected middleware role=editor, got %#v", body["role"])
	}
	if body["ctx_ok"] != true || body["ctx_tenant_id"] != "team-a" {
		t.Fatalf("expected request context tenant_id=team-a, got body=%s", w.Body.String())
	}
	perms, ok := body["permissions"].([]any)
	if !ok || len(perms) != 2 {
		t.Fatalf("expected middleware permissions [read write], got %#v", body["permissions"])
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token, err := issueJWT("admin", "team-a", "admin", []string{"read", "write", "admin"}, "jwt-secret", -1*time.Minute)
	if err != nil {
		t.Fatalf("issue jwt: %v", err)
	}

	r := gin.New()
	r.Use(AuthMiddleware("jwt-secret"))
	r.GET("/protected", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthMiddleware_LegacyTokenWithoutTenantClaimFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	legacyClaims := jwt.MapClaims{
		"sub": "admin",
		"iat": time.Now().UTC().Unix(),
		"exp": time.Now().UTC().Add(time.Hour).Unix(),
	}
	legacyToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyClaims).SignedString([]byte("jwt-secret"))
	if err != nil {
		t.Fatalf("issue legacy token: %v", err)
	}

	r := gin.New()
	r.Use(AuthMiddleware("jwt-secret"))
	r.GET("/protected", func(c *gin.Context) {
		ctxTenantID, ok := tenantctx.FromContext(c.Request.Context())
		c.JSON(200, gin.H{
			"tenant_id":     c.GetString("tenant_id"),
			"role":          c.GetString("role"),
			"permissions":   c.GetStringSlice("permissions"),
			"ctx_ok":        ok,
			"ctx_tenant_id": ctxTenantID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+legacyToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["tenant_id"] != tenantctx.DefaultTenantID {
		t.Fatalf("expected default tenant fallback, got %#v", body["tenant_id"])
	}
	if body["role"] != "admin" {
		t.Fatalf("expected legacy token role fallback admin, got %#v", body["role"])
	}
	if body["ctx_ok"] != true || body["ctx_tenant_id"] != tenantctx.DefaultTenantID {
		t.Fatalf("expected context default tenant fallback, got body=%s", w.Body.String())
	}
	perms, ok := body["permissions"].([]any)
	if !ok || len(perms) != 3 {
		t.Fatalf("expected legacy token permissions fallback, got %#v", body["permissions"])
	}
}
