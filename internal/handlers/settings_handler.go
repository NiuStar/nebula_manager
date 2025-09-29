package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nebula_manager/internal/services"
)

// SettingsHandler exposes network settings endpoints.
type SettingsHandler struct {
	service *services.SettingsService
}

// NewSettingsHandler constructs a handler instance.
func NewSettingsHandler(service *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

// Get returns the singleton network settings object.
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.service.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// Update mutates the network settings.
func (h *SettingsHandler) Update(c *gin.Context) {
	var req services.UpdateNetworkSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings, err := h.service.Update(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}
