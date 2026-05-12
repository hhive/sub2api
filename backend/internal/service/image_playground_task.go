package service

import (
	"context"
	"time"
)

const (
	ImagePlaygroundTaskStatusQueued    = "queued"
	ImagePlaygroundTaskStatusRunning   = "running"
	ImagePlaygroundTaskStatusSucceeded = "succeeded"
	ImagePlaygroundTaskStatusFailed    = "failed"
	ImagePlaygroundTaskStatusCanceled  = "canceled"
	ImagePlaygroundTaskStatusExpired   = "expired"

	ImagePlaygroundEndpointGenerations = "/v1/images/generations"
	ImagePlaygroundEndpointEdits       = "/v1/images/edits"
)

type ImagePlaygroundTask struct {
	ID           int64
	UserID       int64
	APIKeyID     int64
	GroupID      *int64
	Endpoint     string
	Status       string
	RequestJSON  []byte
	ResultJSON   []byte
	ErrorCode    *string
	ErrorMessage *string
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
	ExpiresAt    time.Time
	CanceledAt   *time.Time
	UpdatedAt    time.Time
}

type ImagePlaygroundTaskRepository interface {
	CreateTask(ctx context.Context, task *ImagePlaygroundTask) error
	CreateTaskIfQueueAvailable(ctx context.Context, task *ImagePlaygroundTask, limits ImagePlaygroundTaskQueueLimits, now time.Time) error
	GetTaskByOwner(ctx context.Context, userID int64, taskID int64) (*ImagePlaygroundTask, error)
	ListRecentTasksByOwner(ctx context.Context, userID int64, limit int) ([]ImagePlaygroundTask, error)
	CountQueuedTasks(ctx context.Context, userID int64, apiKeyID int64, now time.Time) (ImagePlaygroundQueuedTaskCounts, error)
	CancelTask(ctx context.Context, userID int64, taskID int64, now time.Time) (bool, error)
	ClaimNextQueuedTask(ctx context.Context, now time.Time) (*ImagePlaygroundTask, error)
	MarkTaskRunning(ctx context.Context, taskID int64, now time.Time) (bool, error)
	MarkTaskSucceeded(ctx context.Context, taskID int64, resultJSON []byte, now time.Time) (bool, error)
	MarkTaskFailed(ctx context.Context, taskID int64, errorCode string, errorMessage string, now time.Time) (bool, error)
	MarkTaskExpired(ctx context.Context, taskID int64, now time.Time) (bool, error)
	CleanupExpiredPayloads(ctx context.Context, now time.Time, batchSize int) (int64, error)
}

type ImagePlaygroundQueuedTaskCounts struct {
	User   int
	APIKey int
	Global int
}

type ImagePlaygroundTaskQueueLimits struct {
	MaxQueuedPerUser   int
	MaxQueuedPerAPIKey int
	MaxQueuedGlobal    int
}
