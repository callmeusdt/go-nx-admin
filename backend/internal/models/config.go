package models

import (
	"time"
)

type SystemConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"uniqueIndex;size:128;not null" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Group       string    `gorm:"size:64;index" json:"group"`
	Type        string    `gorm:"size:16;default:string" json:"type"`
	Label       string    `gorm:"size:128" json:"label"`
	Description string    `gorm:"size:255" json:"description"`
	Sort        int       `gorm:"default:0" json:"sort"`
	UpdatedAt   time.Time `json:"updated_at"`
}
