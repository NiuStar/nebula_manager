package models

import "time"

// NodeStatus stores the latest runtime metrics reported by a node agent.
type NodeStatus struct {
	ID          uint `gorm:"primaryKey"`
	NodeID      uint `gorm:"uniqueIndex"`
	CPUUsage    float64
	Load1       float64
	Load5       float64
	Load15      float64
	MemoryTotal uint64
	MemoryUsed  uint64
	SwapTotal   uint64
	SwapUsed    uint64
	DiskTotal   uint64
	DiskUsed    uint64
	NetRxBytes  uint64
	NetTxBytes  uint64
	Processes   int
	Uptime      uint64
	ReportedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
