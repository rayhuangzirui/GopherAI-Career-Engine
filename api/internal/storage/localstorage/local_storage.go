package localstorage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/storage"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("local storage base directory is empty")
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create local storage base directory: %w", err)
	}

	return &LocalStorage{baseDir: baseDir}, nil
}

func (s *LocalStorage) Name() string {
	return "local"
}

func (s *LocalStorage) Put(ctx context.Context, key string, contentType string, data []byte) error {
	_ = ctx
	_ = contentType

	cleanKey := filepath.Clean(strings.TrimPrefix(key, "/"))
	if cleanKey == "." || cleanKey == "" {
		return fmt.Errorf("invalid storage key: %q", key)
	}

	fullPath := filepath.Join(s.baseDir, cleanKey)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("create artifact parent directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return fmt.Errorf("write artifact file: %w", err)
	}

	return nil
}

var _ storage.Storage = (*LocalStorage)(nil)
