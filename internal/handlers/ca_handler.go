package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nebula_manager/internal/models"
	"nebula_manager/internal/services"
)

// CAHandler exposes endpoints to manage certificate authorities.
type CAHandler struct {
	service *services.CAService
}

// NewCAHandler creates a new handler instance.
func NewCAHandler(service *services.CAService) *CAHandler {
	return &CAHandler{service: service}
}

// Get returns the current CA metadata.
func (h *CAHandler) Get(c *gin.Context) {
	ca, err := h.service.GetCA()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ca == nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCA(ca)})
}

// Generate creates or replaces the CA using the provided payload.
func (h *CAHandler) Generate(c *gin.Context) {
	var req services.CreateCARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ca, err := h.service.GenerateOrReplaceCA(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCA(ca)})
}

// Certificate returns the CA certificate PEM for download.
func (h *CAHandler) Certificate(c *gin.Context) {
	ca, err := h.service.GetCA()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ca == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CA not found"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=nebula-ca.crt")
	c.Data(http.StatusOK, "application/x-pem-file", []byte(ca.CertificatePEM))
}

func presentCA(ca *models.CA) gin.H {
	return gin.H{
		"id":          ca.ID,
		"name":        ca.Name,
		"description": ca.Description,
		"created_at":  ca.CreatedAt,
	}
}
