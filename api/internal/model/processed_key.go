package model

import "time"

type ProcessedKey struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	KeyValue  string    `gorm:"column:key_value;type:varchar(100);not null" json:"key_value"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}
