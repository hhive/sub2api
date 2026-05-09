package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

type BalanceExpiryService struct {
	creditRepo           BalanceCreditRepository
	redeemRepo           RedeemCodeRepository
	entClient            *dbent.Client
	settingService       *SettingService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCacheService  *BillingCacheService
	mu                   sync.Mutex
	settlementHour       int
	lastSettlementDate   string
	stopCh               chan struct{}
	rescheduleCh         chan struct{}
	running              bool
	stopped              bool
	wg                   sync.WaitGroup
}

func NewBalanceExpiryService(
	creditRepo BalanceCreditRepository,
	redeemRepo RedeemCodeRepository,
	entClient *dbent.Client,
	settingService *SettingService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCacheService *BillingCacheService,
	settlementHour int,
) *BalanceExpiryService {
	return &BalanceExpiryService{
		creditRepo:           creditRepo,
		redeemRepo:           redeemRepo,
		entClient:            entClient,
		settingService:       settingService,
		authCacheInvalidator: authCacheInvalidator,
		billingCacheService:  billingCacheService,
		settlementHour:       settlementHour,
	}
}

func (s *BalanceExpiryService) Start() {
	if s == nil {
		return
	}
	s.startWorker(s.settlementHour)
}

func (s *BalanceExpiryService) ReloadSchedule(ctx context.Context) error {
	if s == nil || s.settingService == nil {
		return nil
	}
	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return err
	}
	if settings.BalanceCreditDailySettlementHour == nil {
		s.stopWorker()
		return nil
	}
	s.startWorker(*settings.BalanceCreditDailySettlementHour)
	return nil
}

func (s *BalanceExpiryService) startWorker(hour int) {
	if s == nil || s.creditRepo == nil || s.redeemRepo == nil || s.entClient == nil || hour < 0 || hour > 23 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.settlementHour = hour
	if s.running {
		select {
		case s.rescheduleCh <- struct{}{}:
		default:
		}
		return
	}
	s.stopCh = make(chan struct{})
	s.rescheduleCh = make(chan struct{}, 1)
	s.running = true
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			timer := time.NewTimer(durationUntilSettlementHour(time.Now(), s.currentSettlementHour()))
			select {
			case <-timer.C:
				s.runOnce()
			case <-s.rescheduleCh:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				continue
			case <-s.stopCh:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			}
		}
	}()
}

func (s *BalanceExpiryService) currentSettlementHour() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settlementHour
}

func (s *BalanceExpiryService) stopWorker() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	stopCh := s.stopCh
	s.running = false
	s.stopCh = nil
	s.rescheduleCh = nil
	s.mu.Unlock()

	close(stopCh)
	s.wg.Wait()
}

func (s *BalanceExpiryService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	s.stopWorker()
}

func durationUntilSettlementHour(now time.Time, hour int) time.Duration {
	location := now.Location()
	localNow := now.In(location)
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, 0, 0, 0, location)
	if !localNow.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(localNow)
}

func (s *BalanceExpiryService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	dayStart, dayEnd := previousSettlementWindow(now)
	settlementKey := dayStart.Format("2006-01-02")
	if s.lastSettlementDate == settlementKey {
		return
	}

	expiredUsers, err := s.settleAndExpireDay(ctx, dayStart, dayEnd, 500)
	if err != nil {
		log.Printf("[BalanceExpiry] Daily settlement failed for %s: %v", settlementKey, err)
		return
	}
	s.lastSettlementDate = settlementKey
	for _, userID := range expiredUsers {
		s.invalidateCaches(ctx, userID)
	}
}

func previousSettlementWindow(now time.Time) (time.Time, time.Time) {
	location := now.Location()
	todayStart := time.Date(now.In(location).Year(), now.In(location).Month(), now.In(location).Day(), 0, 0, 0, 0, location)
	dayStart := todayStart.AddDate(0, 0, -1)
	return dayStart, todayStart
}

func (s *BalanceExpiryService) settleAndExpireDay(ctx context.Context, dayStart time.Time, dayEnd time.Time, limit int) ([]int64, error) {
	usages, err := s.creditRepo.ListDailyBalanceUsage(ctx, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("list daily balance usage: %w", err)
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	for _, usage := range usages {
		if err := s.creditRepo.SettleDailyUsage(txCtx, usage.UserID, usage.Amount, dayStart, dayEnd); err != nil {
			return nil, fmt.Errorf("settle daily usage for user %d: %w", usage.UserID, err)
		}
	}

	touched := make(map[int64]struct{})
	for {
		expired, err := s.creditRepo.ExpireDueCredits(txCtx, dayEnd, dayStart, limit)
		if err != nil {
			return nil, err
		}
		for _, credit := range expired {
			if credit.Amount <= 0 {
				continue
			}
			if err := s.applyExpiredCredit(txCtx, tx.Client(), credit); err != nil {
				return nil, err
			}
			touched[credit.UserID] = struct{}{}
		}
		if len(expired) < limit {
			break
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit balance daily settlement: %w", err)
	}
	userIDs := make([]int64, 0, len(touched))
	for userID := range touched {
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}

func (s *BalanceExpiryService) applyExpiredCredit(ctx context.Context, client *dbent.Client, credit ExpiredBalanceCredit) error {
	if _, err := client.ExecContext(ctx, `
UPDATE users
SET balance = GREATEST(balance - $1, 0),
    updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL
`, credit.Amount, credit.UserID); err != nil {
		return fmt.Errorf("update user balance: %w", err)
	}

	code, err := GenerateRedeemCode()
	if err != nil {
		return fmt.Errorf("generate balance expired code: %w", err)
	}
	now := time.Now()
	notes := fmt.Sprintf("余额批次 %d 于 %s 过期自动清零", credit.ID, credit.ExpiresAt.Format(time.RFC3339))
	record := &RedeemCode{
		Code:   code,
		Type:   AdjustmentTypeBalanceExpired,
		Value:  -credit.Amount,
		Status: StatusUsed,
		UsedBy: &credit.UserID,
		UsedAt: &now,
		Notes:  notes,
	}
	if err := s.redeemRepo.Create(ctx, record); err != nil {
		return fmt.Errorf("create balance expired history: %w", err)
	}
	return nil
}

func (s *BalanceExpiryService) invalidateCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService == nil {
		return
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.billingCacheService.InvalidateUserBalance(cacheCtx, userID)
	}()
}

func ProvideBalanceExpiryService(
	creditRepo BalanceCreditRepository,
	redeemRepo RedeemCodeRepository,
	entClient *dbent.Client,
	settingService *SettingService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCacheService *BillingCacheService,
) *BalanceExpiryService {
	if settingService == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	settings, err := settingService.GetAllSettings(ctx)
	hour := -1
	if err == nil && settings.BalanceCreditDailySettlementHour != nil {
		hour = *settings.BalanceCreditDailySettlementHour
	}
	svc := NewBalanceExpiryService(creditRepo, redeemRepo, entClient, settingService, authCacheInvalidator, billingCacheService, hour)
	if hour >= 0 {
		svc.Start()
	}
	return svc
}
