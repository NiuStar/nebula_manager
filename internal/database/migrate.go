package database

import (
	"log"

	"nebula_manager/internal/models"
)

// AutoMigrate runs Gorm migrations for the application's models.
func AutoMigrate() {
	conn := DB()
	if err := conn.AutoMigrate(&models.CA{}, &models.ConfigTemplate{}, &models.NetworkSetting{}, &models.Node{}, &models.NodePing{}, &models.NodeStatus{}); err != nil {
		log.Fatalf("auto migration failed: %v", err)
	}
}
