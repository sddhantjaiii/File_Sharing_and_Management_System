package models

import (
	"time"

	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	UserID       uint      `gorm:"not null" json:"user_id"`
	User         *User     `gorm:"foreignKey:UserID" json:"-"`
	Filename     string    `gorm:"not null" json:"filename"`
	OriginalName string    `gorm:"not null" json:"original_name"`
	Size         int64     `gorm:"not null" json:"size"`
	MimeType     string    `gorm:"not null" json:"mime_type"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	IsPublic     bool      `gorm:"default:false" json:"is_public"`
	ShareToken   string    `gorm:"uniqueIndex" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
