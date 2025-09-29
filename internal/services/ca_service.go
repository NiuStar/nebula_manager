package services

import (
	"errors"

	"gorm.io/gorm"

	"nebula_manager/internal/models"
	"nebula_manager/internal/utils"
)

// CAService handles CRUD operations for certificate authorities.
type CAService struct {
	db *gorm.DB
}

// NewCAService creates a new CAService instance.
func NewCAService(db *gorm.DB) *CAService {
	return &CAService{db: db}
}

// CreateCARequest describes the input required to create a CA.
type CreateCARequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	ValidityDays int    `json:"validity_days"`
}

// GetCA returns the first CA entry if present.
func (s *CAService) GetCA() (*models.CA, error) {
	var ca models.CA
	if err := s.db.First(&ca).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ca, nil
}

// GenerateOrReplaceCA creates a CA and stores it, replacing the existing one if any.
func (s *CAService) GenerateOrReplaceCA(req CreateCARequest) (*models.CA, error) {
	cert, key, err := utils.GenerateCA(req.Name, req.ValidityDays)
	if err != nil {
		return nil, err
	}

	ca := &models.CA{
		Name:           req.Name,
		Description:    req.Description,
		CertificatePEM: cert,
		PrivateKeyPEM:  key,
	}

	tx := s.db.Begin()
	if err := tx.Error; err != nil {
		return nil, err
	}

	if err := tx.Where("1 = 1").Delete(&models.CA{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(ca).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return ca, nil
}
