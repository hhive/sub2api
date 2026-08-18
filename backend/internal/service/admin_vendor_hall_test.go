package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type disablingAfterPauseRepo struct{ *relayMonitorTempUnschedRepo }

func (r *disablingAfterPauseRepo) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	if err := r.relayMonitorTempUnschedRepo.SetTempUnschedulable(ctx, id, until, reason); err != nil {
		return err
	}
	r.account.Schedulable = false
	return nil
}

func TestPauseVendorHallAccountUsesFixedHourAndManualReason(t *testing.T) {
	repo := &relayMonitorTempUnschedRepo{account: &Account{ID: 7, Schedulable: true}}
	svc := &adminServiceImpl{accountRepo: repo}
	before := time.Now()

	until, err := svc.PauseVendorHallAccount(context.Background(), 7)

	after := time.Now()
	require.NoError(t, err)
	require.Equal(t, 1, repo.setCalls)
	require.Equal(t, vendorHallManualPauseReason, repo.setReason)
	require.False(t, until.Before(before.Add(time.Hour)))
	require.False(t, until.After(after.Add(time.Hour)))
}

func TestPauseVendorHallAccountDoesNotShortenExistingWindow(t *testing.T) {
	existing := time.Now().Add(2 * time.Hour).UTC()
	repo := &relayMonitorTempUnschedRepo{account: &Account{ID: 7, Schedulable: true, TempUnschedulableUntil: &existing}}
	svc := &adminServiceImpl{accountRepo: repo}

	until, err := svc.PauseVendorHallAccount(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, existing, until)
}

func TestPauseVendorHallAccountReturnsNotFound(t *testing.T) {
	repo := &relayMonitorTempUnschedRepo{account: &Account{ID: 1, Schedulable: true}}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.PauseVendorHallAccount(context.Background(), 404)

	require.ErrorIs(t, err, ErrAccountNotFound)
}

func TestPauseVendorHallAccountRejectsPermanentlyDisabledAccount(t *testing.T) {
	repo := &relayMonitorTempUnschedRepo{account: &Account{ID: 7, Schedulable: false}}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.PauseVendorHallAccount(context.Background(), 7)

	require.Error(t, err)
	require.Equal(t, 0, repo.setCalls)
}

func TestPauseVendorHallAccountReportsConcurrentPermanentDisable(t *testing.T) {
	base := &relayMonitorTempUnschedRepo{account: &Account{ID: 7, Schedulable: true}}
	svc := &adminServiceImpl{accountRepo: &disablingAfterPauseRepo{relayMonitorTempUnschedRepo: base}}

	_, err := svc.PauseVendorHallAccount(context.Background(), 7)

	require.Error(t, err)
	require.Equal(t, 1, base.setCalls)
}
