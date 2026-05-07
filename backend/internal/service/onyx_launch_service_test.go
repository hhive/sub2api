//go:build unit

package service

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type onyxLaunchTokenStoreStub struct {
	storeCalls  []onyxLaunchTokenStoreCall
	storeErr    error
	getToken    string
	getCalls    int
	getResult   *OnyxLaunchTokenData
	getErr      error
	deleteToken string
	deleteCalls int
	deleteErr   error
}

type onyxLaunchTokenStoreCall struct {
	token string
	data  *OnyxLaunchTokenData
	ttl   time.Duration
}

func (s *onyxLaunchTokenStoreStub) StoreLaunchToken(ctx context.Context, token string, data *OnyxLaunchTokenData, ttl time.Duration) error {
	s.storeCalls = append(s.storeCalls, onyxLaunchTokenStoreCall{token: token, data: data, ttl: ttl})
	return s.storeErr
}

func (s *onyxLaunchTokenStoreStub) GetLaunchToken(ctx context.Context, token string) (*OnyxLaunchTokenData, error) {
	s.getCalls++
	s.getToken = token
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResult, nil
}

func (s *onyxLaunchTokenStoreStub) DeleteLaunchToken(ctx context.Context, token string) error {
	s.deleteCalls++
	s.deleteToken = token
	return s.deleteErr
}

type onyxLaunchAPIKeyRepoStub struct {
	APIKeyRepository
	listByUserIDResult []APIKey
	listByUserIDErr    error
	listByUserIDCalls  []int64
	getByIDResult      *APIKey
	getByIDErr         error
	getByIDCalls       []int64
}

func (s *onyxLaunchAPIKeyRepoStub) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	s.listByUserIDCalls = append(s.listByUserIDCalls, userID)
	return s.listByUserIDResult, &pagination.PaginationResult{Total: int64(len(s.listByUserIDResult))}, s.listByUserIDErr
}

func (s *onyxLaunchAPIKeyRepoStub) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	s.getByIDCalls = append(s.getByIDCalls, id)
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	return s.getByIDResult, nil
}

func TestOnyxLaunchService_SelectFirstEligibleAPIKey_TreatsUnlimitedQuotaAsEligibleAndPicksEarliestCreatedKey(t *testing.T) {
	now := time.Now()
	repo := &onyxLaunchAPIKeyRepoStub{
		listByUserIDResult: []APIKey{
			{
				ID:        101,
				UserID:    42,
				Status:    StatusActive,
				Quota:     0,
				QuotaUsed: 0,
				CreatedAt: now.Add(-5 * time.Hour),
			},
			{
				ID:        102,
				UserID:    42,
				Status:    "inactive",
				Quota:     10,
				QuotaUsed: 1,
				CreatedAt: now.Add(-4 * time.Hour),
			},
			{
				ID:        103,
				UserID:    42,
				Status:    StatusActive,
				Quota:     10,
				QuotaUsed: 10,
				CreatedAt: now.Add(-3 * time.Hour),
			},
			{
				ID:        104,
				UserID:    42,
				Status:    StatusActive,
				Quota:     10,
				QuotaUsed: 2,
				ExpiresAt: onyxTimePtr(now.Add(-1 * time.Minute)),
				CreatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:        105,
				UserID:    42,
				Status:    StatusActive,
				Quota:     10,
				QuotaUsed: 3,
				CreatedAt: now.Add(-90 * time.Minute),
			},
			{
				ID:        106,
				UserID:    42,
				Status:    StatusActive,
				Quota:     20,
				QuotaUsed: 1,
				CreatedAt: now.Add(-30 * time.Minute),
			},
		},
	}
	svc := &OnyxLaunchService{apiKeyRepo: repo}

	key, err := svc.selectFirstEligibleAPIKey(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, int64(101), key.ID)
	require.Equal(t, []int64{42}, repo.listByUserIDCalls)
}

func TestOnyxLaunchService_SelectFirstEligibleAPIKey_ReturnsConflictWhenNoEligibleKeyExists(t *testing.T) {
	now := time.Now()
	repo := &onyxLaunchAPIKeyRepoStub{
		listByUserIDResult: []APIKey{
			{
				ID:        201,
				UserID:    42,
				Status:    "inactive",
				Quota:     10,
				QuotaUsed: 0,
				CreatedAt: now.Add(-3 * time.Hour),
			},
			{
				ID:        202,
				UserID:    42,
				Status:    StatusActive,
				Quota:     5,
				QuotaUsed: 5,
				CreatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:        203,
				UserID:    42,
				Status:    StatusActive,
				Quota:     5,
				QuotaUsed: 5,
				CreatedAt: now.Add(-1 * time.Hour),
			},
		},
	}
	svc := &OnyxLaunchService{apiKeyRepo: repo}

	key, err := svc.selectFirstEligibleAPIKey(context.Background(), 42)
	require.Nil(t, key)
	require.ErrorIs(t, err, ErrOnyxNoEligibleAPIKey)
}

func TestOnyxLaunchService_CreateLaunch_ReturnsServiceUnavailableWhenOnyxBaseURLMissing(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyOnyxEnabled:        "true",
		SettingKeyOnyxExchangeSecret: "exchange-secret",
	}}, &config.Config{})
	svc := &OnyxLaunchService{settingService: settings}

	result, err := svc.CreateLaunch(context.Background(), 42)
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
}

func TestOnyxLaunchService_CreateLaunch_ReturnsServiceUnavailableWhenOnyxExchangeSecretMissing(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyOnyxEnabled: "true",
		SettingKeyOnyxBaseURL: "https://onyx.example.com",
	}}, &config.Config{})
	svc := &OnyxLaunchService{settingService: settings}

	result, err := svc.CreateLaunch(context.Background(), 42)
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
}

func TestOnyxLaunchService_CreateLaunch_ReturnsConflictWhenNoEligibleAPIKeyExists(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyOnyxEnabled:        "true",
		SettingKeyOnyxBaseURL:        "https://onyx.example.com",
		SettingKeyOnyxExchangeSecret: "exchange-secret",
	}}, &config.Config{})
	repo := &onyxLaunchAPIKeyRepoStub{listByUserIDResult: []APIKey{}}
	svc := &OnyxLaunchService{apiKeyRepo: repo, settingService: settings}

	result, err := svc.CreateLaunch(context.Background(), 42)
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrOnyxNoEligibleAPIKey)
	require.Equal(t, []int64{42}, repo.listByUserIDCalls)
}

func TestOnyxLaunchService_CreateLaunch_ReturnsRedirectURLWithLaunchToken(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyOnyxEnabled:        "true",
		SettingKeyOnyxBaseURL:        "https://onyx.example.com",
		SettingKeyOnyxExchangeSecret: "exchange-secret",
	}}, &config.Config{})
	now := time.Now()
	repo := &onyxLaunchAPIKeyRepoStub{listByUserIDResult: []APIKey{{
		ID:        301,
		UserID:    42,
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		CreatedAt: now.Add(-1 * time.Hour),
	}}}
	svc := &OnyxLaunchService{apiKeyRepo: repo, settingService: settings}

	result, err := svc.CreateLaunch(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.RedirectURL)

	redirectURL, parseErr := url.Parse(result.RedirectURL)
	require.NoError(t, parseErr)
	require.Equal(t, "https", redirectURL.Scheme)
	require.Equal(t, "onyx.example.com", redirectURL.Host)
	require.Equal(t, "/api/sub2api/exchange", redirectURL.Path)
	require.NotEmpty(t, redirectURL.Query().Get("token"))
	require.Len(t, redirectURL.Query(), 1)
}

func TestOnyxLaunchService_CreateLaunch_StoresLaunchTokenWithTTLAndBinding(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyOnyxEnabled:               "true",
		SettingKeyOnyxBaseURL:               "https://onyx.example.com",
		SettingKeyOnyxExchangeSecret:        "exchange-secret",
		SettingKeyOnyxLaunchTokenTTLSeconds: "60",
	}}, &config.Config{})
	now := time.Now()
	repo := &onyxLaunchAPIKeyRepoStub{listByUserIDResult: []APIKey{{
		ID:        302,
		UserID:    42,
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		CreatedAt: now.Add(-1 * time.Hour),
	}}}
	store := &onyxLaunchTokenStoreStub{}
	svc := &OnyxLaunchService{apiKeyRepo: repo, settingService: settings, tokenStore: store}

	result, err := svc.CreateLaunch(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, store.storeCalls, 1)
	require.NotEmpty(t, store.storeCalls[0].token)
	require.Equal(t, 60*time.Second, store.storeCalls[0].ttl)
	require.NotNil(t, store.storeCalls[0].data)
	require.Equal(t, int64(42), store.storeCalls[0].data.UserID)
	require.Equal(t, int64(302), store.storeCalls[0].data.APIKeyID)

	redirectURL, parseErr := url.Parse(result.RedirectURL)
	require.NoError(t, parseErr)
	require.Equal(t, store.storeCalls[0].token, redirectURL.Query().Get("token"))
}

func TestOnyxLaunchService_CreateImagePlaygroundLaunch_ReturnsRedirectURLWithAPISettings(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyAPIBaseURL: "https://xiaoni-ai.zle.ee/v1",
	}}, &config.Config{})
	now := time.Now()
	repo := &onyxLaunchAPIKeyRepoStub{listByUserIDResult: []APIKey{{
		ID:        401,
		UserID:    42,
		Key:       "sk-user-image-key",
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		CreatedAt: now.Add(-1 * time.Hour),
	}}}
	svc := &OnyxLaunchService{apiKeyRepo: repo, settingService: settings}

	result, err := svc.CreateImagePlaygroundLaunch(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)

	redirectURL, parseErr := url.Parse(result.RedirectURL)
	require.NoError(t, parseErr)
	require.Equal(t, "https", redirectURL.Scheme)
	require.Equal(t, "xiaoni-ai.zle.ee", redirectURL.Host)
	require.Equal(t, "/image_playground/", redirectURL.Path)
	require.Equal(t, "https://xiaoni-ai.zle.ee/v1", redirectURL.Query().Get("apiUrl"))
	require.Equal(t, "sk-user-image-key", redirectURL.Query().Get("apiKey"))
	require.Equal(t, "images", redirectURL.Query().Get("apiMode"))
	require.Equal(t, "true", redirectURL.Query().Get("codexCli"))
	require.Len(t, redirectURL.Query(), 4)
	require.Equal(t, []int64{42}, repo.listByUserIDCalls)
}

func TestOnyxLaunchService_CreateImagePlaygroundLaunch_ReturnsServiceUnavailableWhenAPIBaseURLMissing(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	svc := &OnyxLaunchService{settingService: settings}

	result, err := svc.CreateImagePlaygroundLaunch(context.Background(), 42)
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
}

func TestOnyxLaunchService_CreateImagePlaygroundLaunch_ReturnsConflictWhenNoEligibleAPIKeyExists(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyAPIBaseURL: "https://xiaoni-ai.zle.ee/v1",
	}}, &config.Config{})
	repo := &onyxLaunchAPIKeyRepoStub{listByUserIDResult: []APIKey{}}
	svc := &OnyxLaunchService{apiKeyRepo: repo, settingService: settings}

	result, err := svc.CreateImagePlaygroundLaunch(context.Background(), 42)
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrOnyxNoEligibleAPIKey)
	require.Equal(t, []int64{42}, repo.listByUserIDCalls)
}

func TestOnyxLaunchService_ConsumeLaunch_ReturnsUnauthorizedWhenTokenEmpty(t *testing.T) {
	store := &onyxLaunchTokenStoreStub{}
	svc := &OnyxLaunchService{tokenStore: store}

	result, err := svc.ConsumeLaunch(context.Background(), "")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsUnauthorized(err))
	require.Equal(t, 0, store.getCalls)
	require.Empty(t, store.getToken)
}

func TestOnyxLaunchService_ConsumeLaunch_InvalidatesTokenAfterSuccessfulConsumption(t *testing.T) {
	store := &onyxLaunchTokenStoreStub{getResult: &OnyxLaunchTokenData{UserID: 42, APIKeyID: 1001}}
	repo := &onyxLaunchAPIKeyRepoStub{getByIDResult: &APIKey{
		ID:        1001,
		UserID:    42,
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
	}}
	svc := &OnyxLaunchService{apiKeyRepo: repo, tokenStore: store}

	result, err := svc.ConsumeLaunch(context.Background(), "  single-use-token  ")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(42), result.UserID)
	require.Equal(t, int64(1001), result.APIKeyID)
	require.Equal(t, 1, store.deleteCalls)
	require.Equal(t, "single-use-token", store.deleteToken)
}

func TestOnyxLaunchService_ConsumeLaunch_ReturnsServiceUnavailableWhenAPIKeyRepoMissing(t *testing.T) {
	store := &onyxLaunchTokenStoreStub{getResult: &OnyxLaunchTokenData{UserID: 42, APIKeyID: 1001}}
	svc := &OnyxLaunchService{tokenStore: store}

	result, err := svc.ConsumeLaunch(context.Background(), "valid-token")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, 1, store.deleteCalls)
	require.Equal(t, "valid-token", store.deleteToken)
}

func TestOnyxLaunchService_ConsumeLaunch_ReturnsConflictWhenBoundAPIKeyNoLongerEligible(t *testing.T) {
	store := &onyxLaunchTokenStoreStub{getResult: &OnyxLaunchTokenData{UserID: 42, APIKeyID: 1001}}
	repo := &onyxLaunchAPIKeyRepoStub{getByIDResult: &APIKey{
		ID:        1001,
		UserID:    42,
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 10,
	}}
	svc := &OnyxLaunchService{apiKeyRepo: repo, tokenStore: store}

	result, err := svc.ConsumeLaunch(context.Background(), "quota-exhausted-token")
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrOnyxNoEligibleAPIKey)
	require.Equal(t, []int64{1001}, repo.getByIDCalls)
	require.Equal(t, 1, store.deleteCalls)
	require.Equal(t, "quota-exhausted-token", store.deleteToken)
}

func TestOnyxLaunchService_ConsumeLaunch_ReturnsExchangePayloadForEligibleBoundAPIKey(t *testing.T) {
	store := &onyxLaunchTokenStoreStub{getResult: &OnyxLaunchTokenData{UserID: 42, APIKeyID: 1001}}
	repo := &onyxLaunchAPIKeyRepoStub{getByIDResult: &APIKey{
		ID:        1001,
		UserID:    42,
		Key:       "sk-user-key",
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		User: &User{
			ID:       42,
			Email:    "user@example.com",
			Username: "onyx-user",
		},
	}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyAPIBaseURL:            "https://sub2api.example.com",
		SettingKeyOnyxDefaultTextModel:  "gpt-5.5-mini",
		SettingKeyOnyxDefaultImageModel: "gpt-image-2",
	}}, &config.Config{})
	svc := &OnyxLaunchService{apiKeyRepo: repo, settingService: settings, tokenStore: store}

	result, err := svc.ConsumeLaunch(context.Background(), "valid-token")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(42), result.UserID)
	require.Equal(t, "user@example.com", result.Email)
	require.Equal(t, "onyx-user", result.Username)
	require.Equal(t, int64(1001), result.APIKeyID)
	require.Equal(t, "sk-user-key", result.APIKey)
	require.Equal(t, "https://sub2api.example.com/v1", result.APIBaseURL)
	require.Equal(t, "gpt-5.5-mini", result.TextModelName)
	require.Equal(t, "gpt-image-2", result.ImageModelName)
}

func TestNormalizeOnyxOpenAICompatibleAPIBaseURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "windows host loopback becomes docker host with v1",
			input:    "http://127.0.0.1:8080",
			expected: "http://host.docker.internal:8080/v1",
		},
		{
			name:     "localhost becomes docker host with v1",
			input:    "http://localhost:8080/",
			expected: "http://host.docker.internal:8080/v1",
		},
		{
			name:     "external base gets v1 path",
			input:    "https://sub2api.example.com",
			expected: "https://sub2api.example.com/v1",
		},
		{
			name:     "existing v1 path stays stable",
			input:    "https://sub2api.example.com/v1",
			expected: "https://sub2api.example.com/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, normalizeOnyxOpenAICompatibleAPIBaseURL(tc.input))
		})
	}
}

func TestOnyxLaunchService_ConsumeLaunch_ReturnsUnauthorizedWhenTokenDataMissing(t *testing.T) {
	store := &onyxLaunchTokenStoreStub{getResult: nil}
	svc := &OnyxLaunchService{tokenStore: store}

	result, err := svc.ConsumeLaunch(context.Background(), "nil-data-token")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsUnauthorized(err))
	require.Equal(t, "nil-data-token", store.getToken)
}

func TestOnyxLaunchService_ConsumeLaunch_DeletesTokenWhenTokenDataHasInvalidIDs(t *testing.T) {
	store := &onyxLaunchTokenStoreStub{getResult: &OnyxLaunchTokenData{UserID: 42, APIKeyID: 0}}
	svc := &OnyxLaunchService{tokenStore: store}

	result, err := svc.ConsumeLaunch(context.Background(), "invalid-binding-token")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsUnauthorized(err))
	require.Equal(t, "invalid-binding-token", store.getToken)
	require.Equal(t, 1, store.deleteCalls)
	require.Equal(t, "invalid-binding-token", store.deleteToken)
}

func TestOnyxLaunchService_ConsumeLaunch_ReturnsUnauthorizedWhenTokenDataHasInvalidIDs(t *testing.T) {
	testCases := []struct {
		name string
		data *OnyxLaunchTokenData
	}{
		{
			name: "user id is zero",
			data: &OnyxLaunchTokenData{UserID: 0, APIKeyID: 1001},
		},
		{
			name: "api key id is zero",
			data: &OnyxLaunchTokenData{UserID: 42, APIKeyID: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := &onyxLaunchTokenStoreStub{getResult: tc.data}
			svc := &OnyxLaunchService{tokenStore: store}

			result, err := svc.ConsumeLaunch(context.Background(), "invalid-binding-token")
			require.Nil(t, result)
			require.Error(t, err)
			require.True(t, infraerrors.IsUnauthorized(err))
			require.Equal(t, "invalid-binding-token", store.getToken)
		})
	}
}

func onyxTimePtr(value time.Time) *time.Time {
	return &value
}
