package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler("admin", "pass123", "jwt-secret", time.Hour)
	r := gin.New()
	r.POST("/auth/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"admin","password":"pass123"}`))
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
	token, err := issueJWT("admin", "jwt-secret", time.Hour)
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

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token, err := issueJWT("admin", "jwt-secret", -1*time.Minute)
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
