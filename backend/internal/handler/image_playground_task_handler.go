package handler

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const defaultImagePlaygroundTaskPollAfterMS = 1000

type imagePlaygroundTaskManager interface {
	CreateTask(ctx context.Context, req service.ImagePlaygroundTaskCreateRequest) (*service.ImagePlaygroundTask, error)
	GetTask(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error)
	CancelTask(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error)
	ListRecentTasks(ctx context.Context, userID int64, limit int) ([]service.ImagePlaygroundTask, error)
}

type imagePlaygroundAPIKeyResolver interface {
	GetByID(ctx context.Context, id int64) (*service.APIKey, error)
	GetByKey(ctx context.Context, key string) (*service.APIKey, error)
}

type ImagePlaygroundTaskHandler struct {
	taskService  imagePlaygroundTaskManager
	apiKeyLookup imagePlaygroundAPIKeyResolver
}

func NewImagePlaygroundTaskHandler(taskService *service.ImagePlaygroundTaskService, apiKeyLookup *service.APIKeyService) *ImagePlaygroundTaskHandler {
	return &ImagePlaygroundTaskHandler{
		taskService:  taskService,
		apiKeyLookup: apiKeyLookup,
	}
}

func newImagePlaygroundTaskHandlerForTest(taskService imagePlaygroundTaskManager, apiKeyLookup imagePlaygroundAPIKeyResolver) *ImagePlaygroundTaskHandler {
	return &ImagePlaygroundTaskHandler{
		taskService:  taskService,
		apiKeyLookup: apiKeyLookup,
	}
}

type imagePlaygroundCreateTaskRequest struct {
	APIKeyID    int64           `json:"api_key_id"`
	APIKey      string          `json:"api_key"`
	Endpoint    string          `json:"endpoint" binding:"required"`
	RequestJSON json.RawMessage `json:"request" binding:"required"`
}

type imagePlaygroundTaskResponse struct {
	TaskID       int64           `json:"task_id"`
	Status       string          `json:"status"`
	Endpoint     string          `json:"endpoint,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
	ExpiresAt    time.Time       `json:"expires_at"`
	CanceledAt   *time.Time      `json:"canceled_at,omitempty"`
	UpdatedAt    time.Time       `json:"updated_at"`
	PollAfterMS  int             `json:"poll_after_ms,omitempty"`
}

func (h *ImagePlaygroundTaskHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.taskService == nil || h.apiKeyLookup == nil {
		response.InternalError(c, "image playground task service unavailable")
		return
	}

	var req imagePlaygroundCreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	apiKey, err := h.resolveAPIKey(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if apiKey.UserID != subject.UserID {
		response.Forbidden(c, "Not authorized to use this API key")
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), service.ImagePlaygroundTaskCreateRequest{
		UserID:      subject.UserID,
		APIKeyID:    apiKey.ID,
		Endpoint:    req.Endpoint,
		RequestJSON: append([]byte(nil), req.RequestJSON...),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Accepted(c, imagePlaygroundTaskResponseFromService(task, false))
}

func (h *ImagePlaygroundTaskHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.taskService == nil {
		response.InternalError(c, "image playground task service unavailable")
		return
	}
	taskID, ok := parseImagePlaygroundTaskID(c)
	if !ok {
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), subject.UserID, taskID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, imagePlaygroundTaskResponseFromService(task, true))
}

func (h *ImagePlaygroundTaskHandler) Cancel(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.taskService == nil {
		response.InternalError(c, "image playground task service unavailable")
		return
	}
	taskID, ok := parseImagePlaygroundTaskID(c)
	if !ok {
		return
	}

	task, err := h.taskService.CancelTask(c.Request.Context(), subject.UserID, taskID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, imagePlaygroundTaskResponseFromService(task, false))
}

func (h *ImagePlaygroundTaskHandler) Recent(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.taskService == nil {
		response.InternalError(c, "image playground task service unavailable")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			response.BadRequest(c, "Invalid limit")
			return
		}
		limit = parsed
	}
	tasks, err := h.taskService.ListRecentTasks(c.Request.Context(), subject.UserID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]imagePlaygroundTaskResponse, 0, len(tasks))
	for i := range tasks {
		out = append(out, imagePlaygroundTaskResponseFromService(&tasks[i], true))
	}
	response.Success(c, out)
}

func (h *ImagePlaygroundTaskHandler) resolveAPIKey(ctx context.Context, req imagePlaygroundCreateTaskRequest) (*service.APIKey, error) {
	if strings.TrimSpace(req.APIKey) != "" {
		return h.apiKeyLookup.GetByKey(ctx, strings.TrimSpace(req.APIKey))
	}
	if req.APIKeyID > 0 {
		return h.apiKeyLookup.GetByID(ctx, req.APIKeyID)
	}
	return nil, service.ErrAPIKeyNotFound
}

func parseImagePlaygroundTaskID(c *gin.Context) (int64, bool) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		response.BadRequest(c, "Invalid task ID")
		return 0, false
	}
	return taskID, true
}

func imagePlaygroundTaskResponseFromService(task *service.ImagePlaygroundTask, includeResult bool) imagePlaygroundTaskResponse {
	if task == nil {
		return imagePlaygroundTaskResponse{}
	}
	out := imagePlaygroundTaskResponse{
		TaskID:      task.ID,
		Status:      task.Status,
		Endpoint:    task.Endpoint,
		CreatedAt:   task.CreatedAt,
		StartedAt:   task.StartedAt,
		FinishedAt:  task.FinishedAt,
		ExpiresAt:   task.ExpiresAt,
		CanceledAt:  task.CanceledAt,
		UpdatedAt:   task.UpdatedAt,
		PollAfterMS: defaultImagePlaygroundTaskPollAfterMS,
	}
	if task.ErrorCode != nil {
		out.ErrorCode = *task.ErrorCode
	}
	if task.ErrorMessage != nil {
		out.ErrorMessage = *task.ErrorMessage
	}
	if includeResult && task.Status == service.ImagePlaygroundTaskStatusSucceeded && task.ExpiresAt.After(time.Now()) && len(task.ResultJSON) > 0 {
		out.Result = append(json.RawMessage(nil), task.ResultJSON...)
	}
	if task.Status == service.ImagePlaygroundTaskStatusSucceeded ||
		task.Status == service.ImagePlaygroundTaskStatusFailed ||
		task.Status == service.ImagePlaygroundTaskStatusCanceled ||
		task.Status == service.ImagePlaygroundTaskStatusExpired {
		out.PollAfterMS = 0
	}
	return out
}
