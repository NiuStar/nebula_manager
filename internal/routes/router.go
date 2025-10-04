package routes

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nebula_manager/internal/handlers"
	"nebula_manager/internal/middleware"
	"nebula_manager/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Dependencies collects request handlers used by the router.
type Dependencies struct {
	CA        *handlers.CAHandler
	Settings  *handlers.SettingsHandler
	Templates *handlers.TemplateHandler
	Nodes     *handlers.NodeHandler
	Auth      *handlers.AuthHandler
	AuthSvc   *services.AuthService
}

// New constructs the Gin router with all application routes.
func New(deps Dependencies, staticDir string) *gin.Engine {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:       12 * time.Hour,
	}))

	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/api/login", deps.Auth.Login)
	router.POST("/api/logout", deps.Auth.Logout)
	router.GET("/api/public/status", deps.Nodes.PublicStatus)
	router.GET("/api/public/nodes/:id/network", deps.Nodes.PublicNetworkStatus)

	protected := router.Group("/api")
	protected.Use(middleware.RequireAuth(deps.AuthSvc))

	protected.GET("/ca", deps.CA.Get)
	protected.POST("/ca", deps.CA.Generate)
	protected.GET("/ca/certificate", deps.CA.Certificate)

	protected.GET("/settings", deps.Settings.Get)
	protected.PUT("/settings", deps.Settings.Update)

	protected.GET("/templates", deps.Templates.List)
	protected.POST("/templates", deps.Templates.Upsert)
	protected.DELETE("/templates/:id", deps.Templates.Delete)

	protected.GET("/nodes", deps.Nodes.List)
	protected.POST("/nodes", deps.Nodes.Create)
	protected.GET("/nodes/:id/artifacts", deps.Nodes.Artifacts)
	protected.GET("/nodes/:id/config", deps.Nodes.Config)
	protected.GET("/nodes/:id/install-script", deps.Nodes.InstallScript)
	protected.GET("/nodes/:id/bundle", deps.Nodes.Bundle)
	protected.GET("/nodes/:id/network", deps.Nodes.NetworkStatus)
	protected.POST("/nodes/:id/status", deps.Nodes.SubmitStatus)
	protected.GET("/nodes/:id/network/targets", deps.Nodes.NetworkTargets)
	protected.POST("/nodes/:id/network/samples", deps.Nodes.SubmitNetworkSamples)
	protected.DELETE("/nodes/:id", deps.Nodes.Delete)
	protected.GET("/me", deps.Auth.Profile)

	if staticDir != "" {
		indexFile := filepath.Join(staticDir, "index.html")
		router.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}

			requested := filepath.Clean(c.Request.URL.Path)
			if requested == "." || requested == "/" {
				c.File(indexFile)
				return
			}

			assetPath := filepath.Join(staticDir, requested)
			if info, err := os.Stat(assetPath); err == nil && !info.IsDir() {
				c.File(assetPath)
				return
			}

			c.File(indexFile)
		})
	} else {
		router.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})
	}

	return router
}
