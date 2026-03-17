package model

import (
	"time"
)

type Message struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID string    `gorm:"type:varchar(36);index;not null" json:"session_id"`
	UserID    int64     `gorm:"index;not null" json:"user_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsUser    bool      `gorm:"type:boolean;not null" json:"is_user"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Session Session `gorm:"foreignKey:SessionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	User    User    `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type History struct {
	IsUser  bool   `json:"is_user"`
	Content string `json:"content"`
}
