package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func contextPermissions(c *gin.Context) []string {
	raw, exists := c.Get("permissions")
	if !exists {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, strings.ToLower(strings.TrimSpace(s)))
			}
		}
		return out
	default:
		return nil
	}
}

func hasRequiredPermission(perms []string, required string) bool {
	required = strings.ToLower(strings.TrimSpace(required))
	if required == "" {
		return true
	}

	hasAdmin := false
	for _, p := range perms {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "admin":
			hasAdmin = true
		case required:
			return true
		}
	}

	if hasAdmin {
		return true
	}
	return false
}

func RequirePermission(permission string) gin.HandlerFunc {
	required := strings.ToLower(strings.TrimSpace(permission))
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		perms := contextPermissions(c)
		if !hasRequiredPermission(perms, required) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"ok":      false,
				"message": "forbidden: insufficient permissions",
			})
			return
		}
		c.Next()
	}
}

func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	required := make([]string, 0, len(permissions))
	for _, p := range permissions {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			required = append(required, p)
		}
	}

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		perms := contextPermissions(c)
		for _, p := range required {
			if hasRequiredPermission(perms, p) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"ok":      false,
			"message": "forbidden: insufficient permissions",
		})
	}
}

func RequireDangerConfirm() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		confirmValue := strings.ToLower(strings.TrimSpace(c.GetHeader("X-MF-Confirm")))
		if confirmValue != "confirm" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"ok":      false,
				"message": "missing danger confirmation header",
			})
			return
		}
		c.Next()
	}
}
