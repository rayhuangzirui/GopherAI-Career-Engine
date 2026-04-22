package repository

import (
	"context"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"gorm.io/gorm"
)

type UploadRepository struct {
	db *gorm.DB
}

func NewUploadRepository(db *gorm.DB) *UploadRepository {
	return &UploadRepository{db: db}
}

func (r *UploadRepository) CreateUpload(ctx context.Context, upload *model.Upload) error {
	return r.db.WithContext(ctx).Create(upload).Error
}

func (r *UploadRepository) GetUploadByStorageKey(ctx context.Context, storageKey string) (*model.Upload, error) {
	var upload model.Upload

	if err := r.db.WithContext(ctx).
		Where("storage_key = ?", storageKey).
		 First(&upload).Error; err != nil {
			return nil, err
		 }

	return &upload, nil
}
