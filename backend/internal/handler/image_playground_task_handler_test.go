package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImagePlaygroundTaskHandlerCreateUsesAPIKeyAndDoesNotEchoSecret(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	tasks.createTask = func(ctx context.Context, req service.ImagePlaygroundTaskCreateRequest) (*service.ImagePlaygroundTask, error) {
		require.Equal(t, int64(7), req.UserID)
		require.Equal(t, int64(9), req.APIKeyID)
		require.JSONEq(t, `{"prompt":"draw"}`, string(req.RequestJSON))
		return imagePlaygroundTaskHandlerTestTask(101, 7, service.ImagePlaygroundTaskStatusQueued), nil
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodPost, "/image-playground/tasks", `{
		"api_key":"sk-secret",
		"endpoint":"/v1/images/generations",
		"request":{"prompt":"draw"}
	}`)

	require.Equal(t, http.StatusAccepted, resp.Code)
	require.NotContains(t, resp.Body.String(), "sk-secret")
	require.NotContains(t, resp.Body.String(), `"request"`)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	require.Equal(t, float64(101), data["task_id"])
	require.Equal(t, service.ImagePlaygroundTaskStatusQueued, data["status"])
	require.Equal(t, float64(defaultImagePlaygroundTaskPollAfterMS), data["poll_after_ms"])
}

func TestImagePlaygroundTaskHandlerCreateQueueLimitReturns429(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	tasks.createTask = func(ctx context.Context, req service.ImagePlaygroundTaskCreateRequest) (*service.ImagePlaygroundTask, error) {
		return nil, service.ErrImagePlaygroundTaskUserQueueFull
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodPost, "/image-playground/tasks", `{
		"api_key_id":9,
		"endpoint":"/v1/images/generations",
		"request":{"prompt":"draw"}
	}`)

	require.Equal(t, http.StatusTooManyRequests, resp.Code)
	require.Contains(t, resp.Body.String(), "IMAGE_PLAYGROUND_TASK_USER_QUEUE_FULL")
}

func TestImagePlaygroundTaskHandlerGetSucceededIncludesResult(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	tasks.getTask = func(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error) {
		require.Equal(t, int64(7), userID)
		require.Equal(t, int64(101), taskID)
		task := imagePlaygroundTaskHandlerTestTask(101, 7, service.ImagePlaygroundTaskStatusSucceeded)
		task.ResultJSON = []byte(`{"data":[{"b64_json":"aW1hZ2U="}]}`)
		return task, nil
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodGet, "/image-playground/tasks/101", "")

	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), `"result":{"data":[{"b64_json":"aW1hZ2U="}]}`)
	require.NotContains(t, resp.Body.String(), "api_key")
}

func TestImagePlaygroundTaskHandlerGetFailedIncludesErrorWithoutResult(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	code := "handler_status_429"
	message := "Image generation concurrency limit exceeded"
	tasks.getTask = func(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error) {
		task := imagePlaygroundTaskHandlerTestTask(taskID, userID, service.ImagePlaygroundTaskStatusFailed)
		task.ErrorCode = &code
		task.ErrorMessage = &message
		task.ResultJSON = []byte(`{"data":[{"b64_json":"should-not-return"}]}`)
		return task, nil
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodGet, "/image-playground/tasks/101", "")

	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), code)
	require.Contains(t, resp.Body.String(), message)
	require.NotContains(t, resp.Body.String(), "should-not-return")
}

func TestImagePlaygroundTaskHandlerGetExpiredSucceededDoesNotReturnResult(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	tasks.getTask = func(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error) {
		task := imagePlaygroundTaskHandlerTestTask(taskID, userID, service.ImagePlaygroundTaskStatusSucceeded)
		task.ExpiresAt = time.Now().Add(-time.Minute)
		task.ResultJSON = []byte(`{"data":[{"b64_json":"expired"}]}`)
		return task, nil
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodGet, "/image-playground/tasks/101", "")

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotContains(t, resp.Body.String(), "expired")
	require.Contains(t, resp.Body.String(), service.ImagePlaygroundTaskStatusSucceeded)
}

func TestImagePlaygroundTaskHandlerMissingServiceReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewImagePlaygroundTaskHandler(nil, nil)
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Next()
	})
	router.GET("/image-playground/tasks/:id", handler.Get)
	router.POST("/image-playground/tasks/:id/cancel", handler.Cancel)
	router.GET("/image-playground/tasks/recent", handler.Recent)

	for _, tt := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/image-playground/tasks/101"},
		{method: http.MethodPost, path: "/image-playground/tasks/101/cancel"},
		{method: http.MethodGet, path: "/image-playground/tasks/recent"},
	} {
		resp := performImagePlaygroundTaskRequest(router, tt.method, tt.path, "")
		require.Equal(t, http.StatusInternalServerError, resp.Code)
	}
}

func TestImagePlaygroundTaskHandlerCancelUsesOwner(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	tasks.cancelTask = func(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error) {
		require.Equal(t, int64(7), userID)
		require.Equal(t, int64(101), taskID)
		return imagePlaygroundTaskHandlerTestTask(101, 7, service.ImagePlaygroundTaskStatusCanceled), nil
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodPost, "/image-playground/tasks/101/cancel", "")

	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), service.ImagePlaygroundTaskStatusCanceled)
}

func TestImagePlaygroundTaskHandlerRecentReturnsRestorableTasks(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	tasks.listRecentTasks = func(ctx context.Context, userID int64, limit int) ([]service.ImagePlaygroundTask, error) {
		require.Equal(t, int64(7), userID)
		require.Equal(t, 5, limit)
		task := imagePlaygroundTaskHandlerTestTask(101, 7, service.ImagePlaygroundTaskStatusSucceeded)
		task.ResultJSON = []byte(`{"data":[{"url":"data:image/png;base64,aW1hZ2U="}]}`)
		return []service.ImagePlaygroundTask{*task}, nil
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodGet, "/image-playground/tasks/recent?limit=5", "")

	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), `"task_id":101`)
	require.Contains(t, resp.Body.String(), `"result":{"data":[{"url":"data:image/png;base64,aW1hZ2U="}]}`)
}

func TestImagePlaygroundTaskHandlerCrossUserTaskReturns404(t *testing.T) {
	router, tasks, _ := newImagePlaygroundTaskHandlerTestRouter(t)
	tasks.getTask = func(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error) {
		return nil, service.ErrImagePlaygroundTaskNotFound
	}

	resp := performImagePlaygroundTaskRequest(router, http.MethodGet, "/image-playground/tasks/999", "")

	require.Equal(t, http.StatusNotFound, resp.Code)
	require.Contains(t, resp.Body.String(), "IMAGE_PLAYGROUND_TASK_NOT_FOUND")
}

func newImagePlaygroundTaskHandlerTestRouter(t *testing.T) (*gin.Engine, *imagePlaygroundTaskManagerStub, *imagePlaygroundAPIKeyResolverStub) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tasks := &imagePlaygroundTaskManagerStub{}
	keys := &imagePlaygroundAPIKeyResolverStub{
		byID:  map[int64]*service.APIKey{9: {ID: 9, UserID: 7}},
		byKey: map[string]*service.APIKey{"sk-secret": {ID: 9, UserID: 7}},
	}
	h := NewImagePlaygroundTaskHandler(tasks, keys)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7, Concurrency: 3})
		c.Next()
	})
	router.POST("/image-playground/tasks", h.Create)
	router.GET("/image-playground/tasks/recent", h.Recent)
	router.GET("/image-playground/tasks/:id", h.Get)
	router.POST("/image-playground/tasks/:id/cancel", h.Cancel)
	return router, tasks, keys
}

func performImagePlaygroundTaskRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func imagePlaygroundTaskHandlerTestTask(id, userID int64, status string) *service.ImagePlaygroundTask {
	now := time.Now()
	return &service.ImagePlaygroundTask{
		ID:        id,
		UserID:    userID,
		APIKeyID:  9,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    status,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
		UpdatedAt: now,
	}
}

type imagePlaygroundTaskManagerStub struct {
	createTask      func(context.Context, service.ImagePlaygroundTaskCreateRequest) (*service.ImagePlaygroundTask, error)
	getTask         func(context.Context, int64, int64) (*service.ImagePlaygroundTask, error)
	cancelTask      func(context.Context, int64, int64) (*service.ImagePlaygroundTask, error)
	listRecentTasks func(context.Context, int64, int) ([]service.ImagePlaygroundTask, error)
}

func (s *imagePlaygroundTaskManagerStub) CreateTask(ctx context.Context, req service.ImagePlaygroundTaskCreateRequest) (*service.ImagePlaygroundTask, error) {
	if s.createTask != nil {
		return s.createTask(ctx, req)
	}
	return imagePlaygroundTaskHandlerTestTask(101, req.UserID, service.ImagePlaygroundTaskStatusQueued), nil
}

func (s *imagePlaygroundTaskManagerStub) GetTask(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error) {
	if s.getTask != nil {
		return s.getTask(ctx, userID, taskID)
	}
	return nil, service.ErrImagePlaygroundTaskNotFound
}

func (s *imagePlaygroundTaskManagerStub) CancelTask(ctx context.Context, userID, taskID int64) (*service.ImagePlaygroundTask, error) {
	if s.cancelTask != nil {
		return s.cancelTask(ctx, userID, taskID)
	}
	return nil, service.ErrImagePlaygroundTaskNotFound
}

func (s *imagePlaygroundTaskManagerStub) ListRecentTasks(ctx context.Context, userID int64, limit int) ([]service.ImagePlaygroundTask, error) {
	if s.listRecentTasks != nil {
		return s.listRecentTasks(ctx, userID, limit)
	}
	return nil, nil
}

type imagePlaygroundAPIKeyResolverStub struct {
	byID  map[int64]*service.APIKey
	byKey map[string]*service.APIKey
}

func (s *imagePlaygroundAPIKeyResolverStub) GetByID(ctx context.Context, id int64) (*service.APIKey, error) {
	if key := s.byID[id]; key != nil {
		return key, nil
	}
	return nil, service.ErrAPIKeyNotFound
}

func (s *imagePlaygroundAPIKeyResolverStub) GetByKey(ctx context.Context, key string) (*service.APIKey, error) {
	if apiKey := s.byKey[key]; apiKey != nil {
		return apiKey, nil
	}
	return nil, service.ErrAPIKeyNotFound
}
