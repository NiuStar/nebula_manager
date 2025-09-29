package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"nebula_manager/internal/middleware"
	"nebula_manager/internal/services"
)

// AuthHandler exposes login/logout endpoints.
type AuthHandler struct {
	service *services.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login verifies credentials and issues a session cookie.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login payload"})
		return
	}

	if !h.service.ValidateCredentials(req.Username, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, expiresAt, err := h.service.IssueToken(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue session"})
		return
	}

	setSessionCookie(c, h.service.CookieName(), token, expiresAt, h.service.SecureCookies())

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"username": req.Username, "expires_at": expiresAt.UTC(), "token": token}})
}

// Logout removes the session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	setSessionCookie(c, h.service.CookieName(), "", time.Unix(0, 0), h.service.SecureCookies())
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

// Profile returns current authenticated user.
func (h *AuthHandler) Profile(c *gin.Context) {
	user, ok := c.Get(middleware.ContextUserKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"username": user}})
}

func setSessionCookie(c *gin.Context, name, value string, expiresAt time.Time, secure bool) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
	if value == "" {
		cookie.Expires = time.Unix(0, 0)
		cookie.MaxAge = -1
	}
	http.SetCookie(c.Writer, cookie)
}
