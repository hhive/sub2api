package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type relayMonitorTempUnschedRepo struct {
	AccountRepository
	account   *Account
	setCalls  int
	setID     int64
	setUntil  time.Time
	setReason string
}

func (r *relayMonitorTempUnschedRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.account == nil || r.account.ID != id {
		return nil, ErrAccountNotFound
	}
	copy := *r.account
	return &copy, nil
}

func (r *relayMonitorTempUnschedRepo) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.setCalls++
	r.setID = id
	r.setUntil = until
	r.setReason = reason
	if r.account.TempUnschedulableUntil == nil || r.account.TempUnschedulableUntil.Before(until) {
		r.account.TempUnschedulableUntil = &until
		r.account.TempUnschedulableReason = reason
	}
	return nil
}

func TestPauseRelayMonitorPriorityCappedAccountUsesFixedWindowAndReason(t *testing.T) {
	repo := &relayMonitorTempUnschedRepo{account: &Account{ID: 7}}
	svc := &adminServiceImpl{accountRepo: repo}
	before := time.Now()

	actualUntil, err := svc.PauseRelayMonitorPriorityCappedAccount(context.Background(), 7, 12*time.Minute)

	after := time.Now()
	require.NoError(t, err)
	require.Equal(t, 1, repo.setCalls)
	require.Equal(t, int64(7), repo.setID)
	require.Equal(t, relayMonitorPriorityCappedReason, repo.setReason)
	require.False(t, repo.setUntil.Before(before.Add(12*time.Minute)))
	require.False(t, repo.setUntil.After(after.Add(12*time.Minute)))
	require.Equal(t, repo.setUntil, actualUntil)
}

func TestPauseRelayMonitorPriorityCappedAccountDoesNotShortenExistingWindow(t *testing.T) {
	existingUntil := time.Now().Add(5 * time.Minute).UTC()
	repo := &relayMonitorTempUnschedRepo{account: &Account{
		ID:                      7,
		TempUnschedulableUntil:  &existingUntil,
		TempUnschedulableReason: "existing reason",
	}}
	svc := &adminServiceImpl{accountRepo: repo}

	actualUntil, err := svc.PauseRelayMonitorPriorityCappedAccount(context.Background(), 7, time.Minute)

	require.NoError(t, err)
	require.Equal(t, 1, repo.setCalls)
	require.Equal(t, existingUntil, actualUntil)
	require.Equal(t, existingUntil, *repo.account.TempUnschedulableUntil)
	require.Equal(t, "existing reason", repo.account.TempUnschedulableReason)
}

func TestPauseRelayMonitorPriorityCappedAccountRejectsDurationOutsideBoundary(t *testing.T) {
	repo := &relayMonitorTempUnschedRepo{account: &Account{ID: 7}}
	svc := &adminServiceImpl{accountRepo: repo}
	for _, duration := range []time.Duration{0, 59 * time.Second, 61 * time.Minute} {
		_, err := svc.PauseRelayMonitorPriorityCappedAccount(context.Background(), 7, duration)
		require.Error(t, err, duration)
	}
	require.Zero(t, repo.setCalls)
}
