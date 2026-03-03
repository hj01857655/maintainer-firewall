package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"maintainer-firewall/api-go/internal/store"
	"maintainer-firewall/api-go/internal/tenantctx"

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

	RateLimitWindow   time.Duration
	MaxFailedAttempts int
	LockoutDuration   time.Duration
	mu                sync.Mutex
	failedAttempts    map[string]int
	firstFailedAt     map[string]time.Time
	lockedUntil       map[string]time.Time
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TenantID string `json:"tenant_id"`
}

var supportedPermissions = []string{"read", "write", "admin"}

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
	tenantID := tenantctx.MustFromContext(nil, req.TenantID)
	if username == "" || password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid username or password"})
		return
	}

	if wait, locked := h.checkLocked(username); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{"ok": false, "message": fmt.Sprintf("too many failed attempts, try again in %d seconds", wait)})
		return
	}

	if h.Store != nil {
		storeCtx := tenantctx.WithTenantID(c.Request.Context(), tenantID)
		adminUser, err := h.Store.GetAdminUserByUsername(storeCtx, username)
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
			role, permissions := normalizeRoleAndPermissions(adminUser.Role, adminUser.Permissions, false)
			token, issueErr := issueJWT(adminUser.Username, tenantID, role, permissions, h.JWTSecret, h.TokenTTL)
			if issueErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "failed to create token"})
				return
			}
			h.clearFailures(username)
			_ = h.Store.UpdateAdminUserLastLogin(storeCtx, adminUser.ID, time.Now().UTC())
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

	role, permissions := normalizeRoleAndPermissions("admin", []string{"read", "write", "admin"}, false)
	token, err := issueJWT(h.AdminUsername, tenantID, role, permissions, h.JWTSecret, h.TokenTTL)
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
			tenantID := tenantctx.DefaultTenantID
			if claimTenantID, ok := claims["tenant_id"].(string); ok {
				tenantID = tenantctx.MustFromContext(nil, claimTenantID)
			}
			role, permissions := claimsRoleAndPermissions(claims)
			c.Set("role", role)
			c.Set("permissions", permissions)
			c.Set("tenant_id", tenantID)
			c.Request = c.Request.WithContext(tenantctx.WithTenantID(c.Request.Context(), tenantID))
		} else {
			c.Set("role", "admin")
			c.Set("permissions", []string{"read", "write", "admin"})
			c.Set("tenant_id", tenantctx.DefaultTenantID)
			c.Request = c.Request.WithContext(tenantctx.WithTenantID(c.Request.Context(), tenantctx.DefaultTenantID))
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

func issueJWT(subject string, tenantID string, role string, permissions []string, secret string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	role, permissions = normalizeRoleAndPermissions(role, permissions, false)
	claims := jwt.MapClaims{
		"sub":         strings.TrimSpace(subject),
		"tenant_id":   tenantctx.MustFromContext(nil, tenantID),
		"role":        role,
		"permissions": permissions,
		"iat":         now.Unix(),
		"exp":         now.Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(strings.TrimSpace(secret)))
}

func claimsRoleAndPermissions(claims jwt.MapClaims) (string, []string) {
	role := ""
	if v, ok := claims["role"].(string); ok {
		role = v
	}
	rawPermissions := make([]string, 0, 4)
	switch v := claims["permissions"].(type) {
	case []string:
		rawPermissions = append(rawPermissions, v...)
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				rawPermissions = append(rawPermissions, s)
			}
		}
	case string:
		rawPermissions = append(rawPermissions, v)
	}
	return normalizeRoleAndPermissions(role, rawPermissions, true)
}

func normalizeRoleAndPermissions(role string, permissions []string, legacyAdminFallback bool) (string, []string) {
	role = strings.ToLower(strings.TrimSpace(role))
	normPerms := normalizePermissions(permissions)

	if role != "admin" && role != "editor" && role != "viewer" {
		role = ""
	}

	if len(normPerms) == 0 {
		if role == "" {
			if legacyAdminFallback {
				return "admin", []string{"read", "write", "admin"}
			}
			return "viewer", []string{"read"}
		}
		switch role {
		case "admin":
			return "admin", []string{"read", "write", "admin"}
		case "editor":
			return "editor", []string{"read", "write"}
		default:
			return "viewer", []string{"read"}
		}
	}

	// admin implies all permissions
	if hasPermission(normPerms, "admin") {
		return "admin", []string{"read", "write", "admin"}
	}
	// write implies read
	if hasPermission(normPerms, "write") && !hasPermission(normPerms, "read") {
		normPerms = append(normPerms, "read")
		normPerms = normalizePermissions(normPerms)
	}

	if role == "" {
		if hasPermission(normPerms, "write") {
			role = "editor"
		} else {
			role = "viewer"
		}
	}
	return role, normPerms
}

func normalizePermissions(permissions []string) []string {
	seen := map[string]bool{}
	for _, p := range permissions {
		v := strings.ToLower(strings.TrimSpace(p))
		for _, supported := range supportedPermissions {
			if v == supported {
				seen[v] = true
			}
		}
	}

	out := make([]string, 0, len(supportedPermissions))
	for _, p := range supportedPermissions {
		if seen[p] {
			out = append(out, p)
		}
	}
	return out
}

func hasPermission(permissions []string, target string) bool {
	for _, p := range permissions {
		if p == target {
			return true
		}
	}
	return false
}
