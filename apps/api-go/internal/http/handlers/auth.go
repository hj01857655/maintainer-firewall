package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	AdminUsername string
	AdminPassword string
	JWTSecret     string
	TokenTTL      time.Duration
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(adminUsername string, adminPassword string, jwtSecret string, tokenTTL time.Duration) *AuthHandler {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &AuthHandler{
		AdminUsername: strings.TrimSpace(adminUsername),
		AdminPassword: strings.TrimSpace(adminPassword),
		JWTSecret:     strings.TrimSpace(jwtSecret),
		TokenTTL:      tokenTTL,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	if h.AdminUsername == "" || h.AdminPassword == "" || h.JWTSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "auth is not configured"})
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid JSON payload"})
		return
	}

	if strings.TrimSpace(req.Username) != h.AdminUsername || strings.TrimSpace(req.Password) != h.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid username or password"})
		return
	}

	token, err := issueJWT(h.AdminUsername, h.JWTSecret, h.TokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "failed to create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "token": token})
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	secret := strings.TrimSpace(jwtSecret)
	return func(c *gin.Context) {
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
