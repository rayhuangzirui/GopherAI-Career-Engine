package handler

import (
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/storage"
)

const maxUploadSizeBytes = 1<<20 // 1MB

type UploadHandler struct {
	uploadRepo *repository.UploadRepository
	fileStore storage.Storage
}

func NewUploadHandler(uploadRepo *repository.UploadRepository, fileStore storage.Storage) *UploadHandler {
	return &UploadHandler{uploadRepo: uploadRepo, fileStore: fileStore}
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	rawUserID := c.PostForm("user_id")
	if rawUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "missing user ID",
		})
		return
	}
	userID, err := strconv.ParseInt(rawUserID, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "invalid user_id",
		})
		return
	}

	kind := c.PostForm("kind")
	if kind != model.UploadKindResume && kind != model.UploadKindJD {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "kind must be resume or jd",
		})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "missing file",
		})
		return
	}

	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "empty file",
		})
		return
	}

	if fileHeader.Size > maxUploadSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "file size exceeds the 1 MB limit",
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".txt" {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "only .txt files are supported in this version",
		})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "failed to open uploaded file",
		})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "failed to read uploaded file",
		})
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	fileKey := storage.BuildUploadKey(userID, kind, fileHeader.Filename)
	if err := h.fileStore.Put(c.Request.Context(), fileKey, contentType, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "failed to store uploaded file",
		})
		return
	}

	upload := &model.Upload{
		UserID: userID,
		FileKind: kind,
		StorageKey: fileKey,
		OriginalFilename: fileHeader.Filename,
		ContentType: contentType,
		SizeBytes: int64(len(data)),
	}

	if err := h.uploadRepo.CreateUpload(c.Request.Context(), upload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"upload_id": upload.ID,
		"user_id": upload.UserID,
		"file_kind": upload.FileKind,
		"file_key": upload.StorageKey,
		"original_filename": upload.OriginalFilename,
		"content_type": upload.ContentType,
		"size_bytes": upload.SizeBytes,
		"storage": h.fileStore.Name(),
	})
}
