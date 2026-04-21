package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/ratelimit"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/taskcache"
	"gorm.io/gorm"
)

type TaskPublisher interface {
	PublishTask(ctx context.Context, taskID int64, taskType string, attempt int) error
}

type TaskHandler struct {
	taskRepo           *repository.TaskRepository
	publisher          TaskPublisher
	rateLimiter        *ratelimit.RateLimiter
	rateLimitPerMinute int
	taskCache          *taskcache.TaskCache
}

func NewTaskHandler(
	taskRepo *repository.TaskRepository,
	publisher TaskPublisher,
	rateLimiter *ratelimit.RateLimiter,
	rateLimitPerMinute int,
	taskCache *taskcache.TaskCache,
) *TaskHandler {
	return &TaskHandler{
		taskRepo:           taskRepo,
		publisher:          publisher,
		rateLimiter:        rateLimiter,
		rateLimitPerMinute: rateLimitPerMinute,
		taskCache:          taskCache,
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

type TaskResponse struct {
	OK           bool        `json:"ok"`
	ID           int64       `json:"id"`
	UserID       int64       `json:"user_id"`
	TaskType     string      `json:"task_type"`
	Status       string      `json:"status"`
	ErrorMessage *string     `json:"error_message"`
	RetryCount   int         `json:"retry_count"`
	StartedAt    interface{} `json:"started_at"`
	CompletedAt  interface{} `json:"completed_at"`
	CreatedAt    interface{} `json:"created_at"`
	UpdatedAt    interface{} `json:"updated_at"`
}

type TaskListItemResponse struct {
	ID           int64       `json:"id"`
	UserID       int64       `json:"user_id"`
	TaskType     string      `json:"task_type"`
	Status       string      `json:"status"`
	ErrorMessage *string     `json:"error_message"`
	RetryCount   int         `json:"retry_count"`
	StartedAt    interface{} `json:"started_at"`
	CompletedAt  interface{} `json:"completed_at"`
	CreatedAt    interface{} `json:"created_at"`
	UpdatedAt    interface{} `json:"updated_at"`
}

type TaskListResponse struct {
	OK    bool                   `json:"ok"`
	Tasks []TaskListItemResponse `json:"tasks"`
}

type TaskResultCompletedResponse struct {
	OK     bool        `json:"ok"`
	Status string      `json:"status"`
	Result interface{} `json:"result"`
}

type TaskResultFailedResponse struct {
	OK           bool    `json:"ok"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error_message"`
}

func (h *TaskHandler) CreateResumeAnalysisTask(c *gin.Context) {
	var req CreateResumeAnalysisTaskRequest
	if !bindJSON(c, &req) {
		return
	}

	if !h.enforceRateLimit(c, req.UserID, model.TaskTypeResumeAnalysis) {
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

	if !h.enforceRateLimit(c, req.UserID, model.TaskTypeResumeJDMatch) {
		return
	}

	input := model.ResumeJDMatchInput{
		ResumeText:         req.ResumeText,
		JobDescriptionText: req.JobDescriptionText,
	}

	h.createTask(c, req.UserID, model.TaskTypeResumeJDMatch, input)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}

	cacheKey := taskcache.BuildTaskKey(taskID)
	if h.taskCache != nil {
		var cached TaskResponse
		hit, err := h.taskCache.Get(c.Request.Context(), cacheKey, &cached)
		if err != nil {
			log.Printf("task_cache_error task_id=%d err=%v", taskID, err)
		} else if hit {
			log.Printf("task_cache_hit task_id=%d", taskID)
			c.JSON(http.StatusOK, cached)
			return
		} else {
			log.Printf("task_cache_miss task_id=%d", taskID)
		}
	}

	task, ok := h.getTaskOrRespond(c, taskID)
	if !ok {
		return
	}

	response := buildTaskResponse(task)

	// Only cache final states so polling won't be dulled by cached queued/processing/retrying states.
	if h.taskCache != nil {
		if task.Status == model.TaskStatusCompleted || task.Status == model.TaskStatusPermanentlyFailed {
			if err := h.taskCache.SetTask(c.Request.Context(), taskID, response); err != nil {
				log.Printf("task_cache_set_error task_id=%d err=%v", taskID, err)
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *TaskHandler) GetTaskResult(c *gin.Context) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}

	if h.taskCache != nil {
		var cached map[string]interface{}
		hit, err := h.taskCache.Get(c.Request.Context(), taskcache.BuildTaskResultKey(taskID), &cached)
		if err != nil {
			log.Printf("task_result_cache_error task_id=%d err=%v", taskID, err)
		} else if hit {
			log.Printf("task_result_cache_hit task_id=%d", taskID)
			c.JSON(http.StatusOK, cached)
			return
		} else {
			log.Printf("task_result_cache_miss task_id=%d", taskID)
		}
	}

	task, ok := h.getTaskOrRespond(c, taskID)
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

		response := TaskResultCompletedResponse{
			OK:     true,
			Status: task.Status,
			Result: result,
		}

		if h.taskCache != nil {
			if err := h.taskCache.SetTaskResult(c.Request.Context(), taskID, response); err != nil {
				log.Printf("task_result_cache_set_error task_id=%d err=%v", taskID, err)
			}
		}

		c.JSON(http.StatusOK, response)
		return

	case model.TaskStatusPermanentlyFailed:
		response := TaskResultFailedResponse{
			OK:           false,
			Status:       task.Status,
			ErrorMessage: task.ErrorMessage,
		}

		if h.taskCache != nil {
			if err := h.taskCache.SetTaskResult(c.Request.Context(), taskID, response); err != nil {
				log.Printf("task_result_cache_set_error task_id=%d err=%v", taskID, err)
			}
		}

		c.JSON(http.StatusOK, response)
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

func (h *TaskHandler) ListTasks(c *gin.Context) {
	userID, ok := parseUserIDFromQuery(c)
	if !ok {
		return
	}

	limit, ok := parseLimitFromQuery(c)
	if !ok {
		return
	}

	cacheKey := taskcache.BuildTaskListKey(userID, limit)
	if h.taskCache != nil {
		var cached TaskListResponse
		hit, err := h.taskCache.Get(c.Request.Context(), cacheKey, &cached)
		if err != nil {
			log.Printf("task_list_cache_error user_id=%d limit=%d err=%v", userID, limit, err)
		} else if hit {
			log.Printf("task_list_cache_hit user_id=%d limit=%d", userID, limit)
			c.JSON(http.StatusOK, cached)
			return
		} else {
			log.Printf("task_list_cache_miss user_id=%d limit=%d", userID, limit)
		}
	}

	tasks, err := h.taskRepo.ListTasks(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	response := TaskListResponse{
		OK:    true,
		Tasks: make([]TaskListItemResponse, 0, len(tasks)),
	}

	for _, task := range tasks {
		response.Tasks = append(response.Tasks, buildTaskListItemResponse(task))
	}

	if h.taskCache != nil {
		if err := h.taskCache.SetTaskList(c.Request.Context(), userID, limit, response); err != nil {
			log.Printf("task_list_cache_set_error user_id=%d limit=%d err=%v", userID, limit, err)
		}
	}

	c.JSON(http.StatusOK, response)
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

	if err := h.publisher.PublishTask(c.Request.Context(), task.ID, taskType, 0); err != nil {
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

	if h.taskCache != nil {
		if err := h.taskCache.DeleteTaskListsForUser(c.Request.Context(), userID); err != nil {
			log.Printf("task_list_cache_delete_error user_id=%d err=%v", userID, err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"ok":      true,
		"task_id": task.ID,
		"status":  model.TaskStatusQueued,
	})
}

func (h *TaskHandler) enforceRateLimit(c *gin.Context, userID int64, taskType string) bool {
	if h.rateLimiter == nil || h.rateLimitPerMinute <= 0 {
		return true
	}

	key := "rate_limit:user:" + strconv.FormatInt(userID, 10) + ":task_type:" + taskType

	allowed, current, resetIn, err := h.rateLimiter.Allow(
		c.Request.Context(),
		key,
		h.rateLimitPerMinute,
		time.Minute,
	)
	if err != nil {
		log.Printf("rate_limit_check error user_id=%d task_type=%s err=%v", userID, taskType, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "rate limiter unavailable",
		})
		return false
	}

	log.Printf(
		"rate_limit_check user_id=%d task_type=%s allowed=%t count=%d limit=%d reset_in=%s",
		userID,
		taskType,
		allowed,
		current,
		h.rateLimitPerMinute,
		resetIn.String(),
	)

	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"ok":    false,
			"error": "rate limit exceeded",
		})
		return false
	}

	return true
}

func (h *TaskHandler) getTaskOrRespond(c *gin.Context, taskID int64) (*model.Task, bool) {
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

func parseUserIDFromQuery(c *gin.Context) (int64, bool) {
	rawUserID := c.Query("user_id")
	if rawUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "missing user ID",
		})
		return 0, false
	}

	userID, err := strconv.ParseInt(rawUserID, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "invalid user ID",
		})
		return 0, false
	}

	return userID, true
}

func parseLimitFromQuery(c *gin.Context) (int, bool) {
	rawLimit := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "invalid limit",
		})
		return 0, false
	}

	return limit, true
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

func buildTaskResponse(task *model.Task) TaskResponse {
	return TaskResponse{
		OK:           true,
		ID:           task.ID,
		UserID:       task.UserID,
		TaskType:     task.TaskType,
		Status:       task.Status,
		ErrorMessage: task.ErrorMessage,
		RetryCount:   task.RetryCount,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}

func buildTaskListItemResponse(task model.Task) TaskListItemResponse {
	return TaskListItemResponse{
		ID:           task.ID,
		UserID:       task.UserID,
		TaskType:     task.TaskType,
		Status:       task.Status,
		ErrorMessage: task.ErrorMessage,
		RetryCount:   task.RetryCount,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}