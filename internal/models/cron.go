package models

import (
	"time"

	"gorm.io/gorm"
)

type CronJob struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	Name       string         `json:"name" gorm:"not null"`
	Schedule   string         `json:"schedule" gorm:"not null"` // Cron syntax: * * * * *
	Command    string         `json:"command" gorm:"not null"`
	Enabled    bool           `json:"enabled" gorm:"default:true"`
	LastRun    *time.Time     `json:"last_run"`
	LastStatus string         `json:"last_status"` // success, error
	LastResult string         `json:"last_result"` // Output or error message
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
