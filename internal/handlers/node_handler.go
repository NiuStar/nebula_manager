package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nebula_manager/internal/services"
)

// NodeHandler exposes endpoints for Nebula nodes.
type NodeHandler struct {
	service *services.NodeService
}

// NewNodeHandler constructs a new handler.
func NewNodeHandler(service *services.NodeService) *NodeHandler {
	return &NodeHandler{service: service}
}

// List returns all nodes in the system.
func (h *NodeHandler) List(c *gin.Context) {
	nodes, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nodes})
}

// Create provisions a new node and returns its metadata.
func (h *NodeHandler) Create(c *gin.Context) {
	var req services.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := h.service.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": node})
}

// Artifacts returns the certificates and config for a node.
func (h *NodeHandler) Artifacts(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}
	artifacts, err := h.service.GetArtifacts(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": artifacts})
}

// Config returns the rendered configuration YAML for a node.
func (h *NodeHandler) Config(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}
	config, err := h.service.GetConfig(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/yaml", []byte(config))
}

// InstallScript returns a helper shell script to install node artifacts.
func (h *NodeHandler) InstallScript(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}
	script, err := h.service.GenerateInstallScript(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/plain", []byte(script))
}

// Bundle returns a downloadable zip archive for the node.
func (h *NodeHandler) Bundle(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}
	data, err := h.service.BuildBundle(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=nebula-node.tar.gz")
	c.Data(http.StatusOK, "application/gzip", data)
}

// Delete removes a node from the system.
func (h *NodeHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func parseUintParam(val string) (uint, error) {
	parsed, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}
