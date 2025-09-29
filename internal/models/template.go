package models

import "time"

// ConfigTemplate stores reusable Nebula configuration text snippets.
type ConfigTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null;unique" json:"name"`
	Content   string    `gorm:"type:longtext" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
