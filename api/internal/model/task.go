package model

import "time"

const (
	TaskTypeResumeAnalysis = "resume_analysis"
	TaskTypeResumeJDMatch  = "resume_jd_match"
)

const (
	TaskStatusPending           = "pending"
	TaskStatusQueued            = "queued"
	TaskStatusProcessing        = "processing"
	TaskStatusCompleted         = "completed"
	TaskStatusFailed            = "failed"
	TaskStatusRetrying          = "retrying"
	TaskStatusPermanentlyFailed = "permanently_failed"
	//TaskStatusCancelled         = "cancelled"
	//TaskStatusExpired = "expired"
	//TaskStatusRetried = "retried"
	//TaskStatusTimeout = "timeout"
)

type Task struct {
	ID            int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64      `gorm:"index;not null" json:"user_id"`
	TaskType      string     `gorm:"type:varchar(50);not null;index" json:"task_type"`
	Status        string     `gorm:"type:varchar(20);not null;index" json:"status"`
	InputPayload  string     `gorm:"type:json;not null" json:"input_payload"`
	ResultPayload *string    `gorm:"type:json" json:"result_payload"`
	ErrorMessage  *string    `gorm:"type:text" json:"error_message"`
	RetryCount    int        `gorm:"default:0;not null" json:"retry_count"`
	StartedAt     *time.Time `gorm:"type:datetime" json:"started_at"`   // nullable, only set when task is started
	CompletedAt   *time.Time `gorm:"type:datetime" json:"completed_at"` // nullable, only set when task is completed
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
