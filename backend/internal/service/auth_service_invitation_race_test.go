//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// raceSafeUserRepo 是仅覆盖注册路径的并发安全用户仓储桩。
// 未用到的方法走嵌入接口（调用即 panic，注册路径不会触发）。
type raceSafeUserRepo struct {
	UserRepository

	mu      sync.Mutex
	nextID  int64
	byEmail map[string]*User
	byID    map[int64]*User
}

func newRaceSafeUserRepo() *raceSafeUserRepo {
	return &raceSafeUserRepo{nextID: 1, byEmail: map[string]*User{}, byID: map[int64]*User{}}
}

func (s *raceSafeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.byEmail[email]
	return ok, nil
}

func (s *raceSafeUserRepo) ExistsByEmailAlias(ctx context.Context, email string) (bool, error) {
	return s.ExistsByEmail(ctx, email)
}

func (s *raceSafeUserRepo) CreateWithEmailAliasGuard(_ context.Context, user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byEmail[user.Email]; ok {
		return ErrEmailExists
	}
	user.ID = s.nextID
	s.nextID++
	clone := *user
	s.byEmail[user.Email] = &clone
	s.byID[user.ID] = &clone
	return nil
}

func (s *raceSafeUserRepo) GetByEmail(_ context.Context, email string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}

func (s *raceSafeUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}

func (s *raceSafeUserRepo) Update(context.Context, *User, UserUpdateFields) error {
	return nil
}

// raceSafeRedeemRepo 是并发安全的兑换码仓储桩：Use 以互斥锁 + 状态条件
// 模拟数据库的条件更新（WHERE status='unused'），语义与线上实现一致。
type raceSafeRedeemRepo struct {
	RedeemCodeRepository

	mu    sync.Mutex
	codes map[string]*RedeemCode
}

func (s *raceSafeRedeemRepo) GetByCode(_ context.Context, code string) (*RedeemCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return nil, ErrRedeemCodeNotFound
	}
	clone := *c
	return &clone, nil
}

func (s *raceSafeRedeemRepo) Use(_ context.Context, id, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.codes {
		if c.ID != id {
			continue
		}
		if c.Status != StatusUnused {
			return ErrRedeemCodeUsed
		}
		now := time.Now().UTC()
		c.Status = StatusUsed
		c.UsedBy = &userID
		c.UsedAt = &now
		return nil
	}
	return ErrRedeemCodeNotFound
}

// invitationAffiliateRepoStub 覆盖返利绑定所需的仓储方法，其余走嵌入接口。
type invitationAffiliateRepoStub struct {
	AffiliateRepository

	mu       sync.Mutex
	profiles map[int64]*AffiliateSummary
	byCode   map[string]*AffiliateSummary
	lookups  []string
	binds    []struct {
		userID    int64
		inviterID int64
	}
}

func (s *invitationAffiliateRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[userID]
	if !ok {
		profile = &AffiliateSummary{UserID: userID, AffCode: fmt.Sprintf("SELF%d", userID)}
		s.profiles[userID] = profile
	}
	cloned := *profile
	return &cloned, nil
}

func (s *invitationAffiliateRepoStub) GetAffiliateByCode(_ context.Context, code string) (*AffiliateSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lookups = append(s.lookups, code)
	profile, ok := s.byCode[code]
	if !ok {
		return nil, ErrAffiliateProfileNotFound
	}
	cloned := *profile
	return &cloned, nil
}

func (s *invitationAffiliateRepoStub) BindInviter(_ context.Context, userID, inviterID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[userID]
	if !ok {
		profile = &AffiliateSummary{UserID: userID}
		s.profiles[userID] = profile
	}
	if profile.InviterID != nil {
		return false, nil
	}
	profile.InviterID = &inviterID
	s.binds = append(s.binds, struct {
		userID    int64
		inviterID int64
	}{userID: userID, inviterID: inviterID})
	return true, nil
}

// TestCreateEmailOAuthUserBindsAffiliateFromInvitationField OAuth 首次注册同样接受返利码。
func TestCreateEmailOAuthUserBindsAffiliateFromInvitationField(t *testing.T) {
	const affiliateCode = "4W52TTV5G5A3"
	const inviterUserID int64 = 9002
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
		"affiliate_enabled":       "true",
	}
	svc := newOAuthEmailFlowAuthService(
		&userRepoStub{nextID: 501, usersByEmail: map[string]*User{}},
		&redeemCodeRepoStub{codesByCode: map[string]*RedeemCode{}},
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)
	inviter := &AffiliateSummary{UserID: inviterUserID, AffCode: affiliateCode}
	affiliateRepo := &invitationAffiliateRepoStub{
		profiles: map[int64]*AffiliateSummary{inviterUserID: inviter},
		byCode:   map[string]*AffiliateSummary{affiliateCode: inviter},
	}
	svc.affiliateService = NewAffiliateService(affiliateRepo, svc.settingService, nil, nil, nil)

	user, err := svc.createEmailOAuthUser(context.Background(), "oauth-aff@example.com", "oauth", "wechat", affiliateCode, "")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, affiliateRepo.binds, 1, "OAuth 首次注册时邀请码字段里的返利码必须绑定邀请人")
	require.Equal(t, inviterUserID, affiliateRepo.binds[0].inviterID)
}

// TestAuthService_Register_InvitationCodeSingleUseUnderConcurrency 回归测试：
// 同一邀请码并发注册必须恰好成功 1 次，其余请求以 INVITATION_CODE_INVALID 拒绝。
//
// 修复前：邀请码“检查(CanUse) 与 标记已用(Use)”分离且不在同一事务，Use 失败被吞，
// 并发请求全部注册成功（一个邀请码可创建任意数量账号）。此测试在该实现下必然失败。
// 修复后：用户创建与邀请码占用在同一事务内原子完成（或退化路径下由 Use 条件更新
// 兜底），并发下仅最先占码的注册成功。
func TestAuthService_Register_InvitationCodeSingleUseUnderConcurrency(t *testing.T) {
	const code = "INV-RACE-001"
	userRepo := newRaceSafeUserRepo()
	redeemRepo := &raceSafeRedeemRepo{codes: map[string]*RedeemCode{
		code: {ID: 1, Code: code, Type: RedeemTypeInvitation, Status: StatusUnused},
	}}
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
	}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		settings,
		nil, // emailCache：注册不要求邮箱验证，保持关闭
		&userPlatformQuotaRepoStub{},
	)

	const n = 8
	ctx := context.Background()
	start := make(chan struct{})
	results := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			email := fmt.Sprintf("race-%d@example.com", i)
			_, _, err := svc.RegisterWithVerification(ctx, email, "Password123!", "", "", code, "")
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	rejected := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInvitationCodeInvalid):
			rejected++
		default:
			t.Fatalf("unexpected registration error: %v", err)
		}
	}
	require.Equal(t, 1, successes, "同一邀请码并发注册必须恰好成功 1 次")
	require.Equal(t, n-1, rejected, "其余并发请求必须以 INVITATION_CODE_INVALID 拒绝")

	claimed, err := redeemRepo.GetByCode(ctx, code)
	require.NoError(t, err)
	require.Equal(t, StatusUsed, claimed.Status, "邀请码最终必须处于 used 状态")
	require.NotNil(t, claimed.UsedBy, "used_by 必须记录实际注册用户")
}

// TestAuthService_Register_InvitationCodeAlreadyUsedDoesNotBlock 顺序路径：
// 已使用过的邀请型兑换码再次出现在邀请码字段时不阻断注册，也不会被重复消费。
func TestAuthService_Register_InvitationCodeAlreadyUsedDoesNotBlock(t *testing.T) {
	const code = "INV-RACE-002"
	userRepo := newRaceSafeUserRepo()
	redeemRepo := &raceSafeRedeemRepo{codes: map[string]*RedeemCode{
		code: {ID: 2, Code: code, Type: RedeemTypeInvitation, Status: StatusUsed},
	}}
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
	}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)

	_, user, err := svc.RegisterWithVerification(context.Background(), "later@example.com", "Password123!", "", "", code, "")
	require.NoError(t, err)
	require.NotNil(t, user)

	existing, err := redeemRepo.GetByCode(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, StatusUsed, existing.Status, "已使用的邀请码不得被重复消费")
}

// TestAuthService_Register_InvitationCodeExpiredDoesNotBlock 过期邀请码不阻断注册，也不会被消费。
func TestAuthService_Register_InvitationCodeExpiredDoesNotBlock(t *testing.T) {
	const code = "INV-EXPIRED-001"
	expiredAt := time.Now().Add(-time.Hour)
	userRepo := newRaceSafeUserRepo()
	redeemRepo := &raceSafeRedeemRepo{codes: map[string]*RedeemCode{
		code: {ID: 3, Code: code, Type: RedeemTypeInvitation, Status: StatusUnused, ExpiresAt: &expiredAt},
	}}
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
	}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)

	_, user, err := svc.RegisterWithVerification(context.Background(), "expired-invite@example.com", "Password123!", "", "", code, "")
	require.NoError(t, err)
	require.NotNil(t, user)

	existing, err := redeemRepo.GetByCode(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, StatusUnused, existing.Status, "过期邀请码不得被消费")
}

// TestAuthService_Register_AffiliateCodeInInvitationFieldBindsInviter 回归：
// 注册页邀请码字段里填好友返利码时必须注册成功并绑定邀请人。
func TestAuthService_Register_AffiliateCodeInInvitationFieldBindsInviter(t *testing.T) {
	const affiliateCode = "4W52TTV5G5A3"
	const inviterUserID int64 = 9001
	userRepo := newRaceSafeUserRepo()
	redeemRepo := &raceSafeRedeemRepo{codes: map[string]*RedeemCode{}}
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
		"affiliate_enabled":       "true",
	}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)
	inviter := &AffiliateSummary{UserID: inviterUserID, AffCode: affiliateCode}
	affiliateRepo := &invitationAffiliateRepoStub{
		profiles: map[int64]*AffiliateSummary{inviterUserID: inviter},
		byCode:   map[string]*AffiliateSummary{affiliateCode: inviter},
	}
	svc.affiliateService = NewAffiliateService(affiliateRepo, svc.settingService, nil, nil, nil)

	_, user, err := svc.RegisterWithVerification(context.Background(), "manual-aff@example.com", "Password123!", "", "", affiliateCode, "")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, affiliateRepo.binds, 1, "邀请码字段里的返利码必须绑定邀请人")
	require.Equal(t, user.ID, affiliateRepo.binds[0].userID)
	require.Equal(t, inviterUserID, affiliateRepo.binds[0].inviterID)
}

// TestAuthService_Register_UnknownInvitationCodeDoesNotBlock 未知字符串不阻断注册，
// 且会作为返利邀请码候选透传（查找失败只记日志）。
func TestAuthService_Register_UnknownInvitationCodeDoesNotBlock(t *testing.T) {
	const unknownCode = "AAAA2222BBBB"
	userRepo := newRaceSafeUserRepo()
	redeemRepo := &raceSafeRedeemRepo{codes: map[string]*RedeemCode{}}
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
		"affiliate_enabled":       "true",
	}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)
	affiliateRepo := &invitationAffiliateRepoStub{
		profiles: map[int64]*AffiliateSummary{},
		byCode:   map[string]*AffiliateSummary{},
	}
	svc.affiliateService = NewAffiliateService(affiliateRepo, svc.settingService, nil, nil, nil)

	_, user, err := svc.RegisterWithVerification(context.Background(), "unknown-code@example.com", "Password123!", "", "", unknownCode, "")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, []string{unknownCode}, affiliateRepo.lookups, "未知码必须作为返利邀请码候选透传")
	require.Empty(t, affiliateRepo.binds, "无法识别的码不得产生邀请绑定")
}

// TestAuthService_OAuthCompleteRegistration_AffiliateCodeInInvitationField 覆盖
// Linux.do / OIDC / 钉钉 / 微信 complete-registration 共用路径。
func TestAuthService_OAuthCompleteRegistration_AffiliateCodeInInvitationField(t *testing.T) {
	const affiliateCode = "4W52TTV5G5A3"
	const inviterUserID int64 = 9003
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
		"affiliate_enabled":       "true",
	}
	svc := newOAuthEmailFlowAuthService(
		&userRepoStub{nextID: 601, usersByEmail: map[string]*User{}},
		&raceSafeRedeemRepo{codes: map[string]*RedeemCode{}},
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)
	inviter := &AffiliateSummary{UserID: inviterUserID, AffCode: affiliateCode}
	affiliateRepo := &invitationAffiliateRepoStub{
		profiles: map[int64]*AffiliateSummary{inviterUserID: inviter},
		byCode:   map[string]*AffiliateSummary{affiliateCode: inviter},
	}
	svc.affiliateService = NewAffiliateService(affiliateRepo, svc.settingService, nil, nil, nil)

	_, user, err := svc.LoginOrRegisterOAuthWithTokenPair(context.Background(), "oauth-complete@example.com", "oauth", affiliateCode, "", "wechat")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, affiliateRepo.binds, 1, "complete-registration 路径必须接受邀请码字段里的返利码")
	require.Equal(t, inviterUserID, affiliateRepo.binds[0].inviterID)
}

// TestFinalizeOAuthEmailAccount_BindsAffiliateFromInvitationField 覆盖 pending create-account 收尾路径。
func TestFinalizeOAuthEmailAccount_BindsAffiliateFromInvitationField(t *testing.T) {
	const affiliateCode = "4W52TTV5G5A3"
	const inviterUserID int64 = 9004
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
		"affiliate_enabled":       "true",
	}
	svc := newOAuthEmailFlowAuthService(
		newRaceSafeUserRepo(),
		&raceSafeRedeemRepo{codes: map[string]*RedeemCode{}},
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)
	inviter := &AffiliateSummary{UserID: inviterUserID, AffCode: affiliateCode}
	affiliateRepo := &invitationAffiliateRepoStub{
		profiles: map[int64]*AffiliateSummary{inviterUserID: inviter},
		byCode:   map[string]*AffiliateSummary{affiliateCode: inviter},
	}
	svc.affiliateService = NewAffiliateService(affiliateRepo, svc.settingService, nil, nil, nil)

	user := &User{ID: 777, Email: "pending-aff@example.com", Status: StatusActive}
	require.NoError(t, svc.FinalizeOAuthEmailAccount(context.Background(), user, affiliateCode, "wechat", ""))
	require.Len(t, affiliateRepo.binds, 1, "pending create-account 收尾时必须绑定返利邀请人")
	require.Equal(t, inviterUserID, affiliateRepo.binds[0].inviterID)
}

// TestAuthService_Register_InvitationCodeOptionalWhenEnabled 保留本地产品契约：
// 开启邀请码入口后仍允许不填写；只有提供的邀请码才参与原子校验与占用。
func TestAuthService_Register_InvitationCodeOptionalWhenEnabled(t *testing.T) {
	userRepo := newRaceSafeUserRepo()
	redeemRepo := &raceSafeRedeemRepo{codes: map[string]*RedeemCode{}}
	settings := map[string]string{
		"registration_enabled":    "true",
		"invitation_code_enabled": "true",
	}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		settings,
		nil,
		&userPlatformQuotaRepoStub{},
	)

	token, user, err := svc.RegisterWithVerification(context.Background(), "no-invite@example.com", "Password123!", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)

	ok, err := userRepo.ExistsByEmail(context.Background(), "no-invite@example.com")
	require.NoError(t, err)
	require.True(t, ok, "可选邀请码缺省时应正常产生用户")
}
