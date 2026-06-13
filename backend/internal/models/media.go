package models

import (
	"time"
)

type Media struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OriginalName string    `gorm:"size:255;not null" json:"original_name"`
	FileName     string    `gorm:"size:255;not null" json:"file_name"`
	Path         string    `gorm:"size:512;not null" json:"path"`
	URL          string    `gorm:"size:512;not null" json:"url"`
	Ext          string    `gorm:"size:16" json:"ext"`
	MimeType     string    `gorm:"size:128" json:"mime_type"`
	Size         int64     `json:"size"`
	Type         string    `gorm:"size:16;index" json:"type"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}
