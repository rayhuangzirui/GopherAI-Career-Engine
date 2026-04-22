package model

import "time"

const (
	UploadKindResume = "resume"
	UploadKindJD = "jd"
)

type Upload struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID int64 `gorm:"index;not null" json:"user_id"`
	FileKind string `gorm:"type:varchar(20);not null;index" json:"file_kind"`
	StorageKey string `gorm:"type:varchar(255);not null;uniqueIndex" json:"storage_key"`
	OriginalFilename string `gorm:"type:varchar(255);not null" json:"original_filename"`
	ContentType string `gorm:"type:varchar(100);not null" json:"content_type"`
	SizeBytes int64 `gorm:"not null" json:"size_bytes"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
