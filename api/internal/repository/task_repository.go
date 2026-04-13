package repository

import (
	"context"
	"errors"
	"time"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"

	"gorm.io/gorm"
)

var ErrInvalidStateTransition = errors.New("invalid state transition")

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) CreateTask(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *TaskRepository) GetTask(ctx context.Context, id int64) (*model.Task, error) {
	var task model.Task
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

func (r *TaskRepository) ListTasks(ctx context.Context, userID int64, limit int) ([]model.Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var tasks []model.Task
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) MarkQueued(ctx context.Context, id int64) error {
	//return r.db.WithContext(ctx).
	//	Model(&model.Task{}).
	//	Where("id = ?", id).
	//	Update("status", model.TaskStatusQueued).Error
	tx := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND status = ?", id, model.TaskStatusPending).
		Update("status", model.TaskStatusQueued)
	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return ErrInvalidStateTransition
	}

	return nil
}

func (r *TaskRepository) MarkProcessing(ctx context.Context, id int64) error {
	now := time.Now()

	tx := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND status IN ?", id, []string{
			model.TaskStatusPending,
			model.TaskStatusQueued,
			model.TaskStatusRetrying,
		}).
		Updates(map[string]interface{}{
			"status":     model.TaskStatusProcessing,
			"started_at": &now,
		})
	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return ErrInvalidStateTransition
	}

	return nil
}

func (r *TaskRepository) MarkRetrying(ctx context.Context, id int64, errorMessage string) error {
	tx := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND status = ?", id, model.TaskStatusProcessing).
		Updates(map[string]interface{}{
			"status":        model.TaskStatusRetrying,
			"error_message": errorMessage,
			"retry_count":   gorm.Expr("retry_count + 1"),
		})

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return ErrInvalidStateTransition
	}
	return nil
}

//func (r *TaskRepository) RequeueFromRetrying(ctx context.Context, id int64) error {
//	tx := r.db.WithContext(ctx).
//		Model(&model.Task{}).
//		Where("id = ? AND status = ?", id, model.TaskStatusRetrying).
//		Updates(map[string]interface{}{
//			"status": model.TaskStatusQueued,
//		})
//	if tx.Error != nil {
//		return tx.Error
//	}
//
//	if tx.RowsAffected == 0 {
//		return ErrInvalidStateTransition
//	}
//	return nil
//}

func (r *TaskRepository) MarkPermanentlyFailed(ctx context.Context, id int64, errorMessage string) error {
	now := time.Now()

	tx := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND status = ?", id, model.TaskStatusProcessing).
		Updates(map[string]interface{}{
			"status":        model.TaskStatusPermanentlyFailed,
			"error_message": errorMessage,
			"completed_at":  &now,
		})

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return ErrInvalidStateTransition
	}

	return nil
}

func (r *TaskRepository) MarkCompleted(ctx context.Context, id int64, resultPayload string) error {
	now := time.Now()
	tx := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND status = ?", id, model.TaskStatusProcessing).
		Updates(map[string]interface{}{
			"status":         model.TaskStatusCompleted,
			"result_payload": resultPayload,
			"completed_at":   &now,
		})
	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return ErrInvalidStateTransition
	}

	return nil
}

func (r *TaskRepository) MarkFailed(ctx context.Context, id int64, errorMessage string) error {
	now := time.Now()
	tx := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND status = ?", id, model.TaskStatusProcessing).
		Updates(map[string]interface{}{
			"status":        model.TaskStatusFailed,
			"error_message": errorMessage,
			"completed_at":  &now,
			"retry_count":   gorm.Expr("retry_count + 1"),
		})
	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return ErrInvalidStateTransition
	}

	return nil
}
