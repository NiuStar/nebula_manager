package models

import "time"

// CA represents a certificate authority used to sign Nebula node certificates.
type CA struct {
	ID             uint   `gorm:"primaryKey"`
	Name           string `gorm:"size:100;not null;unique"`
	Description    string `gorm:"size:255"`
	CertificatePEM string `gorm:"type:longtext"`
	PrivateKeyPEM  string `gorm:"type:longtext"`
	CreatedAt      time.Time
}

// TableName customises the table name for CA to avoid awkward inflection.
func (CA) TableName() string {
	return "cas"
}
