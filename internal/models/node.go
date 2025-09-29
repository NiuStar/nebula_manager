package models

import "time"

// NodeRole enumerates supported Nebula node types.
const (
	NodeRoleLighthouse = "lighthouse"
	NodeRoleStandard   = "standard"
)

// Node records metadata and generated artifacts for a Nebula node.
type Node struct {
	ID                uint   `gorm:"primaryKey"`
	Name              string `gorm:"size:100;not null;unique"`
	Role              string `gorm:"size:20;not null"`
	SubnetIP          string `gorm:"size:64"`
	SubnetCIDR        string `gorm:"column:subnet_c_id_r;size:64"`
	SubnetHost        string `gorm:"size:64"`
	PublicIP          string `gorm:"size:64"`
	Port              int
	Tags              string `gorm:"size:255"`
	DownloadProxyMode string `gorm:"size:16"`
	CertificatePEM    string `gorm:"type:longtext"`
	PrivateKeyPEM     string `gorm:"type:longtext"`
	ConfigContent     string `gorm:"type:longtext"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
