package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const vendorHallManualPauseReason = "vendor hall manual pause"

// PauseVendorHallAccount applies the fixed browser-admin pause policy while reusing
// the repository's monotonic temporary-unschedulable update and scheduler refresh.
func (s *adminServiceImpl) PauseVendorHallAccount(ctx context.Context, id int64) (time.Time, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return time.Time{}, err
	}
	if !account.Schedulable {
		return time.Time{}, infraerrors.Conflict("ACCOUNT_SCHEDULING_DISABLED", "account scheduling is disabled")
	}
	until := time.Now().Add(time.Hour)
	if err := s.accountRepo.SetTempUnschedulable(ctx, id, until, vendorHallManualPauseReason); err != nil {
		return time.Time{}, err
	}
	account, err = s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return time.Time{}, err
	}
	if !account.Schedulable {
		return time.Time{}, infraerrors.Conflict("ACCOUNT_SCHEDULING_DISABLED", "account scheduling is disabled")
	}
	if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(until) {
		return *account.TempUnschedulableUntil, nil
	}
	return until, nil
}
