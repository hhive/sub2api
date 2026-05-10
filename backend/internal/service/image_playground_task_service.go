package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	defaultImagePlaygroundTaskBodyBytes      = 8 * 1024 * 1024
	defaultImagePlaygroundTaskResultTTL      = 24 * time.Hour
	defaultImagePlaygroundTaskUserQueueLimit = 10
	defaultImagePlaygroundTaskKeyQueueLimit  = 5
	defaultImagePlaygroundTaskQueueLimit     = 100
	defaultImagePlaygroundTaskWorkers        = 2
	defaultImagePlaygroundWorkerPollInterval = 250 * time.Millisecond
	defaultImagePlaygroundWorkerEmptyBackoff = time.Second
	defaultImagePlaygroundWorkerExecTimeout  = 10 * time.Minute
	defaultImagePlaygroundTerminalWriteTTL   = 5 * time.Second
)

var (
	ErrImagePlaygroundTaskInvalidEndpoint  = infraerrors.BadRequest("IMAGE_PLAYGROUND_TASK_INVALID_ENDPOINT", "image playground endpoint is invalid")
	ErrImagePlaygroundTaskBodyTooLarge     = infraerrors.BadRequest("IMAGE_PLAYGROUND_TASK_BODY_TOO_LARGE", "image playground request body is too large")
	ErrImagePlaygroundTaskInvalidResultTTL = infraerrors.BadRequest("IMAGE_PLAYGROUND_TASK_INVALID_RESULT_TTL", "image playground result ttl must be positive")
	ErrImagePlaygroundTaskUserQueueFull    = infraerrors.TooManyRequests("IMAGE_PLAYGROUND_TASK_USER_QUEUE_FULL", "too many queued image playground tasks for this user")
	ErrImagePlaygroundTaskAPIKeyQueueFull  = infraerrors.TooManyRequests("IMAGE_PLAYGROUND_TASK_API_KEY_QUEUE_FULL", "too many queued image playground tasks for this api key")
	ErrImagePlaygroundTaskGlobalQueueFull  = infraerrors.TooManyRequests("IMAGE_PLAYGROUND_TASK_GLOBAL_QUEUE_FULL", "too many queued image playground tasks")
	ErrImagePlaygroundTaskExecutorMissing  = infraerrors.BadRequest("IMAGE_PLAYGROUND_TASK_EXECUTOR_MISSING", "image playground task executor is not configured")
)

type ImagePlaygroundTaskCreateRequest struct {
	UserID      int64
	APIKeyID    int64
	Endpoint    string
	RequestJSON []byte
}

type ImagePlaygroundTaskExecutor interface {
	ExecuteImagePlaygroundTask(ctx context.Context, task ImagePlaygroundTask) ([]byte, error)
}

type ImagePlaygroundTaskExecutionError struct {
	Code    string
	Message string
}

func (e ImagePlaygroundTaskExecutionError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return e.Code
}

type ImagePlaygroundTaskServiceOptions struct {
	MaxRequestBodyBytes  int
	ResultTTL            time.Duration
	MaxQueuedPerUser     int
	MaxQueuedPerAPIKey   int
	MaxQueuedGlobal      int
	WorkerCount          int
	WorkerPollInterval   time.Duration
	WorkerEmptyBackoff   time.Duration
	WorkerExecuteTimeout time.Duration
	Clock                func() time.Time
}

type ImagePlaygroundTaskService struct {
	repo     ImagePlaygroundTaskRepository
	executor ImagePlaygroundTaskExecutor
	opts     ImagePlaygroundTaskServiceOptions
}

func NewImagePlaygroundTaskService(repo ImagePlaygroundTaskRepository, executor ImagePlaygroundTaskExecutor, opts ImagePlaygroundTaskServiceOptions) *ImagePlaygroundTaskService {
	opts = normalizeImagePlaygroundTaskServiceOptions(opts)
	return &ImagePlaygroundTaskService{
		repo:     repo,
		executor: executor,
		opts:     opts,
	}
}

func (s *ImagePlaygroundTaskService) CreateTask(ctx context.Context, req ImagePlaygroundTaskCreateRequest) (*ImagePlaygroundTask, error) {
	if err := validateImagePlaygroundTaskCreateRequest(req, s.opts); err != nil {
		return nil, err
	}
	now := s.now()
	task := &ImagePlaygroundTask{
		UserID:      req.UserID,
		APIKeyID:    req.APIKeyID,
		Endpoint:    strings.TrimSpace(req.Endpoint),
		Status:      ImagePlaygroundTaskStatusQueued,
		RequestJSON: append([]byte(nil), req.RequestJSON...),
		ExpiresAt:   now.Add(s.opts.ResultTTL),
	}
	if err := s.repo.CreateTaskIfQueueAvailable(ctx, task, ImagePlaygroundTaskQueueLimits{
		MaxQueuedPerUser:   s.opts.MaxQueuedPerUser,
		MaxQueuedPerAPIKey: s.opts.MaxQueuedPerAPIKey,
		MaxQueuedGlobal:    s.opts.MaxQueuedGlobal,
	}, now); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *ImagePlaygroundTaskService) StartWorkerPool(ctx context.Context) *ImagePlaygroundTaskWorkerPool {
	if s == nil {
		return nil
	}
	pool := &ImagePlaygroundTaskWorkerPool{
		svc:    s,
		ctx:    ctx,
		cancel: func() {},
	}
	if pool.ctx == nil {
		pool.ctx = context.Background()
	}
	pool.ctx, pool.cancel = context.WithCancel(pool.ctx)
	pool.wg.Add(s.opts.WorkerCount)
	for i := 0; i < s.opts.WorkerCount; i++ {
		go func() {
			defer pool.wg.Done()
			s.runWorker(pool.ctx)
		}()
	}
	return pool
}

func (s *ImagePlaygroundTaskService) RunWorkerOnce(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return nil
	}
	task, err := s.repo.ClaimNextQueuedTask(ctx, s.now())
	if err != nil || task == nil {
		return err
	}
	return s.executeClaimedTask(ctx, *task)
}

func (s *ImagePlaygroundTaskService) runWorker(ctx context.Context) {
	ticker := time.NewTicker(s.opts.WorkerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if err := s.RunWorkerOnce(ctx); err != nil {
			sleepContext(ctx, s.opts.WorkerEmptyBackoff)
			continue
		}
	}
}

func (s *ImagePlaygroundTaskService) executeClaimedTask(ctx context.Context, task ImagePlaygroundTask) error {
	current, err := s.repo.GetTaskByOwner(ctx, task.UserID, task.ID)
	if err == nil && current != nil && current.Status != ImagePlaygroundTaskStatusRunning {
		return nil
	}
	if s.executor == nil {
		persistCtx, cancel := imagePlaygroundTerminalWriteContext()
		defer cancel()
		_, markErr := s.repo.MarkTaskFailed(persistCtx, task.ID, "executor_missing", ErrImagePlaygroundTaskExecutorMissing.Error(), s.now())
		return markErr
	}

	execCtx, cancel := context.WithTimeout(ctx, s.opts.WorkerExecuteTimeout)
	defer cancel()
	resultJSON, execErr := s.executor.ExecuteImagePlaygroundTask(execCtx, task)
	now := s.now()
	persistCtx, persistCancel := imagePlaygroundTerminalWriteContext()
	defer persistCancel()
	if execErr != nil {
		code, message := imagePlaygroundExecutionFailure(execErr)
		_, err = s.repo.MarkTaskFailed(persistCtx, task.ID, code, message, now)
		return err
	}
	_, err = s.repo.MarkTaskSucceeded(persistCtx, task.ID, resultJSON, now)
	return err
}

func imagePlaygroundExecutionFailure(err error) (string, string) {
	var execErr ImagePlaygroundTaskExecutionError
	if errors.As(err, &execErr) {
		code := strings.TrimSpace(execErr.Code)
		if code == "" {
			code = "execution_error"
		}
		msg := strings.TrimSpace(execErr.Message)
		if msg == "" {
			msg = err.Error()
		}
		return code, msg
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "image playground task failed"
	}
	return "execution_error", msg
}

func validateImagePlaygroundTaskCreateRequest(req ImagePlaygroundTaskCreateRequest, opts ImagePlaygroundTaskServiceOptions) error {
	switch strings.TrimSpace(req.Endpoint) {
	case ImagePlaygroundEndpointGenerations, ImagePlaygroundEndpointEdits:
	default:
		return ErrImagePlaygroundTaskInvalidEndpoint
	}
	if opts.ResultTTL <= 0 {
		return ErrImagePlaygroundTaskInvalidResultTTL
	}
	if len(req.RequestJSON) > opts.MaxRequestBodyBytes {
		return ErrImagePlaygroundTaskBodyTooLarge
	}
	return nil
}

func normalizeImagePlaygroundTaskServiceOptions(opts ImagePlaygroundTaskServiceOptions) ImagePlaygroundTaskServiceOptions {
	if opts.MaxRequestBodyBytes <= 0 {
		opts.MaxRequestBodyBytes = defaultImagePlaygroundTaskBodyBytes
	}
	if opts.ResultTTL == 0 {
		opts.ResultTTL = defaultImagePlaygroundTaskResultTTL
	}
	if opts.MaxQueuedPerUser <= 0 {
		opts.MaxQueuedPerUser = defaultImagePlaygroundTaskUserQueueLimit
	}
	if opts.MaxQueuedPerAPIKey <= 0 {
		opts.MaxQueuedPerAPIKey = defaultImagePlaygroundTaskKeyQueueLimit
	}
	if opts.MaxQueuedGlobal <= 0 {
		opts.MaxQueuedGlobal = defaultImagePlaygroundTaskQueueLimit
	}
	if opts.WorkerCount <= 0 || opts.WorkerCount > defaultImagePlaygroundTaskWorkers {
		opts.WorkerCount = defaultImagePlaygroundTaskWorkers
	}
	if opts.WorkerPollInterval <= 0 {
		opts.WorkerPollInterval = defaultImagePlaygroundWorkerPollInterval
	}
	if opts.WorkerEmptyBackoff <= 0 {
		opts.WorkerEmptyBackoff = defaultImagePlaygroundWorkerEmptyBackoff
	}
	if opts.WorkerExecuteTimeout <= 0 {
		opts.WorkerExecuteTimeout = defaultImagePlaygroundWorkerExecTimeout
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return opts
}

func imagePlaygroundTerminalWriteContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultImagePlaygroundTerminalWriteTTL)
}

func (s *ImagePlaygroundTaskService) now() time.Time {
	if s == nil || s.opts.Clock == nil {
		return time.Now()
	}
	return s.opts.Clock()
}

type ImagePlaygroundTaskWorkerPool struct {
	svc    *ImagePlaygroundTaskService
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (p *ImagePlaygroundTaskWorkerPool) Stop() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func sleepContext(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func cloneImagePlaygroundTask(task *ImagePlaygroundTask) *ImagePlaygroundTask {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.RequestJSON = append([]byte(nil), task.RequestJSON...)
	cloned.ResultJSON = append([]byte(nil), task.ResultJSON...)
	if task.GroupID != nil {
		v := *task.GroupID
		cloned.GroupID = &v
	}
	if task.ErrorCode != nil {
		v := *task.ErrorCode
		cloned.ErrorCode = &v
	}
	if task.ErrorMessage != nil {
		v := *task.ErrorMessage
		cloned.ErrorMessage = &v
	}
	if task.StartedAt != nil {
		v := *task.StartedAt
		cloned.StartedAt = &v
	}
	if task.FinishedAt != nil {
		v := *task.FinishedAt
		cloned.FinishedAt = &v
	}
	if task.CanceledAt != nil {
		v := *task.CanceledAt
		cloned.CanceledAt = &v
	}
	return &cloned
}
