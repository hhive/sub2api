package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestImagePlaygroundTaskRepositoryCreateTaskValidatesEndpoint(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewImagePlaygroundTaskRepository(db)
	err = repo.CreateTask(context.Background(), &service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    "/v1/chat/completions",
		RequestJSON: []byte(`{}`),
		ExpiresAt:   time.Now(),
	})
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCreateTaskDerivesGroupAndForcesQueued(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	inputGroupID := int64(99)
	derivedGroupID := int64(5)
	mock.ExpectQuery("INSERT INTO image_playground_tasks[\\s\\S]+SELECT api_keys.user_id, api_keys.id, api_keys.group_id").
		WithArgs(int64(7), int64(9), service.ImagePlaygroundEndpointGenerations, []byte(`{"prompt":"x"}`), expiresAt).
		WillReturnRows(imagePlaygroundTaskRows().
			AddRow(int64(11), int64(7), int64(9), derivedGroupID, service.ImagePlaygroundEndpointGenerations,
				service.ImagePlaygroundTaskStatusQueued, []byte(`{"prompt":"x"}`), nil, nil, nil,
				now, nil, nil, expiresAt, nil, now))

	repo := NewImagePlaygroundTaskRepository(db)
	task := &service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		GroupID:     &inputGroupID,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		Status:      service.ImagePlaygroundTaskStatusSucceeded,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   expiresAt,
	}
	err = repo.CreateTask(context.Background(), task)
	require.NoError(t, err)
	require.Equal(t, service.ImagePlaygroundTaskStatusQueued, task.Status)
	require.NotNil(t, task.GroupID)
	require.Equal(t, derivedGroupID, *task.GroupID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCreateTaskReturnsNoRowsForMismatchedOwner(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expiresAt := time.Date(2026, 5, 10, 2, 2, 3, 0, time.UTC)
	mock.ExpectQuery("FROM api_keys\\s+WHERE api_keys.id = \\$2\\s+AND api_keys.user_id = \\$1").
		WithArgs(int64(7), int64(9), service.ImagePlaygroundEndpointGenerations, []byte(`{"prompt":"x"}`), expiresAt).
		WillReturnRows(imagePlaygroundTaskRows())

	repo := NewImagePlaygroundTaskRepository(db)
	err = repo.CreateTask(context.Background(), &service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   expiresAt,
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryGetTaskByOwnerFiltersOwner(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectQuery("FROM image_playground_tasks\\s+WHERE id = \\$1 AND user_id = \\$2").
		WithArgs(int64(11), int64(7)).
		WillReturnRows(imagePlaygroundTaskRows().
			AddRow(int64(11), int64(7), int64(9), nil, service.ImagePlaygroundEndpointGenerations,
				service.ImagePlaygroundTaskStatusQueued, []byte(`{"prompt":"x"}`), nil, nil, nil,
				now, nil, nil, now.Add(time.Hour), nil, now))

	repo := NewImagePlaygroundTaskRepository(db)
	task, err := repo.GetTaskByOwner(context.Background(), 7, 11)
	require.NoError(t, err)
	require.Equal(t, int64(11), task.ID)
	require.Equal(t, int64(7), task.UserID)
	require.Nil(t, task.GroupID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryListRecentTasksByOwnerFiltersOwner(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	groupID := int64(5)
	mock.ExpectQuery("WHERE user_id = \\$1\\s+ORDER BY created_at DESC, id DESC\\s+LIMIT \\$2").
		WithArgs(int64(7), 2).
		WillReturnRows(imagePlaygroundTaskRows().
			AddRow(int64(12), int64(7), int64(9), groupID, service.ImagePlaygroundEndpointEdits,
				service.ImagePlaygroundTaskStatusSucceeded, []byte(`{"image":"in"}`), []byte(`{"data":[]}`), nil, nil,
				now, now, now, now.Add(time.Hour), nil, now))

	repo := NewImagePlaygroundTaskRepository(db)
	tasks, err := repo.ListRecentTasksByOwner(context.Background(), 7, 2)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, int64(12), tasks[0].ID)
	require.NotNil(t, tasks[0].GroupID)
	require.Equal(t, groupID, *tasks[0].GroupID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCountQueuedTasks(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectQuery("SELECT\\s+COUNT\\(\\*\\) FILTER \\(WHERE user_id = \\$1\\),\\s+COUNT\\(\\*\\) FILTER \\(WHERE api_key_id = \\$2\\),\\s+COUNT\\(\\*\\)\\s+FROM image_playground_tasks\\s+WHERE status = \\$3\\s+AND expires_at > \\$4").
		WithArgs(int64(7), int64(9), service.ImagePlaygroundTaskStatusQueued, now).
		WillReturnRows(sqlmock.NewRows([]string{"user_count", "api_key_count", "global_count"}).
			AddRow(2, 3, 4))

	repo := NewImagePlaygroundTaskRepository(db)
	counts, err := repo.CountQueuedTasks(context.Background(), 7, 9, now)
	require.NoError(t, err)
	require.Equal(t, service.ImagePlaygroundQueuedTaskCounts{User: 2, APIKey: 3, Global: 4}, counts)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCreateTaskIfQueueAvailableUsesTransactionLockAndInserts(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock\\(\\$1\\)").
		WithArgs(int64(472019050210)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT\\s+COUNT\\(\\*\\) FILTER \\(WHERE user_id = \\$1\\),\\s+COUNT\\(\\*\\) FILTER \\(WHERE api_key_id = \\$2\\),\\s+COUNT\\(\\*\\)\\s+FROM image_playground_tasks\\s+WHERE status = \\$3\\s+AND expires_at > \\$4").
		WithArgs(int64(7), int64(9), service.ImagePlaygroundTaskStatusQueued, now).
		WillReturnRows(sqlmock.NewRows([]string{"user_count", "api_key_count", "global_count"}).
			AddRow(1, 1, 1))
	mock.ExpectQuery("INSERT INTO image_playground_tasks[\\s\\S]+SELECT api_keys.user_id, api_keys.id, api_keys.group_id").
		WithArgs(int64(7), int64(9), service.ImagePlaygroundEndpointGenerations, []byte(`{"prompt":"x"}`), expiresAt).
		WillReturnRows(imagePlaygroundTaskRows().
			AddRow(int64(11), int64(7), int64(9), nil, service.ImagePlaygroundEndpointGenerations,
				service.ImagePlaygroundTaskStatusQueued, []byte(`{"prompt":"x"}`), nil, nil, nil,
				now, nil, nil, expiresAt, nil, now))
	mock.ExpectCommit()

	repo := NewImagePlaygroundTaskRepository(db)
	task := &service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   expiresAt,
	}
	err = repo.CreateTaskIfQueueAvailable(context.Background(), task, service.ImagePlaygroundTaskQueueLimits{
		MaxQueuedPerUser:   2,
		MaxQueuedPerAPIKey: 2,
		MaxQueuedGlobal:    2,
	}, now)
	require.NoError(t, err)
	require.Equal(t, int64(11), task.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCreateTaskIfQueueAvailableRollsBackWhenLimitReached(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock\\(\\$1\\)").
		WithArgs(int64(472019050210)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT\\s+COUNT\\(\\*\\) FILTER").
		WithArgs(int64(7), int64(9), service.ImagePlaygroundTaskStatusQueued, now).
		WillReturnRows(sqlmock.NewRows([]string{"user_count", "api_key_count", "global_count"}).
			AddRow(2, 1, 1))
	mock.ExpectRollback()

	repo := NewImagePlaygroundTaskRepository(db)
	err = repo.CreateTaskIfQueueAvailable(context.Background(), &service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"prompt":"x"}`),
		ExpiresAt:   now.Add(time.Hour),
	}, service.ImagePlaygroundTaskQueueLimits{
		MaxQueuedPerUser:   2,
		MaxQueuedPerAPIKey: 2,
		MaxQueuedGlobal:    2,
	}, now)
	require.ErrorIs(t, err, service.ErrImagePlaygroundTaskUserQueueFull)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryClaimNextQueuedTaskClaimsOldestUnexpired(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectQuery("WITH next AS \\(\\s+SELECT id\\s+FROM image_playground_tasks\\s+WHERE status = \\$1\\s+AND expires_at > \\$2\\s+ORDER BY created_at ASC, id ASC\\s+LIMIT 1\\s+FOR UPDATE SKIP LOCKED").
		WithArgs(service.ImagePlaygroundTaskStatusQueued, now, service.ImagePlaygroundTaskStatusRunning).
		WillReturnRows(imagePlaygroundTaskRows().
			AddRow(int64(21), int64(7), int64(9), nil, service.ImagePlaygroundEndpointGenerations,
				service.ImagePlaygroundTaskStatusRunning, []byte(`{"prompt":"oldest"}`), nil, nil, nil,
				now.Add(-time.Minute), now, nil, now.Add(time.Hour), nil, now))

	repo := NewImagePlaygroundTaskRepository(db)
	task, err := repo.ClaimNextQueuedTask(context.Background(), now)
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, int64(21), task.ID)
	require.Equal(t, service.ImagePlaygroundTaskStatusRunning, task.Status)
	require.NotNil(t, task.StartedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryClaimNextQueuedTaskReturnsNilWhenOnlyExpired(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectQuery("WHERE status = \\$1\\s+AND expires_at > \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusQueued, now, service.ImagePlaygroundTaskStatusRunning).
		WillReturnRows(imagePlaygroundTaskRows())

	repo := NewImagePlaygroundTaskRepository(db)
	task, err := repo.ClaimNextQueuedTask(context.Background(), now)
	require.NoError(t, err)
	require.Nil(t, task)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCancelTaskRequiresOwnerAndMutableStatus(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectExec("UPDATE image_playground_tasks\\s+SET status = \\$1[\\s\\S]+AND expires_at > \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusCanceled, now, int64(31), int64(7),
			service.ImagePlaygroundTaskStatusQueued, service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewImagePlaygroundTaskRepository(db)
	ok, err := repo.CancelTask(context.Background(), 7, 31, now)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCancelTaskReturnsFalseWhenNoRowsAffected(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectExec("WHERE id = \\$3\\s+AND user_id = \\$4\\s+AND status IN \\(\\$5, \\$6\\)\\s+AND expires_at > \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusCanceled, now, int64(31), int64(7),
			service.ImagePlaygroundTaskStatusQueued, service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 0))

	repo := NewImagePlaygroundTaskRepository(db)
	ok, err := repo.CancelTask(context.Background(), 7, 31, now)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryCancelTaskReturnsFalseWhenExpired(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectExec("WHERE id = \\$3\\s+AND user_id = \\$4\\s+AND status IN \\(\\$5, \\$6\\)\\s+AND expires_at > \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusCanceled, now, int64(31), int64(7),
			service.ImagePlaygroundTaskStatusQueued, service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 0))

	repo := NewImagePlaygroundTaskRepository(db)
	ok, err := repo.CancelTask(context.Background(), 7, 31, now)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryListRecentTasksByOwnerReturnsMultipleRows(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectQuery("WHERE user_id = \\$1\\s+ORDER BY created_at DESC, id DESC\\s+LIMIT \\$2").
		WithArgs(int64(7), 100).
		WillReturnRows(imagePlaygroundTaskRows().
			AddRow(int64(14), int64(7), int64(9), nil, service.ImagePlaygroundEndpointGenerations,
				service.ImagePlaygroundTaskStatusQueued, []byte(`{"prompt":"newer"}`), nil, nil, nil,
				now, nil, nil, now.Add(time.Hour), nil, now).
			AddRow(int64(13), int64(7), int64(9), nil, service.ImagePlaygroundEndpointGenerations,
				service.ImagePlaygroundTaskStatusQueued, []byte(`{"prompt":"older"}`), nil, nil, nil,
				now.Add(-time.Minute), nil, nil, now.Add(time.Hour), nil, now))

	repo := NewImagePlaygroundTaskRepository(db)
	tasks, err := repo.ListRecentTasksByOwner(context.Background(), 7, 250)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	require.Equal(t, []int64{14, 13}, []int64{tasks[0].ID, tasks[1].ID})
	for _, task := range tasks {
		require.Equal(t, int64(7), task.UserID)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryStateTransitionsUseCompareAndSwap(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectExec("WHERE id = \\$3\\s+AND status = \\$4\\s+AND expires_at > \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusRunning, now, int64(41), service.ImagePlaygroundTaskStatusQueued).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("WHERE id = \\$4\\s+AND status = \\$5\\s+AND expires_at > \\$3").
		WithArgs(service.ImagePlaygroundTaskStatusSucceeded, []byte(`{"ok":true}`), now, int64(41), service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("WHERE id = \\$5\\s+AND status = \\$6\\s+AND expires_at > \\$4").
		WithArgs(service.ImagePlaygroundTaskStatusFailed, "upstream_error", "failed", now, int64(42), service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("WHERE id = \\$3\\s+AND status IN \\(\\$4, \\$5\\)\\s+AND expires_at <= \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusExpired, now, int64(43),
			service.ImagePlaygroundTaskStatusQueued, service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewImagePlaygroundTaskRepository(db)
	ok, err := repo.MarkTaskRunning(context.Background(), 41, now)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.MarkTaskSucceeded(context.Background(), 41, []byte(`{"ok":true}`), now)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.MarkTaskFailed(context.Background(), 42, " upstream_error ", " failed ", now)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.MarkTaskExpired(context.Background(), 43, now)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryStateTransitionsReturnFalseWhenCompareAndSwapMisses(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	mock.ExpectExec("WHERE id = \\$3\\s+AND status = \\$4\\s+AND expires_at > \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusRunning, now, int64(41), service.ImagePlaygroundTaskStatusQueued).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("WHERE id = \\$4\\s+AND status = \\$5\\s+AND expires_at > \\$3").
		WithArgs(service.ImagePlaygroundTaskStatusSucceeded, []byte(`{"ok":true}`), now, int64(41), service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("WHERE id = \\$5\\s+AND status = \\$6\\s+AND expires_at > \\$4").
		WithArgs(service.ImagePlaygroundTaskStatusFailed, "upstream_error", "failed", now, int64(42), service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("WHERE id = \\$3\\s+AND status IN \\(\\$4, \\$5\\)\\s+AND expires_at <= \\$2").
		WithArgs(service.ImagePlaygroundTaskStatusExpired, now, int64(43),
			service.ImagePlaygroundTaskStatusQueued, service.ImagePlaygroundTaskStatusRunning).
		WillReturnResult(sqlmock.NewResult(0, 0))

	repo := NewImagePlaygroundTaskRepository(db)
	ok, err := repo.MarkTaskRunning(context.Background(), 41, now)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = repo.MarkTaskSucceeded(context.Background(), 41, []byte(`{"ok":true}`), now)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = repo.MarkTaskFailed(context.Background(), 42, " upstream_error ", " failed ", now)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = repo.MarkTaskExpired(context.Background(), 43, now)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImagePlaygroundTaskRepositoryGetTaskByOwnerNotFound(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("WHERE id = \\$1 AND user_id = \\$2").
		WithArgs(int64(11), int64(7)).
		WillReturnRows(imagePlaygroundTaskRows())

	repo := NewImagePlaygroundTaskRepository(db)
	task, err := repo.GetTaskByOwner(context.Background(), 7, 11)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Nil(t, task)
	require.NoError(t, mock.ExpectationsWereMet())
}

func imagePlaygroundTaskRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"user_id",
		"api_key_id",
		"group_id",
		"endpoint",
		"status",
		"request_json",
		"result_json",
		"error_code",
		"error_message",
		"created_at",
		"started_at",
		"finished_at",
		"expires_at",
		"canceled_at",
		"updated_at",
	})
}
