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
	PublishTask(ctx context.Context, taskID int64, taskType string) error
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
	UserID     int64  `json:"user_id" binding:"required"`
	ResumeText string `json:"resume_text" binding:"required"`
}

type CreateResumeJDMatchTaskRequest struct {
	UserID             int64  `json:"user_id" binding:"required"`
	ResumeText         string `json:"resume_text" binding:"required"`
	JobDescriptionText string `json:"job_description_text" binding:"required"`
}

func (h *TaskHandler) CreateResumeAnalysisTask(c *gin.Context) {
	var req CreateResumeAnalysisTaskRequest
	if !bindJSON(c, &req) {
		return
	}

	input := model.ResumeAnalysisInput{
		ResumeText: req.ResumeText,
	}

	h.createTask(c, req.UserID, model.TaskTypeResumeAnalysis, input)
}

func (h *TaskHandler) CreateResumeJDMatchTask(c *gin.Context) {
	var req CreateResumeJDMatchTaskRequest
	if !bindJSON(c, &req) {
		return
	}

	input := model.ResumeJDMatchInput{
		ResumeText:         req.ResumeText,
		JobDescriptionText: req.JobDescriptionText,
	}

	h.createTask(c, req.UserID, model.TaskTypeResumeJDMatch, input)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	task, ok := h.getTaskOrRespond(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"id":        task.ID,
		"user_id":   task.UserID,
		"task_type": task.TaskType,
		"status":    task.Status,
		//"input_payload": task.InputPayload,
		//"result_payload": task.ResultPayload,
		"error_message": task.ErrorMessage,
		"retry_count":   task.RetryCount,
		"started_at":    task.StartedAt,
		"completed_at":  task.CompletedAt,
		"created_at":    task.CreatedAt,
		"updated_at":    task.UpdatedAt,
	})
}

func (h *TaskHandler) GetTaskResult(c *gin.Context) {
	task, ok := h.getTaskOrRespond(c)
	if !ok {
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
			"ok":            false,
			"status":        task.Status,
			"error_message": task.ErrorMessage,
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

func (h *TaskHandler) createTask(c *gin.Context, userID int64, taskType string, input any) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "failed to marshal task input",
		})
		return
	}

	task := &model.Task{
		UserID:       userID,
		TaskType:     taskType,
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

	if err := h.publisher.PublishTask(c.Request.Context(), task.ID, taskType); err != nil {
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
		"ok":      true,
		"task_id": task.ID,
		"status":  model.TaskStatusQueued,
	})
}

func (h *TaskHandler) getTaskOrRespond(c *gin.Context) (*model.Task, bool) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return nil, false
	}

	task, err := h.taskRepo.GetTask(c.Request.Context(), taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"ok":    false,
				"error": "task not found",
			})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return nil, false
	}

	return task, true
}

func bindJSON[T any](c *gin.Context, req *T) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return false
	}
	return true
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
