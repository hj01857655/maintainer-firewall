package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	AdminUsername string
	AdminPassword string
	AccessToken   string
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(adminUsername string, adminPassword string, accessToken string) *AuthHandler {
	return &AuthHandler{
		AdminUsername: strings.TrimSpace(adminUsername),
		AdminPassword: strings.TrimSpace(adminPassword),
		AccessToken:   strings.TrimSpace(accessToken),
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	if h.AdminUsername == "" || h.AdminPassword == "" || h.AccessToken == "" {
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

	c.JSON(http.StatusOK, gin.H{"ok": true, "token": h.AccessToken})
}

func AuthMiddleware(accessToken string) gin.HandlerFunc {
	token := strings.TrimSpace(accessToken)
	return func(c *gin.Context) {
		if token == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "auth token is not configured"})
			return
		}

		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "missing bearer token"})
			return
		}

		provided := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if provided != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid token"})
			return
		}

		c.Next()
	}
}
