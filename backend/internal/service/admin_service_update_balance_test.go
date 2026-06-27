//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type balanceUserRepoStub struct {
	*userRepoStub
	updateErr error
	updated   []*User
}

func (s *balanceUserRepoStub) Update(ctx context.Context, user *User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if user == nil {
		return nil
	}
	clone := *user
	s.updated = append(s.updated, &clone)
	if s.userRepoStub != nil {
		s.userRepoStub.user = &clone
	}
	return nil
}

type balanceRedeemRepoStub struct {
	*redeemRepoStub
	created []*RedeemCode
}

func (s *balanceRedeemRepoStub) Create(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return nil
	}
	clone := *code
	s.created = append(s.created, &clone)
	return nil
}

type authCacheInvalidatorStub struct {
	userIDs  []int64
	groupIDs []int64
	keys     []string
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByKey(ctx context.Context, key string) {
	s.keys = append(s.keys, key)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
	s.groupIDs = append(s.groupIDs, groupID)
}

func TestAdminService_UpdateUserBalance_InvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "", nil)
	require.NoError(t, err)
	require.Equal(t, []int64{7}, invalidator.userIDs)
	require.Len(t, redeemRepo.created, 1)
}

func TestAdminService_UpdateUserBalance_NoChangeNoInvalidate(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 10, "set", "", nil)
	require.NoError(t, err)
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, redeemRepo.created)
}

func TestAdminService_UpdateUserBalance_CreatesCreditWithDefaultValidity(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	creditRepo := &redeemBalanceCreditRepoStub{}
	svc := &adminServiceImpl{
		userRepo:            repo,
		redeemCodeRepo:      redeemRepo,
		balanceCreditRepo:   creditRepo,
		settingService:      newAdminBalanceSettingService("5"),
		billingCacheService: nil,
	}

	before := time.Now().AddDate(0, 0, 5).Add(-2 * time.Second)
	_, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "", nil)
	after := time.Now().AddDate(0, 0, 5).Add(2 * time.Second)

	require.NoError(t, err)
	require.Len(t, creditRepo.created, 1)
	require.Equal(t, BalanceCreditSourceAdmin, creditRepo.created[0].SourceType)
	require.Equal(t, 5.0, creditRepo.created[0].Amount)
	require.NotNil(t, creditRepo.created[0].ExpiresAt)
	require.True(t, !creditRepo.created[0].ExpiresAt.Before(before) && !creditRepo.created[0].ExpiresAt.After(after))
}

func TestAdminService_UpdateUserBalance_CreatesPermanentCreditWithOverrideZero(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	creditRepo := &redeemBalanceCreditRepoStub{}
	svc := &adminServiceImpl{
		userRepo:            repo,
		redeemCodeRepo:      redeemRepo,
		balanceCreditRepo:   creditRepo,
		settingService:      newAdminBalanceSettingService("5"),
		billingCacheService: nil,
	}

	zero := 0
	_, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "", &zero)

	require.NoError(t, err)
	require.Len(t, creditRepo.created, 1)
	require.Nil(t, creditRepo.created[0].ExpiresAt)
}

func TestAdminService_UpdateUserBalance_SubtractDoesNotCreateCredit(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	creditRepo := &redeemBalanceCreditRepoStub{}
	svc := &adminServiceImpl{
		userRepo:            repo,
		redeemCodeRepo:      redeemRepo,
		balanceCreditRepo:   creditRepo,
		settingService:      newAdminBalanceSettingService("5"),
		billingCacheService: nil,
	}

	override := 3
	_, err := svc.UpdateUserBalance(context.Background(), 7, 2, "subtract", "", &override)

	require.NoError(t, err)
	require.Len(t, redeemRepo.created, 1)
	require.Empty(t, creditRepo.created)
}

func newAdminBalanceSettingService(validityDays string) *SettingService {
	return NewSettingService(&settingRepoStub{
		values: map[string]string{
			SettingKeyBalanceCreditValidityDays: validityDays,
		},
	}, &config.Config{
		Default: config.DefaultConfig{
			UserBalance:     0,
			UserConcurrency: 1,
		},
	})
}
