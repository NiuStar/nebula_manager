package handlers

import (
	"net/http"
	"strconv"
	"time"

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

// NetworkStatus returns latency series data for a node.
func (h *NodeHandler) NetworkStatus(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}

	span := parseRangeParam(c.Query("range"))
	series, err := h.service.GetNetworkSeries(id, span)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": series})
}

// SubmitNetworkSamples stores latency data reported by a node agent.
func (h *NodeHandler) SubmitNetworkSamples(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}

	var req struct {
		Samples []services.NetworkSampleInput `json:"samples" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RecordNetworkSamples(id, req.Samples); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

// NetworkTargets lists recommended probe targets for the node agent.
func (h *NodeHandler) NetworkTargets(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}

	targets, err := h.service.ListNetworkTargets(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": targets})
}

func parseUintParam(val string) (uint, error) {
	parsed, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}

func parseRangeParam(val string) time.Duration {
	switch val {
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "1h", "":
		return time.Hour
	default:
		return time.Hour
	}
}
