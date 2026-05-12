package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type imagePlaygroundTaskRepository struct {
	sql sqlExecutor
}

type sqlTxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

const imagePlaygroundTaskQueueAdvisoryLockKey int64 = 472019050210

func NewImagePlaygroundTaskRepository(sqlDB *sql.DB) service.ImagePlaygroundTaskRepository {
	return newImagePlaygroundTaskRepositoryWithSQL(sqlDB)
}

func newImagePlaygroundTaskRepositoryWithSQL(sqlq sqlExecutor) *imagePlaygroundTaskRepository {
	return &imagePlaygroundTaskRepository{sql: sqlq}
}

func (r *imagePlaygroundTaskRepository) CreateTask(ctx context.Context, task *service.ImagePlaygroundTask) error {
	if task == nil {
		return nil
	}
	if err := validateImagePlaygroundEndpoint(task.Endpoint); err != nil {
		return err
	}
	query := `
		INSERT INTO image_playground_tasks (
			user_id, api_key_id, group_id, endpoint, status, request_json, expires_at
		)
		SELECT api_keys.user_id, api_keys.id, api_keys.group_id, $3, 'queued', $4, $5
		FROM api_keys
		WHERE api_keys.id = $2
			AND api_keys.user_id = $1
		RETURNING id, user_id, api_key_id, group_id, endpoint, status,
			request_json, result_json, error_code, error_message,
			created_at, started_at, finished_at, expires_at, canceled_at, updated_at
	`
	return r.scanTask(ctx, query, []any{
		task.UserID,
		task.APIKeyID,
		task.Endpoint,
		task.RequestJSON,
		task.ExpiresAt,
	}, task)
}

func (r *imagePlaygroundTaskRepository) CreateTaskIfQueueAvailable(ctx context.Context, task *service.ImagePlaygroundTask, limits service.ImagePlaygroundTaskQueueLimits, now time.Time) error {
	if task == nil {
		return nil
	}
	if err := validateImagePlaygroundEndpoint(task.Endpoint); err != nil {
		return err
	}

	if beginner, ok := r.sql.(sqlTxBeginner); ok {
		tx, err := beginner.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		txRepo := newImagePlaygroundTaskRepositoryWithSQL(tx)
		if err := txRepo.createTaskIfQueueAvailableInTx(ctx, task, limits, now); err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}
	return r.createTaskIfQueueAvailableInTx(ctx, task, limits, now)
}

func (r *imagePlaygroundTaskRepository) createTaskIfQueueAvailableInTx(ctx context.Context, task *service.ImagePlaygroundTask, limits service.ImagePlaygroundTaskQueueLimits, now time.Time) error {
	if err := r.lockImagePlaygroundTaskQueue(ctx); err != nil {
		return err
	}
	counts, err := r.CountQueuedTasks(ctx, task.UserID, task.APIKeyID, now)
	if err != nil {
		return err
	}
	if limits.MaxQueuedPerUser > 0 && counts.User >= limits.MaxQueuedPerUser {
		return service.ErrImagePlaygroundTaskUserQueueFull
	}
	if limits.MaxQueuedPerAPIKey > 0 && counts.APIKey >= limits.MaxQueuedPerAPIKey {
		return service.ErrImagePlaygroundTaskAPIKeyQueueFull
	}
	if limits.MaxQueuedGlobal > 0 && counts.Global >= limits.MaxQueuedGlobal {
		return service.ErrImagePlaygroundTaskGlobalQueueFull
	}
	return r.CreateTask(ctx, task)
}

func (r *imagePlaygroundTaskRepository) lockImagePlaygroundTaskQueue(ctx context.Context) error {
	_, err := r.sql.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, imagePlaygroundTaskQueueAdvisoryLockKey)
	return err
}

func (r *imagePlaygroundTaskRepository) GetTaskByOwner(ctx context.Context, userID int64, taskID int64) (*service.ImagePlaygroundTask, error) {
	query := `
		SELECT id, user_id, api_key_id, group_id, endpoint, status,
			request_json, result_json, error_code, error_message,
			created_at, started_at, finished_at, expires_at, canceled_at, updated_at
		FROM image_playground_tasks
		WHERE id = $1 AND user_id = $2
	`
	task := &service.ImagePlaygroundTask{}
	if err := r.scanTask(ctx, query, []any{taskID, userID}, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (r *imagePlaygroundTaskRepository) ListRecentTasksByOwner(ctx context.Context, userID int64, limit int) ([]service.ImagePlaygroundTask, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, user_id, api_key_id, group_id, endpoint, status,
			request_json, result_json, error_code, error_message,
			created_at, started_at, finished_at, expires_at, canceled_at, updated_at
		FROM image_playground_tasks
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]service.ImagePlaygroundTask, 0, limit)
	for rows.Next() {
		var task service.ImagePlaygroundTask
		if err := scanImagePlaygroundTask(rows, &task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *imagePlaygroundTaskRepository) CountQueuedTasks(ctx context.Context, userID int64, apiKeyID int64, now time.Time) (service.ImagePlaygroundQueuedTaskCounts, error) {
	var counts service.ImagePlaygroundQueuedTaskCounts
	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE user_id = $1),
			COUNT(*) FILTER (WHERE api_key_id = $2),
			COUNT(*)
		FROM image_playground_tasks
		WHERE status = $3
			AND expires_at > $4
	`,
		userID,
		apiKeyID,
		service.ImagePlaygroundTaskStatusQueued,
		now,
	)
	if err != nil {
		return service.ImagePlaygroundQueuedTaskCounts{}, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return service.ImagePlaygroundQueuedTaskCounts{}, err
		}
		return counts, nil
	}
	if err := rows.Scan(&counts.User, &counts.APIKey, &counts.Global); err != nil {
		return service.ImagePlaygroundQueuedTaskCounts{}, err
	}
	if err := rows.Err(); err != nil {
		return service.ImagePlaygroundQueuedTaskCounts{}, err
	}
	return counts, nil
}

func (r *imagePlaygroundTaskRepository) CancelTask(ctx context.Context, userID int64, taskID int64, now time.Time) (bool, error) {
	query := `
		UPDATE image_playground_tasks
		SET status = $1,
			canceled_at = $2,
			finished_at = COALESCE(finished_at, $2),
			updated_at = $2
		WHERE id = $3
			AND user_id = $4
			AND status IN ($5, $6)
			AND expires_at > $2
	`
	return r.execCAS(ctx, query,
		service.ImagePlaygroundTaskStatusCanceled,
		now,
		taskID,
		userID,
		service.ImagePlaygroundTaskStatusQueued,
		service.ImagePlaygroundTaskStatusRunning,
	)
}

func (r *imagePlaygroundTaskRepository) ClaimNextQueuedTask(ctx context.Context, now time.Time) (*service.ImagePlaygroundTask, error) {
	query := `
		WITH next AS (
			SELECT id
			FROM image_playground_tasks
			WHERE status = $1
				AND expires_at > $2
			ORDER BY created_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE image_playground_tasks AS tasks
		SET status = $3,
			started_at = COALESCE(tasks.started_at, $2),
			updated_at = $2
		FROM next
		WHERE tasks.id = next.id
			AND tasks.status = $1
			AND tasks.expires_at > $2
		RETURNING tasks.id, tasks.user_id, tasks.api_key_id, tasks.group_id, tasks.endpoint, tasks.status,
			tasks.request_json, tasks.result_json, tasks.error_code, tasks.error_message,
			tasks.created_at, tasks.started_at, tasks.finished_at, tasks.expires_at, tasks.canceled_at, tasks.updated_at
	`
	task := &service.ImagePlaygroundTask{}
	if err := r.scanTask(ctx, query, []any{
		service.ImagePlaygroundTaskStatusQueued,
		now,
		service.ImagePlaygroundTaskStatusRunning,
	}, task); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

func (r *imagePlaygroundTaskRepository) MarkTaskRunning(ctx context.Context, taskID int64, now time.Time) (bool, error) {
	query := `
		UPDATE image_playground_tasks
		SET status = $1,
			started_at = COALESCE(started_at, $2),
			updated_at = $2
		WHERE id = $3
			AND status = $4
			AND expires_at > $2
	`
	return r.execCAS(ctx, query,
		service.ImagePlaygroundTaskStatusRunning,
		now,
		taskID,
		service.ImagePlaygroundTaskStatusQueued,
	)
}

func (r *imagePlaygroundTaskRepository) MarkTaskSucceeded(ctx context.Context, taskID int64, resultJSON []byte, now time.Time) (bool, error) {
	query := `
		UPDATE image_playground_tasks
		SET status = $1,
			result_json = $2,
			error_code = NULL,
			error_message = NULL,
			finished_at = $3,
			updated_at = $3
		WHERE id = $4
			AND status = $5
			AND expires_at > $3
	`
	return r.execCAS(ctx, query,
		service.ImagePlaygroundTaskStatusSucceeded,
		resultJSON,
		now,
		taskID,
		service.ImagePlaygroundTaskStatusRunning,
	)
}

func (r *imagePlaygroundTaskRepository) MarkTaskFailed(ctx context.Context, taskID int64, errorCode string, errorMessage string, now time.Time) (bool, error) {
	query := `
		UPDATE image_playground_tasks
		SET status = $1,
			error_code = $2,
			error_message = $3,
			finished_at = $4,
			updated_at = $4
		WHERE id = $5
			AND status = $6
			AND expires_at > $4
	`
	return r.execCAS(ctx, query,
		service.ImagePlaygroundTaskStatusFailed,
		strings.TrimSpace(errorCode),
		strings.TrimSpace(errorMessage),
		now,
		taskID,
		service.ImagePlaygroundTaskStatusRunning,
	)
}

func (r *imagePlaygroundTaskRepository) MarkTaskExpired(ctx context.Context, taskID int64, now time.Time) (bool, error) {
	query := `
		UPDATE image_playground_tasks
		SET status = $1,
			finished_at = COALESCE(finished_at, $2),
			updated_at = $2
		WHERE id = $3
			AND status IN ($4, $5)
			AND expires_at <= $2
	`
	return r.execCAS(ctx, query,
		service.ImagePlaygroundTaskStatusExpired,
		now,
		taskID,
		service.ImagePlaygroundTaskStatusQueued,
		service.ImagePlaygroundTaskStatusRunning,
	)
}

func (r *imagePlaygroundTaskRepository) CleanupExpiredPayloads(ctx context.Context, now time.Time, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 200
	}
	query := `
		WITH expired AS (
			SELECT id
			FROM image_playground_tasks
			WHERE expires_at <= $1
				AND (request_json <> '{}'::jsonb OR result_json IS NOT NULL)
			ORDER BY expires_at ASC, id ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE image_playground_tasks AS tasks
		SET request_json = '{}'::jsonb,
			result_json = NULL,
			status = CASE
				WHEN tasks.status IN ($3, $4) THEN $5
				ELSE tasks.status
			END,
			finished_at = CASE
				WHEN tasks.status IN ($3, $4) THEN COALESCE(tasks.finished_at, $1)
				ELSE tasks.finished_at
			END,
			updated_at = $1
		FROM expired
		WHERE tasks.id = expired.id
	`
	result, err := r.sql.ExecContext(ctx, query,
		now,
		batchSize,
		service.ImagePlaygroundTaskStatusQueued,
		service.ImagePlaygroundTaskStatusRunning,
		service.ImagePlaygroundTaskStatusExpired,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *imagePlaygroundTaskRepository) scanTask(ctx context.Context, query string, args []any, task *service.ImagePlaygroundTask) error {
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := scanImagePlaygroundTask(rows, task); err != nil {
		return err
	}
	return rows.Err()
}

func (r *imagePlaygroundTaskRepository) execCAS(ctx context.Context, query string, args ...any) (bool, error) {
	result, err := r.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func scanImagePlaygroundTask(rows interface {
	Scan(dest ...any) error
}, task *service.ImagePlaygroundTask) error {
	var (
		groupID      sql.NullInt64
		resultJSON   []byte
		errorCode    sql.NullString
		errorMessage sql.NullString
		startedAt    sql.NullTime
		finishedAt   sql.NullTime
		canceledAt   sql.NullTime
	)
	if err := rows.Scan(
		&task.ID,
		&task.UserID,
		&task.APIKeyID,
		&groupID,
		&task.Endpoint,
		&task.Status,
		&task.RequestJSON,
		&resultJSON,
		&errorCode,
		&errorMessage,
		&task.CreatedAt,
		&startedAt,
		&finishedAt,
		&task.ExpiresAt,
		&canceledAt,
		&task.UpdatedAt,
	); err != nil {
		return err
	}
	if groupID.Valid {
		v := groupID.Int64
		task.GroupID = &v
	} else {
		task.GroupID = nil
	}
	task.ResultJSON = resultJSON
	if errorCode.Valid {
		v := errorCode.String
		task.ErrorCode = &v
	} else {
		task.ErrorCode = nil
	}
	if errorMessage.Valid {
		v := errorMessage.String
		task.ErrorMessage = &v
	} else {
		task.ErrorMessage = nil
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	} else {
		task.StartedAt = nil
	}
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	} else {
		task.FinishedAt = nil
	}
	if canceledAt.Valid {
		task.CanceledAt = &canceledAt.Time
	} else {
		task.CanceledAt = nil
	}
	return nil
}

func validateImagePlaygroundEndpoint(endpoint string) error {
	switch endpoint {
	case service.ImagePlaygroundEndpointGenerations, service.ImagePlaygroundEndpointEdits:
		return nil
	default:
		return fmt.Errorf("unsupported image playground endpoint: %s", endpoint)
	}
}
