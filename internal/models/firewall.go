package models

import (
	"time"

	"gorm.io/gorm"
)

type FirewallRule struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"unique;not null"`
	Protocol  string         `json:"protocol"`
	Port      string         `json:"port"`
	Action    string         `json:"action"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
