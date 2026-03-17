package model

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string         `gorm:"type:varchar(50);not null" json:"name"`
	Email        string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	UserName     string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"user_name"` // Unique username for login
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`                    // Store hashed password, never return it in JSON
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`                       // Automatically set to current time when creating a new record
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Sessions []Session `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"` // One-to-many relationship with sessions
	// CASCADE ensures that when a user is deleted, all their sessions are also deleted.
}
