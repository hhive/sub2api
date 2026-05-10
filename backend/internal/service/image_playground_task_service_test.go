package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestImagePlaygroundTaskServiceCreateTaskEnforcesQueueLimits(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		counts ImagePlaygroundQueuedTaskCounts
		want   error
	}{
		{name: "user", counts: ImagePlaygroundQueuedTaskCounts{User: 2}, want: ErrImagePlaygroundTaskUserQueueFull},
		{name: "api key", counts: ImagePlaygroundQueuedTaskCounts{APIKey: 3}, want: ErrImagePlaygroundTaskAPIKeyQueueFull},
		{name: "global", counts: ImagePlaygroundQueuedTaskCounts{Global: 4}, want: ErrImagePlaygroundTaskGlobalQueueFull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &imagePlaygroundTaskRepoStub{counts: tt.counts}
			svc := NewImagePlaygroundTaskService(repo, nil, ImagePlaygroundTaskServiceOptions{
				Clock:                func() time.Time { return now },
				MaxQueuedPerUser:     2,
				MaxQueuedPerAPIKey:   3,
				MaxQueuedGlobal:      4,
				ResultTTL:            time.Hour,
				MaxRequestBodyBytes:  1024,
				WorkerCount:          1,
				WorkerPollInterval:   time.Millisecond,
				WorkerEmptyBackoff:   time.Millisecond,
				WorkerExecuteTimeout: time.Second,
			})

			task, err := svc.CreateTask(context.Background(), ImagePlaygroundTaskCreateRequest{
				UserID:      7,
				APIKeyID:    9,
				Endpoint:    ImagePlaygroundEndpointGenerations,
				RequestJSON: []byte(`{"prompt":"x"}`),
			})

			require.ErrorIs(t, err, tt.want)
			require.Nil(t, task)
			require.Empty(t, repo.created)
		})
	}
}

func TestImagePlaygroundTaskServiceCreateTaskUsesAtomicRepositoryCreate(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	repo := &imagePlaygroundTaskRepoStub{
		countErr: errors.New("separate count must not be called"),
	}
	svc := NewImagePlaygroundTaskService(repo, nil, ImagePlaygroundTaskServiceOptions{
		Clock:               func() time.Time { return now },
		MaxQueuedPerUser:    2,
		MaxQueuedPerAPIKey:  3,
		MaxQueuedGlobal:     4,
		ResultTTL:           time.Hour,
		MaxRequestBodyBytes: 1024,
	})

	task, err := svc.CreateTask(context.Background(), ImagePlaygroundTaskCreateRequest{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"prompt":"x"}`),
	})

	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, 0, repo.countCalls)
	require.Len(t, repo.created, 1)
	require.Equal(t, ImagePlaygroundTaskQueueLimits{MaxQueuedPerUser: 2, MaxQueuedPerAPIKey: 3, MaxQueuedGlobal: 4}, repo.lastLimits)
	require.Equal(t, now, repo.lastCreateNow)
}

func TestImagePlaygroundTaskServiceCreateTaskValidatesEndpointBodyAndTTL(t *testing.T) {
	tests := []struct {
		name string
		req  ImagePlaygroundTaskCreateRequest
		opts ImagePlaygroundTaskServiceOptions
		want error
	}{
		{
			name: "endpoint",
			req:  ImagePlaygroundTaskCreateRequest{UserID: 7, APIKeyID: 9, Endpoint: "/v1/chat/completions", RequestJSON: []byte(`{}`)},
			want: ErrImagePlaygroundTaskInvalidEndpoint,
		},
		{
			name: "body size",
			req:  ImagePlaygroundTaskCreateRequest{UserID: 7, APIKeyID: 9, Endpoint: ImagePlaygroundEndpointGenerations, RequestJSON: []byte(`{}`)},
			opts: ImagePlaygroundTaskServiceOptions{MaxRequestBodyBytes: 1, ResultTTL: time.Hour},
			want: ErrImagePlaygroundTaskBodyTooLarge,
		},
		{
			name: "negative ttl",
			req:  ImagePlaygroundTaskCreateRequest{UserID: 7, APIKeyID: 9, Endpoint: ImagePlaygroundEndpointGenerations, RequestJSON: []byte(`{}`)},
			opts: ImagePlaygroundTaskServiceOptions{ResultTTL: -time.Second},
			want: ErrImagePlaygroundTaskInvalidResultTTL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewImagePlaygroundTaskService(&imagePlaygroundTaskRepoStub{}, nil, tt.opts)
			task, err := svc.CreateTask(context.Background(), tt.req)
			require.ErrorIs(t, err, tt.want)
			require.Nil(t, task)
		})
	}
}

func TestImagePlaygroundTaskServiceCreateTaskZeroTTLUsesDefault(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	repo := &imagePlaygroundTaskRepoStub{}
	svc := NewImagePlaygroundTaskService(repo, nil, ImagePlaygroundTaskServiceOptions{
		Clock:               func() time.Time { return now },
		MaxRequestBodyBytes: 1024,
	})

	task, err := svc.CreateTask(context.Background(), ImagePlaygroundTaskCreateRequest{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{}`),
	})

	require.NoError(t, err)
	require.Equal(t, now.Add(defaultImagePlaygroundTaskResultTTL), task.ExpiresAt)
}

func TestImagePlaygroundTaskServiceWorkerClaimsAndPersistsSuccess(t *testing.T) {
	repo := newImagePlaygroundTaskRepoStubWithTasks(&ImagePlaygroundTask{
		ID:          11,
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    ImagePlaygroundEndpointGenerations,
		Status:      ImagePlaygroundTaskStatusQueued,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	executor := &imagePlaygroundTaskExecutorStub{result: []byte(`{"data":[{"b64_json":"ok"}]}`)}
	svc := NewImagePlaygroundTaskService(repo, executor, ImagePlaygroundTaskServiceOptions{
		WorkerCount:          1,
		WorkerPollInterval:   time.Millisecond,
		WorkerEmptyBackoff:   time.Millisecond,
		WorkerExecuteTimeout: time.Second,
	})

	require.NoError(t, svc.RunWorkerOnce(context.Background()))

	task := repo.mustTask(11)
	require.Equal(t, ImagePlaygroundTaskStatusSucceeded, task.Status)
	require.JSONEq(t, `{"data":[{"b64_json":"ok"}]}`, string(task.ResultJSON))
	require.NotNil(t, task.StartedAt)
	require.NotNil(t, task.FinishedAt)
	require.Equal(t, []int64{11}, executor.taskIDs())
}

func TestImagePlaygroundTaskServiceWorkerSkipsCanceledBeforeExecute(t *testing.T) {
	repo := newImagePlaygroundTaskRepoStubWithTasks(&ImagePlaygroundTask{
		ID:          11,
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    ImagePlaygroundEndpointGenerations,
		Status:      ImagePlaygroundTaskStatusQueued,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	repo.cancelAfterClaim = true
	executor := &imagePlaygroundTaskExecutorStub{result: []byte(`{"data":[]}`)}
	svc := NewImagePlaygroundTaskService(repo, executor, ImagePlaygroundTaskServiceOptions{
		WorkerCount:          1,
		WorkerPollInterval:   time.Millisecond,
		WorkerEmptyBackoff:   time.Millisecond,
		WorkerExecuteTimeout: time.Second,
	})

	require.NoError(t, svc.RunWorkerOnce(context.Background()))

	task := repo.mustTask(11)
	require.Equal(t, ImagePlaygroundTaskStatusCanceled, task.Status)
	require.Empty(t, executor.taskIDs())
}

func TestImagePlaygroundTaskServiceWorkerPersistsFailure(t *testing.T) {
	repo := newImagePlaygroundTaskRepoStubWithTasks(&ImagePlaygroundTask{
		ID:          12,
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    ImagePlaygroundEndpointGenerations,
		Status:      ImagePlaygroundTaskStatusQueued,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	executor := &imagePlaygroundTaskExecutorStub{err: ImagePlaygroundTaskExecutionError{
		Code:    "upstream_error",
		Message: "upstream failed",
	}}
	svc := NewImagePlaygroundTaskService(repo, executor, ImagePlaygroundTaskServiceOptions{
		WorkerCount:          1,
		WorkerPollInterval:   time.Millisecond,
		WorkerEmptyBackoff:   time.Millisecond,
		WorkerExecuteTimeout: time.Second,
	})

	require.NoError(t, svc.RunWorkerOnce(context.Background()))

	task := repo.mustTask(12)
	require.Equal(t, ImagePlaygroundTaskStatusFailed, task.Status)
	require.NotNil(t, task.ErrorCode)
	require.Equal(t, "upstream_error", *task.ErrorCode)
	require.NotNil(t, task.ErrorMessage)
	require.Equal(t, "upstream failed", *task.ErrorMessage)
}

func TestImagePlaygroundTaskServiceWorkerCanceledContextStillPersistsFailure(t *testing.T) {
	repo := newImagePlaygroundTaskRepoStubWithTasks(&ImagePlaygroundTask{
		ID:          13,
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    ImagePlaygroundEndpointGenerations,
		Status:      ImagePlaygroundTaskStatusQueued,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	executor := &imagePlaygroundTaskExecutorStub{delay: time.Second}
	svc := NewImagePlaygroundTaskService(repo, executor, ImagePlaygroundTaskServiceOptions{
		WorkerCount:          1,
		WorkerPollInterval:   time.Millisecond,
		WorkerEmptyBackoff:   time.Millisecond,
		WorkerExecuteTimeout: time.Second,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, svc.RunWorkerOnce(ctx))

	task := repo.mustTask(13)
	require.Equal(t, ImagePlaygroundTaskStatusFailed, task.Status)
	require.NotNil(t, task.ErrorCode)
	require.Equal(t, "execution_error", *task.ErrorCode)
}

func TestImagePlaygroundTaskServiceWorkerPoolUsesBoundedConcurrency(t *testing.T) {
	repo := newImagePlaygroundTaskRepoStubWithTasks(
		&ImagePlaygroundTask{ID: 1, UserID: 7, APIKeyID: 9, Endpoint: ImagePlaygroundEndpointGenerations, Status: ImagePlaygroundTaskStatusQueued, RequestJSON: []byte(`{}`), ExpiresAt: time.Now().Add(time.Hour)},
		&ImagePlaygroundTask{ID: 2, UserID: 7, APIKeyID: 9, Endpoint: ImagePlaygroundEndpointGenerations, Status: ImagePlaygroundTaskStatusQueued, RequestJSON: []byte(`{}`), ExpiresAt: time.Now().Add(time.Hour)},
		&ImagePlaygroundTask{ID: 3, UserID: 7, APIKeyID: 9, Endpoint: ImagePlaygroundEndpointGenerations, Status: ImagePlaygroundTaskStatusQueued, RequestJSON: []byte(`{}`), ExpiresAt: time.Now().Add(time.Hour)},
	)
	executor := &imagePlaygroundTaskExecutorStub{result: []byte(`{"data":[]}`), delay: 20 * time.Millisecond}
	svc := NewImagePlaygroundTaskService(repo, executor, ImagePlaygroundTaskServiceOptions{
		WorkerCount:          2,
		WorkerPollInterval:   time.Millisecond,
		WorkerEmptyBackoff:   time.Millisecond,
		WorkerExecuteTimeout: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	pool := svc.StartWorkerPool(ctx)
	require.NotNil(t, pool)
	require.Eventually(t, func() bool {
		return repo.finishedCount() == 3
	}, time.Second, 10*time.Millisecond)
	cancel()
	pool.Stop()

	require.LessOrEqual(t, executor.maxInflight(), 2)
}

func TestImagePlaygroundTaskServiceGetTaskMarksExpiredQueuedTask(t *testing.T) {
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	task := &ImagePlaygroundTask{
		ID:        10,
		UserID:    7,
		APIKeyID:  9,
		Status:    ImagePlaygroundTaskStatusQueued,
		Endpoint:  ImagePlaygroundEndpointGenerations,
		ExpiresAt: now.Add(-time.Minute),
	}
	repo := newImagePlaygroundTaskRepoStubWithTasks(task)
	svc := NewImagePlaygroundTaskService(repo, nil, ImagePlaygroundTaskServiceOptions{
		ResultTTL: time.Hour,
		Clock:     func() time.Time { return now },
	})

	got, err := svc.GetTask(context.Background(), 7, 10)

	require.NoError(t, err)
	require.Equal(t, ImagePlaygroundTaskStatusExpired, got.Status)
	require.Equal(t, ImagePlaygroundTaskStatusExpired, repo.mustTask(10).Status)
}

func TestImagePlaygroundTaskServiceCancelExpiredQueuedTaskReturnsExpired(t *testing.T) {
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	task := &ImagePlaygroundTask{
		ID:        10,
		UserID:    7,
		APIKeyID:  9,
		Status:    ImagePlaygroundTaskStatusQueued,
		Endpoint:  ImagePlaygroundEndpointGenerations,
		ExpiresAt: now.Add(-time.Minute),
	}
	repo := newImagePlaygroundTaskRepoStubWithTasks(task)
	svc := NewImagePlaygroundTaskService(repo, nil, ImagePlaygroundTaskServiceOptions{
		ResultTTL: time.Hour,
		Clock:     func() time.Time { return now },
	})

	got, err := svc.CancelTask(context.Background(), 7, 10)

	require.NoError(t, err)
	require.Equal(t, ImagePlaygroundTaskStatusExpired, got.Status)
	require.Equal(t, ImagePlaygroundTaskStatusExpired, repo.mustTask(10).Status)
}

func TestImagePlaygroundTaskServiceListRecentTasksFiltersExpired(t *testing.T) {
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	repo := newImagePlaygroundTaskRepoStubWithTasks(
		&ImagePlaygroundTask{ID: 10, UserID: 7, APIKeyID: 9, Status: ImagePlaygroundTaskStatusSucceeded, Endpoint: ImagePlaygroundEndpointGenerations, ExpiresAt: now.Add(time.Hour)},
		&ImagePlaygroundTask{ID: 11, UserID: 7, APIKeyID: 9, Status: ImagePlaygroundTaskStatusSucceeded, Endpoint: ImagePlaygroundEndpointGenerations, ExpiresAt: now.Add(-time.Minute)},
	)
	svc := NewImagePlaygroundTaskService(repo, nil, ImagePlaygroundTaskServiceOptions{
		ResultTTL: time.Hour,
		Clock:     func() time.Time { return now },
	})

	got, err := svc.ListRecentTasks(context.Background(), 7, 20)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(10), got[0].ID)
}

type imagePlaygroundTaskRepoStub struct {
	mu               sync.Mutex
	nextID           int64
	counts           ImagePlaygroundQueuedTaskCounts
	countErr         error
	countCalls       int
	lastLimits       ImagePlaygroundTaskQueueLimits
	lastCreateNow    time.Time
	tasks            map[int64]*ImagePlaygroundTask
	created          []ImagePlaygroundTask
	cancelAfterClaim bool
}

func newImagePlaygroundTaskRepoStubWithTasks(tasks ...*ImagePlaygroundTask) *imagePlaygroundTaskRepoStub {
	repo := &imagePlaygroundTaskRepoStub{tasks: map[int64]*ImagePlaygroundTask{}}
	for _, task := range tasks {
		repo.tasks[task.ID] = cloneImagePlaygroundTask(task)
		if task.ID > repo.nextID {
			repo.nextID = task.ID
		}
	}
	return repo
}

func (r *imagePlaygroundTaskRepoStub) CreateTask(ctx context.Context, task *ImagePlaygroundTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tasks == nil {
		r.tasks = map[int64]*ImagePlaygroundTask{}
	}
	r.nextID++
	task.ID = r.nextID
	task.Status = ImagePlaygroundTaskStatusQueued
	task.CreatedAt = task.ExpiresAt.Add(-time.Hour)
	task.UpdatedAt = task.CreatedAt
	r.tasks[task.ID] = cloneImagePlaygroundTask(task)
	r.created = append(r.created, *cloneImagePlaygroundTask(task))
	return nil
}

func (r *imagePlaygroundTaskRepoStub) CreateTaskIfQueueAvailable(ctx context.Context, task *ImagePlaygroundTask, limits ImagePlaygroundTaskQueueLimits, now time.Time) error {
	r.mu.Lock()
	r.lastLimits = limits
	r.lastCreateNow = now
	counts := r.counts
	if r.tasks != nil {
		counts = ImagePlaygroundQueuedTaskCounts{}
		for _, existing := range r.tasks {
			if existing.Status != ImagePlaygroundTaskStatusQueued || !existing.ExpiresAt.After(now) {
				continue
			}
			counts.Global++
			if existing.UserID == task.UserID {
				counts.User++
			}
			if existing.APIKeyID == task.APIKeyID {
				counts.APIKey++
			}
		}
	}
	r.mu.Unlock()
	if limits.MaxQueuedPerUser > 0 && counts.User >= limits.MaxQueuedPerUser {
		return ErrImagePlaygroundTaskUserQueueFull
	}
	if limits.MaxQueuedPerAPIKey > 0 && counts.APIKey >= limits.MaxQueuedPerAPIKey {
		return ErrImagePlaygroundTaskAPIKeyQueueFull
	}
	if limits.MaxQueuedGlobal > 0 && counts.Global >= limits.MaxQueuedGlobal {
		return ErrImagePlaygroundTaskGlobalQueueFull
	}
	return r.CreateTask(ctx, task)
}

func (r *imagePlaygroundTaskRepoStub) GetTaskByOwner(ctx context.Context, userID int64, taskID int64) (*ImagePlaygroundTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, errors.New("not found")
	}
	return cloneImagePlaygroundTask(task), nil
}

func (r *imagePlaygroundTaskRepoStub) ListRecentTasksByOwner(ctx context.Context, userID int64, limit int) ([]ImagePlaygroundTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ImagePlaygroundTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		if task.UserID == userID {
			out = append(out, *cloneImagePlaygroundTask(task))
		}
	}
	return out, nil
}

func (r *imagePlaygroundTaskRepoStub) CountQueuedTasks(ctx context.Context, userID int64, apiKeyID int64, now time.Time) (ImagePlaygroundQueuedTaskCounts, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.countCalls++
	if r.countErr != nil {
		return ImagePlaygroundQueuedTaskCounts{}, r.countErr
	}
	if r.tasks == nil {
		return r.counts, nil
	}
	var counts ImagePlaygroundQueuedTaskCounts
	for _, task := range r.tasks {
		if task.Status != ImagePlaygroundTaskStatusQueued || !task.ExpiresAt.After(now) {
			continue
		}
		counts.Global++
		if task.UserID == userID {
			counts.User++
		}
		if task.APIKeyID == apiKeyID {
			counts.APIKey++
		}
	}
	return counts, nil
}

func (r *imagePlaygroundTaskRepoStub) CancelTask(ctx context.Context, userID int64, taskID int64, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[taskID]
	if !ok || task.UserID != userID {
		return false, nil
	}
	if task.Status != ImagePlaygroundTaskStatusQueued && task.Status != ImagePlaygroundTaskStatusRunning {
		return false, nil
	}
	if !task.ExpiresAt.After(now) {
		return false, nil
	}
	task.Status = ImagePlaygroundTaskStatusCanceled
	task.CanceledAt = &now
	task.FinishedAt = &now
	task.UpdatedAt = now
	return true, nil
}

func (r *imagePlaygroundTaskRepoStub) ClaimNextQueuedTask(ctx context.Context, now time.Time) (*ImagePlaygroundTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var next *ImagePlaygroundTask
	for _, task := range r.tasks {
		if task.Status != ImagePlaygroundTaskStatusQueued || !task.ExpiresAt.After(now) {
			continue
		}
		if next == nil || task.ID < next.ID {
			next = task
		}
	}
	if next == nil {
		return nil, nil
	}
	next.Status = ImagePlaygroundTaskStatusRunning
	next.StartedAt = &now
	next.UpdatedAt = now
	if r.cancelAfterClaim {
		next.Status = ImagePlaygroundTaskStatusCanceled
		next.CanceledAt = &now
		next.FinishedAt = &now
	}
	return cloneImagePlaygroundTask(next), nil
}

func (r *imagePlaygroundTaskRepoStub) MarkTaskRunning(ctx context.Context, taskID int64, now time.Time) (bool, error) {
	return false, nil
}

func (r *imagePlaygroundTaskRepoStub) MarkTaskSucceeded(ctx context.Context, taskID int64, resultJSON []byte, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	task, ok := r.tasks[taskID]
	if !ok || task.Status != ImagePlaygroundTaskStatusRunning {
		return false, nil
	}
	task.Status = ImagePlaygroundTaskStatusSucceeded
	task.ResultJSON = append([]byte(nil), resultJSON...)
	task.FinishedAt = &now
	task.UpdatedAt = now
	return true, nil
}

func (r *imagePlaygroundTaskRepoStub) MarkTaskFailed(ctx context.Context, taskID int64, errorCode string, errorMessage string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	task, ok := r.tasks[taskID]
	if !ok || task.Status != ImagePlaygroundTaskStatusRunning {
		return false, nil
	}
	task.Status = ImagePlaygroundTaskStatusFailed
	task.ErrorCode = &errorCode
	task.ErrorMessage = &errorMessage
	task.FinishedAt = &now
	task.UpdatedAt = now
	return true, nil
}

func (r *imagePlaygroundTaskRepoStub) MarkTaskExpired(ctx context.Context, taskID int64, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.tasks[taskID]
	if !ok {
		return false, nil
	}
	if task.Status != ImagePlaygroundTaskStatusQueued && task.Status != ImagePlaygroundTaskStatusRunning {
		return false, nil
	}
	task.Status = ImagePlaygroundTaskStatusExpired
	task.FinishedAt = &now
	task.UpdatedAt = now
	return true, nil
}

func (r *imagePlaygroundTaskRepoStub) mustTask(taskID int64) *ImagePlaygroundTask {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneImagePlaygroundTask(r.tasks[taskID])
}

func (r *imagePlaygroundTaskRepoStub) finishedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, task := range r.tasks {
		switch task.Status {
		case ImagePlaygroundTaskStatusSucceeded, ImagePlaygroundTaskStatusFailed, ImagePlaygroundTaskStatusCanceled:
			count++
		}
	}
	return count
}

type imagePlaygroundTaskExecutorStub struct {
	mu       sync.Mutex
	result   []byte
	err      error
	delay    time.Duration
	ids      []int64
	inflight int
	maxRun   int
}

func (e *imagePlaygroundTaskExecutorStub) ExecuteImagePlaygroundTask(ctx context.Context, task ImagePlaygroundTask) ([]byte, error) {
	e.mu.Lock()
	e.ids = append(e.ids, task.ID)
	e.inflight++
	if e.inflight > e.maxRun {
		e.maxRun = e.inflight
	}
	e.mu.Unlock()

	if e.delay > 0 {
		select {
		case <-time.After(e.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	e.mu.Lock()
	e.inflight--
	e.mu.Unlock()

	if e.err != nil {
		return nil, e.err
	}
	return append([]byte(nil), e.result...), nil
}

func (e *imagePlaygroundTaskExecutorStub) taskIDs() []int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]int64(nil), e.ids...)
}

func (e *imagePlaygroundTaskExecutorStub) maxInflight() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.maxRun
}
