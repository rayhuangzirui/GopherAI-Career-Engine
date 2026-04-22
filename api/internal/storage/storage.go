package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Storage interface {
	Name() string
	Put(ctx context.Context, key string, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}

func BuildTaskResultArtifactKey(taskID int64) string {
	return fmt.Sprintf("artifacts/task-%d/result.json", taskID)
}

func BuildUploadKey(userID int64, kind string, originalFileName string) string {
	safeName := sanitizeFileName(originalFileName)
	timestamp := time.Now().UTC().Format("20060102T150405Z")

	switch kind {
	case "resume":
		return fmt.Sprintf("uploads/resumes/user-%d/%s-%s", userID, timestamp, safeName)
	case "jd":
		return fmt.Sprintf("uploads/jds/user-%d/%s-%s", userID, timestamp, safeName)
	default:
		return fmt.Sprintf("uploads/misc/user-%d/%s-%s", userID, timestamp, safeName)
	}
}

func sanitizeFileName(fileName string) string {
	base := filepath.Clean(strings.TrimSpace(fileName))

	if base == "" || base == "." || base == "/" {
		return "file.txt"
	}

	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, " ", "_")

	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	base = re.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")

	if base == "" {
		return "file.txt"
	}

	return base
}
