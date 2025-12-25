package models

import (
	"time"
)

type Setting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Type      string    `gorm:"size:20;default:'string'" json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type InstalledPackage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PackageID   string    `gorm:"size:50;not null;index" json:"package_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Version     string    `gorm:"size:50" json:"version"`
	Category    string    `gorm:"size:50" json:"category"`
	InstallPath string    `gorm:"size:500" json:"install_path"`
	InstalledAt time.Time `json:"installed_at"`
	Status      string    `gorm:"size:20;default:'installed'" json:"status"`
}

type ActivityLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Action    string    `gorm:"size:100;not null" json:"action"`
	Details   string    `gorm:"type:text" json:"details"`
	IP        string    `gorm:"size:45" json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}
