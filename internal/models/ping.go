package models

import "time"

// NodePing stores the measured latency between two managed nodes.
type NodePing struct {
	ID         uint      `gorm:"primaryKey"`
	NodeID     uint      `gorm:"not null;index:idx_node_peer_created"`
	PeerNodeID uint      `gorm:"not null;index:idx_node_peer_created"`
	LatencyMs  float64   `gorm:"type:double"`
	Success    bool      `gorm:"not null"`
	CreatedAt  time.Time `gorm:"index:idx_node_peer_created"`
}
