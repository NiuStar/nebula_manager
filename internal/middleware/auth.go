package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"nebula_manager/internal/services"
)

// ContextUserKey is used to store the authenticated username in Gin context.
const ContextUserKey = "authUser"

// RequireAuth ensures the request carries a valid session token.
func RequireAuth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c, authService.CookieName())
		if token != "" {
			if static := authService.StaticToken(); static != "" && subtle.ConstantTimeCompare([]byte(token), []byte(static)) == 1 {
				c.Set(ContextUserKey, authService.AdminUsername())
				c.Next()
				return
			}
		}
		username, _, err := authService.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set(ContextUserKey, username)
		c.Next()
	}
}

func extractToken(c *gin.Context, cookieName string) string {
	if cookie, err := c.Cookie(cookieName); err == nil && cookie != "" {
		return cookie
	}
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	if token := c.Query("access_token"); token != "" {
		return token
	}
	return ""
}
