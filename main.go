package main

import (
	"fmt"
	"log"

	"nebula_manager/internal/config"
	"nebula_manager/internal/database"
	"nebula_manager/internal/handlers"
	"nebula_manager/internal/routes"
	"nebula_manager/internal/services"
)

func main() {
	cfg := config.Load()

	conn := database.Connect(cfg)
	if cfg.EnableAutoMigrate {
		database.AutoMigrate()
	}

	caService := services.NewCAService(conn)
	templateService := services.NewTemplateService(conn)
	settingsService := services.NewSettingsService(conn)
	authService := services.NewAuthService(cfg.AdminUsername, cfg.AdminPassword, cfg.SessionSecret, cfg.SessionSecureCookie, cfg.StaticAccessToken)
	nodeService := services.NewNodeService(conn, caService, templateService, settingsService, cfg.DataDir, cfg.APIBaseURL, cfg.NebulaVersion, cfg.NebulaDownloadBase, cfg.NebulaProxyPrefix, cfg.StaticAccessToken)

	router := routes.New(routes.Dependencies{
		CA:        handlers.NewCAHandler(caService),
		Settings:  handlers.NewSettingsHandler(settingsService),
		Templates: handlers.NewTemplateHandler(templateService),
		Nodes:     handlers.NewNodeHandler(nodeService),
		Auth:      handlers.NewAuthHandler(authService),
		AuthSvc:   authService,
	}, cfg.FrontendDir)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
