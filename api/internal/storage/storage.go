package storage

import (
	"context"
	"fmt"
)

type Storage interface {
	Name() string
	Put(ctx context.Context, key string, contentType string, data []byte) error
}

func BuildTaskResultArtifactKey(taskID int64) string {
	return fmt.Sprintf("artifacts/task-%d/result.json", taskID)
}
