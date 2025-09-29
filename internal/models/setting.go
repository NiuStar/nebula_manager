package models

import "time"

// NetworkSetting stores global configuration values for the managed Nebula network.
type NetworkSetting struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	DefaultSubnet       string    `gorm:"size:64" json:"default_subnet"`
	HandshakePort       int       `json:"handshake_port"`
	LighthouseHosts     string    `gorm:"type:text" json:"lighthouse_hosts"`
	CertificateValidity int       `json:"certificate_validity"`
	Description         string    `gorm:"size:255" json:"description"`
	UpdatedAt           time.Time `json:"updated_at"`
	CreatedAt           time.Time `json:"created_at"`
}
