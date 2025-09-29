package services

import (
	"errors"

	"gorm.io/gorm"

	"nebula_manager/internal/models"
)

// SettingsService manages network-wide configuration such as subnets and ports.
type SettingsService struct {
	db *gorm.DB
}

// NewSettingsService constructs a SettingsService.
func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{db: db}
}

// UpdateNetworkSettingsRequest carries settings update payload.
type UpdateNetworkSettingsRequest struct {
	DefaultSubnet       string `json:"default_subnet"`
	HandshakePort       int    `json:"handshake_port"`
	LighthouseHosts     string `json:"lighthouse_hosts"`
	CertificateValidity int    `json:"certificate_validity"`
	Description         string `json:"description"`
}

// Get retrieves the singleton network settings row, creating one if absent.
func (s *SettingsService) Get() (*models.NetworkSetting, error) {
	var setting models.NetworkSetting
	if err := s.db.First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting = models.NetworkSetting{
				DefaultSubnet:       "10.10.0.0/24",
				HandshakePort:       4242,
				CertificateValidity: 365,
			}
			if err := s.db.Create(&setting).Error; err != nil {
				return nil, err
			}
			return &setting, nil
		}
		return nil, err
	}
	return &setting, nil
}

// Update applies new values to the singleton network setting.
func (s *SettingsService) Update(req UpdateNetworkSettingsRequest) (*models.NetworkSetting, error) {
	setting, err := s.Get()
	if err != nil {
		return nil, err
	}

	if req.DefaultSubnet != "" {
		setting.DefaultSubnet = req.DefaultSubnet
	}
	if req.HandshakePort != 0 {
		setting.HandshakePort = req.HandshakePort
	}
	if req.CertificateValidity != 0 {
		setting.CertificateValidity = req.CertificateValidity
	}
	if req.LighthouseHosts != "" {
		setting.LighthouseHosts = req.LighthouseHosts
	}
	if req.Description != "" {
		setting.Description = req.Description
	}

	if err := s.db.Save(setting).Error; err != nil {
		return nil, err
	}
	return setting, nil
}
