package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"nebula_manager/internal/models"
	"nebula_manager/internal/services"
)

// TemplateHandler exposes CRUD endpoints for config templates.
type TemplateHandler struct {
	service *services.TemplateService
}

// NewTemplateHandler constructs a handler.
func NewTemplateHandler(service *services.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

// List returns all templates.
func (h *TemplateHandler) List(c *gin.Context) {
	templates, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// Upsert creates or updates a template.
func (h *TemplateHandler) Upsert(c *gin.Context) {
	var payload models.ConfigTemplate
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template name required"})
		return
	}
	if err := h.service.Upsert(&payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

// Delete removes a template by ID.
func (h *TemplateHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
