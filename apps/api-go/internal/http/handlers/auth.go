package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"maintainer-firewall/api-go/internal/store"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthStore interface {
	GetAdminUserByUsername(ctx context.Context, username string) (store.AdminUser, error)
	UpdateAdminUserLastLogin(ctx context.Context, id int64, at time.Time) error
}

type AuthHandler struct {
	Store            AuthStore
	AdminUsername    string
	AdminPassword    string
	JWTSecret        string
	TokenTTL         time.Duration
	AllowEnvFallback bool

	RateLimitWindow    time.Duration
	MaxFailedAttempts  int
	LockoutDuration    time.Duration
	mu                 sync.Mutex
	failedAttempts     map[string]int
	firstFailedAt      map[string]time.Time
	lockedUntil        map[string]time.Time
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(adminUsername string, adminPassword string, jwtSecret string, tokenTTL time.Duration) *AuthHandler {
	return NewAuthHandlerWithStore(nil, adminUsername, adminPassword, jwtSecret, tokenTTL, true)
}

func NewAuthHandlerWithStore(authStore AuthStore, adminUsername string, adminPassword string, jwtSecret string, tokenTTL time.Duration, allowEnvFallback bool) *AuthHandler {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &AuthHandler{
		Store:             authStore,
		AdminUsername:     strings.TrimSpace(adminUsername),
		AdminPassword:     strings.TrimSpace(adminPassword),
		JWTSecret:         strings.TrimSpace(jwtSecret),
		TokenTTL:          tokenTTL,
		AllowEnvFallback:  allowEnvFallback,
		RateLimitWindow:   10 * time.Minute,
		MaxFailedAttempts: 5,
		LockoutDuration:   15 * time.Minute,
		failedAttempts:    map[string]int{},
		firstFailedAt:     map[string]time.Time{},
		lockedUntil:       map[string]time.Time{},
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	if h.JWTSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "auth is not configured"})
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid JSON payload"})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid username or password"})
		return
	}

	if wait, locked := h.checkLocked(username); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{"ok": false, "message": fmt.Sprintf("too many failed attempts, try again in %d seconds", wait)})
		return
	}

	if h.Store != nil {
		adminUser, err := h.Store.GetAdminUserByUsername(c.Request.Context(), username)
		if err == nil {
			if !adminUser.IsActive {
				h.recordFailure(username)
				c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "admin user is disabled"})
				return
			}
			if bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(password)) != nil {
				h.recordFailure(username)
				c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid username or password"})
				return
			}
			token, issueErr := issueJWT(adminUser.Username, h.JWTSecret, h.TokenTTL)
			if issueErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "failed to create token"})
				return
			}
			h.clearFailures(username)
			_ = h.Store.UpdateAdminUserLastLogin(c.Request.Context(), adminUser.ID, time.Now().UTC())
			c.JSON(http.StatusOK, gin.H{"ok": true, "token": token})
			return
		}

		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "not found") {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "load admin user failed"})
			return
		}
		if !h.AllowEnvFallback {
			h.recordFailure(username)
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid username or password"})
			return
		}
	}

	if h.AdminUsername == "" || h.AdminPassword == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "auth is not configured"})
		return
	}
	if username != h.AdminUsername || password != h.AdminPassword {
		h.recordFailure(username)
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid username or password"})
		return
	}

	token, err := issueJWT(h.AdminUsername, h.JWTSecret, h.TokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "failed to create token"})
		return
	}

	h.clearFailures(username)
	c.JSON(http.StatusOK, gin.H{"ok": true, "token": token})
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	secret := strings.TrimSpace(jwtSecret)
	return func(c *gin.Context) {
		// 允许OPTIONS预检请求跳过认证
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		if secret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "auth token is not configured"})
			return
		}

		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "missing bearer token"})
			return
		}

		provided := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		parsed, err := jwt.Parse(provided, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || parsed == nil || !parsed.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid bearer token"})
			return
		}

		if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
			if sub, ok := claims["sub"].(string); ok {
				c.Set("actor", strings.TrimSpace(sub))
			}
		}

		c.Next()
	}
}

func (h *AuthHandler) checkLocked(username string) (int64, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UTC()
	if until, ok := h.lockedUntil[username]; ok {
		if now.Before(until) {
			return int64(until.Sub(now).Seconds()), true
		}
		delete(h.lockedUntil, username)
	}
	return 0, false
}

func (h *AuthHandler) recordFailure(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().UTC()
	if h.RateLimitWindow <= 0 {
		h.RateLimitWindow = 10 * time.Minute
	}
	if h.LockoutDuration <= 0 {
		h.LockoutDuration = 15 * time.Minute
	}
	if h.MaxFailedAttempts <= 0 {
		h.MaxFailedAttempts = 5
	}

	first, ok := h.firstFailedAt[username]
	if !ok || now.Sub(first) > h.RateLimitWindow {
		h.firstFailedAt[username] = now
		h.failedAttempts[username] = 1
		return
	}

	h.failedAttempts[username] = h.failedAttempts[username] + 1
	if h.failedAttempts[username] >= h.MaxFailedAttempts {
		h.lockedUntil[username] = now.Add(h.LockoutDuration)
		h.failedAttempts[username] = 0
		h.firstFailedAt[username] = time.Time{}
	}
}

func (h *AuthHandler) clearFailures(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.failedAttempts, username)
	delete(h.firstFailedAt, username)
	delete(h.lockedUntil, username)
}

func issueJWT(subject string, secret string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub": strings.TrimSpace(subject),
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(strings.TrimSpace(secret)))
}
