//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestImagePlaygroundTaskRepositoryIntegrationCreateDerivesAPIKeyOwnerAndGroup(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newImagePlaygroundTaskRepositoryWithSQL(tx)

	userID, apiKeyID := insertImagePlaygroundTaskOwner(t, ctx, tx, "create-owner")
	otherUserID, _ := insertImagePlaygroundTaskOwner(t, ctx, tx, "create-other")
	wrongGroupID := insertImagePlaygroundGroup(t, ctx, tx, "create-wrong")
	derivedGroupID := insertImagePlaygroundGroup(t, ctx, tx, "create-derived")
	require.NoError(t, tx.QueryRowContext(ctx, `
		UPDATE api_keys
		SET group_id = $1
		WHERE id = $2
		RETURNING id
	`, derivedGroupID, apiKeyID).Scan(&apiKeyID))

	expiresAt := time.Date(2026, 5, 10, 2, 2, 3, 0, time.UTC)
	task := &service.ImagePlaygroundTask{
		UserID:      userID,
		APIKeyID:    apiKeyID,
		GroupID:     &wrongGroupID,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		Status:      service.ImagePlaygroundTaskStatusSucceeded,
		RequestJSON: []byte(`{"prompt":"test"}`),
		ExpiresAt:   expiresAt,
	}
	require.NoError(t, repo.CreateTask(ctx, task))
	require.NotZero(t, task.ID)
	require.Equal(t, service.ImagePlaygroundTaskStatusQueued, task.Status)
	require.NotNil(t, task.GroupID)
	require.Equal(t, derivedGroupID, *task.GroupID)

	mismatched := &service.ImagePlaygroundTask{
		UserID:      otherUserID,
		APIKeyID:    apiKeyID,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"prompt":"test"}`),
		ExpiresAt:   expiresAt,
	}
	require.ErrorIs(t, repo.CreateTask(ctx, mismatched), sql.ErrNoRows)

	_, err := tx.ExecContext(ctx, `
		INSERT INTO image_playground_tasks (
			user_id, api_key_id, endpoint, status, request_json, expires_at
		)
		VALUES ($1, $2, $3, $4, '{"prompt":"mismatch"}'::jsonb, $5)
	`, otherUserID, apiKeyID, service.ImagePlaygroundEndpointGenerations, service.ImagePlaygroundTaskStatusQueued, expiresAt)
	require.Error(t, err)
}

func TestImagePlaygroundTaskRepositoryIntegrationOwnerFilteringAndRecentList(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newImagePlaygroundTaskRepositoryWithSQL(tx)

	ownerUserID, ownerAPIKeyID := insertImagePlaygroundTaskOwner(t, ctx, tx, "owner-filter-a")
	otherUserID, otherAPIKeyID := insertImagePlaygroundTaskOwner(t, ctx, tx, "owner-filter-b")
	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)

	olderOwnerTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    ownerUserID,
		APIKeyID:  ownerAPIKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusQueued,
		CreatedAt: now.Add(-2 * time.Minute),
		ExpiresAt: now.Add(time.Hour),
	})
	newerOwnerTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    ownerUserID,
		APIKeyID:  ownerAPIKeyID,
		Endpoint:  service.ImagePlaygroundEndpointEdits,
		Status:    service.ImagePlaygroundTaskStatusSucceeded,
		CreatedAt: now.Add(-time.Minute),
		ExpiresAt: now.Add(time.Hour),
	})
	otherTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    otherUserID,
		APIKeyID:  otherAPIKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusQueued,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	})

	gotOther, err := repo.GetTaskByOwner(ctx, ownerUserID, otherTaskID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Nil(t, gotOther)

	tasks, err := repo.ListRecentTasksByOwner(ctx, ownerUserID, 10)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	require.Equal(t, newerOwnerTaskID, tasks[0].ID)
	require.Equal(t, olderOwnerTaskID, tasks[1].ID)
	for _, task := range tasks {
		require.Equal(t, ownerUserID, task.UserID)
	}
}

func TestImagePlaygroundTaskRepositoryIntegrationClaimSkipsExpiredAndClaimsOldestQueued(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newImagePlaygroundTaskRepositoryWithSQL(tx)

	userID, apiKeyID := insertImagePlaygroundTaskOwner(t, ctx, tx, "claim-order")
	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)

	expiredTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusQueued,
		CreatedAt: now.Add(-3 * time.Minute),
		ExpiresAt: now.Add(-time.Second),
	})
	firstValidTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusQueued,
		CreatedAt: now.Add(-2 * time.Minute),
		ExpiresAt: now.Add(time.Hour),
	})
	secondValidTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusQueued,
		CreatedAt: now.Add(-time.Minute),
		ExpiresAt: now.Add(time.Hour),
	})

	firstClaim, err := repo.ClaimNextQueuedTask(ctx, now)
	require.NoError(t, err)
	require.NotNil(t, firstClaim)
	require.Equal(t, firstValidTaskID, firstClaim.ID)
	require.Equal(t, service.ImagePlaygroundTaskStatusRunning, firstClaim.Status)

	secondClaim, err := repo.ClaimNextQueuedTask(ctx, now)
	require.NoError(t, err)
	require.NotNil(t, secondClaim)
	require.Equal(t, secondValidTaskID, secondClaim.ID)

	thirdClaim, err := repo.ClaimNextQueuedTask(ctx, now)
	require.NoError(t, err)
	require.Nil(t, thirdClaim)

	var expiredStatus string
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT status FROM image_playground_tasks WHERE id = $1", expiredTaskID).Scan(&expiredStatus))
	require.Equal(t, service.ImagePlaygroundTaskStatusQueued, expiredStatus)
}

func TestImagePlaygroundTaskRepositoryIntegrationCancelAndCASBehavior(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newImagePlaygroundTaskRepositoryWithSQL(tx)

	userID, apiKeyID := insertImagePlaygroundTaskOwner(t, ctx, tx, "cancel-cas-a")
	otherUserID, otherAPIKeyID := insertImagePlaygroundTaskOwner(t, ctx, tx, "cancel-cas-b")
	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)

	queuedTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusQueued,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	})
	succeededTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Endpoint:   service.ImagePlaygroundEndpointGenerations,
		Status:     service.ImagePlaygroundTaskStatusSucceeded,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Hour),
		FinishedAt: &now,
	})
	otherTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    otherUserID,
		APIKeyID:  otherAPIKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusQueued,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	})
	expiredTaskID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusRunning,
		CreatedAt: now,
		ExpiresAt: now.Add(-time.Second),
	})

	ok, err := repo.CancelTask(ctx, userID, otherTaskID, now)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusQueued, imagePlaygroundTaskStatus(t, ctx, tx, otherTaskID))

	ok, err = repo.CancelTask(ctx, userID, succeededTaskID, now)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusSucceeded, imagePlaygroundTaskStatus(t, ctx, tx, succeededTaskID))

	ok, err = repo.CancelTask(ctx, userID, queuedTaskID, now)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusCanceled, imagePlaygroundTaskStatus(t, ctx, tx, queuedTaskID))
	require.True(t, imagePlaygroundTaskHasCanceledAt(t, ctx, tx, queuedTaskID))

	ok, err = repo.CancelTask(ctx, userID, expiredTaskID, now)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusRunning, imagePlaygroundTaskStatus(t, ctx, tx, expiredTaskID))

	ok, err = repo.MarkTaskRunning(ctx, queuedTaskID, now)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = repo.MarkTaskSucceeded(ctx, succeededTaskID, []byte(`{"data":[]}`), now)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = repo.MarkTaskExpired(ctx, succeededTaskID, now)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = repo.MarkTaskExpired(ctx, expiredTaskID, now)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusExpired, imagePlaygroundTaskStatus(t, ctx, tx, expiredTaskID))
}

func TestImagePlaygroundTaskRepositoryIntegrationExpiredRunningCannotSucceedOrFail(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newImagePlaygroundTaskRepositoryWithSQL(tx)

	userID, apiKeyID := insertImagePlaygroundTaskOwner(t, ctx, tx, "expired-running")
	now := time.Date(2026, 5, 10, 1, 2, 3, 0, time.UTC)
	expiredSucceededCandidateID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusRunning,
		CreatedAt: now.Add(-time.Hour),
		ExpiresAt: now.Add(-time.Second),
		StartedAt: &now,
	})
	expiredFailedCandidateID := insertImagePlaygroundTaskRow(t, ctx, tx, imagePlaygroundTaskSeed{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Endpoint:  service.ImagePlaygroundEndpointGenerations,
		Status:    service.ImagePlaygroundTaskStatusRunning,
		CreatedAt: now.Add(-time.Hour),
		ExpiresAt: now.Add(-time.Second),
		StartedAt: &now,
	})

	ok, err := repo.MarkTaskSucceeded(ctx, expiredSucceededCandidateID, []byte(`{"data":[]}`), now)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusRunning, imagePlaygroundTaskStatus(t, ctx, tx, expiredSucceededCandidateID))

	ok, err = repo.MarkTaskFailed(ctx, expiredFailedCandidateID, "expired", "expired", now)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusRunning, imagePlaygroundTaskStatus(t, ctx, tx, expiredFailedCandidateID))

	ok, err = repo.MarkTaskExpired(ctx, expiredSucceededCandidateID, now)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, service.ImagePlaygroundTaskStatusExpired, imagePlaygroundTaskStatus(t, ctx, tx, expiredSucceededCandidateID))
}

type imagePlaygroundTaskSeed struct {
	UserID     int64
	APIKeyID   int64
	Endpoint   string
	Status     string
	CreatedAt  time.Time
	StartedAt  *time.Time
	ExpiresAt  time.Time
	FinishedAt *time.Time
}

func insertImagePlaygroundTaskOwner(t *testing.T, ctx context.Context, tx *sql.Tx, suffix string) (int64, int64) {
	t.Helper()

	unique := fmt.Sprintf("%s-%d", suffix, time.Now().UnixNano())
	var userID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, balance, concurrency)
		VALUES ($1, 'hash', 'user', 'active', 0, 5)
		RETURNING id
	`, unique+"@example.test").Scan(&userID))

	var apiKeyID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO api_keys (user_id, key, name, status)
		VALUES ($1, $2, 'Image Playground Test Key', 'active')
		RETURNING id
	`, userID, "sk-test-"+unique).Scan(&apiKeyID))

	return userID, apiKeyID
}

func insertImagePlaygroundGroup(t *testing.T, ctx context.Context, tx *sql.Tx, suffix string) int64 {
	t.Helper()

	unique := fmt.Sprintf("%s-%d", suffix, time.Now().UnixNano())
	var groupID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO groups (name, platform, rate_multiplier, is_exclusive, status, subscription_type)
		VALUES ($1, 'openai', 1.0, false, 'active', 'standard')
		RETURNING id
	`, "image-playground-"+unique).Scan(&groupID))
	return groupID
}

func insertImagePlaygroundTaskRow(t *testing.T, ctx context.Context, tx *sql.Tx, seed imagePlaygroundTaskSeed) int64 {
	t.Helper()

	var taskID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO image_playground_tasks (
			user_id, api_key_id, endpoint, status, request_json,
			created_at, started_at, expires_at, finished_at, updated_at
		)
		VALUES ($1, $2, $3, $4, '{"prompt":"test"}'::jsonb, $5, $6, $7, $8, $5)
		RETURNING id
	`, seed.UserID, seed.APIKeyID, seed.Endpoint, seed.Status, seed.CreatedAt, seed.StartedAt, seed.ExpiresAt, seed.FinishedAt).Scan(&taskID))
	return taskID
}

func imagePlaygroundTaskStatus(t *testing.T, ctx context.Context, tx *sql.Tx, taskID int64) string {
	t.Helper()

	var status string
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT status FROM image_playground_tasks WHERE id = $1", taskID).Scan(&status))
	return status
}

func imagePlaygroundTaskHasCanceledAt(t *testing.T, ctx context.Context, tx *sql.Tx, taskID int64) bool {
	t.Helper()

	var canceledAt sql.NullTime
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT canceled_at FROM image_playground_tasks WHERE id = $1", taskID).Scan(&canceledAt))
	return canceledAt.Valid
}
