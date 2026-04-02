package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"gorm.io/gorm"
)

type TaskPublisher interface {
	PublishTask(ctx context.Context, taskID int64) error
}

type TaskHandler struct {
	taskRepo  *repository.TaskRepository
	publisher TaskPublisher
}

func NewTaskHandler(taskRepo *repository.TaskRepository, publisher TaskPublisher) *TaskHandler {
	return &TaskHandler{
		taskRepo:  taskRepo,
		publisher: publisher,
	}
}

type CreateResumeAnalysisTaskRequest struct {
	UserID     int64  `json:"userId" binding:"required"`
	ResumeText string `json:"resumeText" binding:"required"`
}

func (h *TaskHandler) CreateResumeAnalysisTask(c *gin.Context) {
	var req CreateResumeAnalysisTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "invalid request body",
		})
		return
	}

	input := model.ResumeAnalysisInput{
		ResumeText: req.ResumeText,
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "failed to marshal task input",
		})
		return
	}

	task := &model.Task{
		UserID:       req.UserID,
		TaskType:     model.TaskTypeResumeAnalysis,
		Status:       model.TaskStatusPending,
		InputPayload: string(inputBytes),
	}

	if err := h.taskRepo.CreateTask(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	if err := h.publisher.PublishTask(c.Request.Context(), task.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	if err := h.taskRepo.MarkQueued(c.Request.Context(), task.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"ok":     true,
		"taskId": task.ID,
		"status": model.TaskStatusQueued,
	})
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}

	task, err := h.taskRepo.GetTask(c.Request.Context(), taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"ok":    false,
				"error": "task not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"id":       task.ID,
		"userId":   task.UserID,
		"taskType": task.TaskType,
		"status":   task.Status,
		//"inputPayload": task.InputPayload,
		//"resultPayload": task.ResultPayload,
		//"errorMessage": task.ErrorMessage,
		"retryCount":  task.RetryCount,
		"startedAt":   task.StartedAt,
		"completedAt": task.CompletedAt,
		"createdAt":   task.CreatedAt,
		"updatedAt":   task.UpdatedAt,
	})
}

func (h *TaskHandler) GetTaskResult(c *gin.Context) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}

	task, err := h.taskRepo.GetTask(c.Request.Context(), taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"ok":    false,
				"error": "task not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	switch task.Status {
	case model.TaskStatusCompleted:
		var result interface{}
		if task.ResultPayload != nil && *task.ResultPayload != "" {
			if err := json.Unmarshal([]byte(*task.ResultPayload), &result); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"ok":    false,
					"error": "failed to unmarshal result payload",
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"status": task.Status,
			"result": result,
		})
		return
	case model.TaskStatusFailed:
		c.JSON(http.StatusOK, gin.H{
			"ok":           false,
			"status":       task.Status,
			"errorMessage": task.ErrorMessage,
		})
		return
	default:
		c.JSON(http.StatusOK, gin.H{
			"ok":      true,
			"status":  task.Status,
			"message": "task is not completed yet",
		})
		return
	}
}

func parseTaskID(c *gin.Context) (int64, bool) {
	rawID := c.Param("id")
	taskID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || taskID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "invalid task ID",
		})
		return 0, false
	}

	return taskID, true
}
