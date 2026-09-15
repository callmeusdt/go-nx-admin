package models

import "time"

// BrowserSession stores only hashes of random cookie credentials. Purpose
// separates pre-MFA login challenges from fully authorized sessions.
type BrowserSession struct {
	ID                uint      `gorm:"primaryKey"`
	UserID            uint      `gorm:"index;not null"`
	TokenHash         string    `gorm:"uniqueIndex;size:64;not null"`
	CSRFHash          string    `gorm:"size:64;not null"`
	Purpose           string    `gorm:"size:16;not null"`
	ExpiresAt         time.Time `gorm:"index;not null"`
	ReauthenticatedAt *time.Time
	Attempts          int `gorm:"not null;default:0"`
	CreatedAt         time.Time
	IP                string `gorm:"size:45"`
	UserAgent         string `gorm:"size:255"`
}

func (BrowserSession) TableName() string { return "admin_browser_sessions" }
