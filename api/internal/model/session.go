package model

import (
	"gorm.io/gorm"
	"time"
)

// Session represents a chat session between a user and the system.
// Each session has a unique ID, is associated with a user, and contains metadata about the session.
// The mapping to the "sessions" table in the database is defined by GORM tags.
type Session struct {
	ID        string         `gorm:"primaryKey;type:varchar(36)" json:"id"`   // Unique session ID, typically a UUID
	UserID    int64          `gorm:"index;not null" json:"user_id"`           // Foreign key referencing the user who owns this session, indexed for efficient queries
	Title     string         `gorm:"type:varchar(100);not null" json:"title"` // The Title of the session, can be set by the user
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`        // Timestamp when the session was created
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`        // Timestamp when the session was last updated
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                          // Soft delete field, indexed for efficient queries

	// GORM will automatically create a foreign key constraint for UserID referencing the users table.
	// Belongs to one user, and when the user is updated or deleted, the session will be updated or deleted accordingly (CASCADE).
	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	// One session can have many messages, and when the session is deleted, all associated messages will also be deleted (CASCADE).
	Messages []Message `gorm:"foreignKey:SessionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type SessionSummary struct {
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
}
