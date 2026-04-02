package repository

import (
	"context"
	"errors"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"gorm.io/gorm"
)

var ErrProcessedKeyAlreadyExists = errors.New("processed key already exists")

type ProcessedKeyRepository struct {
	db *gorm.DB
}

func NewProcessedKeyRepository(db *gorm.DB) *ProcessedKeyRepository {
	return &ProcessedKeyRepository{db: db}
}

func (r *ProcessedKeyRepository) Exists(ctx context.Context, key string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.ProcessedKey{}).
		Where("key_value = ?", key).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *ProcessedKeyRepository) Create(ctx context.Context, key string) error {
	record := &model.ProcessedKey{
		KeyValue: key,
	}

	err := r.db.WithContext(ctx).Create(record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProcessedKeyAlreadyExists
		}
		return err
	}

	return nil
}
