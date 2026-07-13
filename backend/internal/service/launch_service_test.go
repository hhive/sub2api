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

type launchTokenStoreStub struct {
	storeCalls    []launchTokenStoreCall
	storeErr      error
	consumeToken  string
	consumeCalls  int
	consumeResult *LaunchTokenData
	consumeErr    error
}

type launchTokenStoreCall struct {
	token string
	data  *LaunchTokenData
	ttl   time.Duration
}

func (s *launchTokenStoreStub) StoreLaunchToken(ctx context.Context, token string, data *LaunchTokenData, ttl time.Duration) error {
	s.storeCalls = append(s.storeCalls, launchTokenStoreCall{token: token, data: data, ttl: ttl})
	return s.storeErr
}

func (s *launchTokenStoreStub) ConsumeLaunchToken(ctx context.Context, token string) (*LaunchTokenData, error) {
	s.consumeCalls++
	s.consumeToken = token
	if s.consumeErr != nil {
		return nil, s.consumeErr
	}
	return s.consumeResult, nil
}

type launchAPIKeyRepoStub struct {
	APIKeyRepository
	listByUserIDResult []APIKey
	listByUserIDErr    error
	listByUserIDCalls  []int64
	getByIDResult      *APIKey
	getByIDErr         error
	getByIDCalls       []int64
}

func (s *launchAPIKeyRepoStub) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	s.listByUserIDCalls = append(s.listByUserIDCalls, userID)
	return s.listByUserIDResult, &pagination.PaginationResult{Total: int64(len(s.listByUserIDResult))}, s.listByUserIDErr
}

func (s *launchAPIKeyRepoStub) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	s.getByIDCalls = append(s.getByIDCalls, id)
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	return s.getByIDResult, nil
}

func TestLaunchService_SelectFirstEligibleAPIKey_TreatsUnlimitedQuotaAsEligibleAndPicksEarliestCreatedKey(t *testing.T) {
	now := time.Now()
	repo := &launchAPIKeyRepoStub{
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
				ExpiresAt: launchTimePtr(now.Add(-1 * time.Minute)),
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
	svc := &LaunchService{apiKeyRepo: repo}

	key, err := svc.selectFirstEligibleAPIKey(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, int64(101), key.ID)
	require.Equal(t, []int64{42}, repo.listByUserIDCalls)
}

func TestLaunchService_SelectFirstEligibleAPIKey_PrefersImageGenerationGroup(t *testing.T) {
	now := time.Now()
	repo := &launchAPIKeyRepoStub{
		listByUserIDResult: []APIKey{
			{
				ID:        101,
				UserID:    42,
				Status:    StatusActive,
				Quota:     10,
				QuotaUsed: 1,
				CreatedAt: now.Add(-5 * time.Hour),
				Group:     &Group{AllowImageGeneration: false},
			},
			{
				ID:        102,
				UserID:    42,
				Status:    StatusActive,
				Quota:     10,
				QuotaUsed: 1,
				CreatedAt: now.Add(-4 * time.Hour),
				Group:     &Group{AllowImageGeneration: true},
			},
			{
				ID:        103,
				UserID:    42,
				Status:    StatusActive,
				Quota:     10,
				QuotaUsed: 1,
				CreatedAt: now.Add(-3 * time.Hour),
				Group:     &Group{AllowImageGeneration: true},
			},
		},
	}
	svc := &LaunchService{apiKeyRepo: repo}

	key, err := svc.selectFirstEligibleAPIKey(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, int64(102), key.ID)
}

func TestLaunchService_SelectFirstEligibleAPIKey_ReturnsConflictWhenNoEligibleKeyExists(t *testing.T) {
	now := time.Now()
	repo := &launchAPIKeyRepoStub{
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
	svc := &LaunchService{apiKeyRepo: repo}

	key, err := svc.selectFirstEligibleAPIKey(context.Background(), 42)
	require.Nil(t, key)
	require.ErrorIs(t, err, ErrLaunchNoEligibleAPIKey)
}

func TestLaunchService_CreateLobeHubLaunch_ReturnsRedirectURLWithLaunchTokenOnly(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyLobeHubEnabled:        "true",
		SettingKeyLobeHubBaseURL:        "https://lobe.example.com",
		SettingKeyLobeHubExchangeSecret: "exchange-secret",
	}}, &config.Config{})
	now := time.Now()
	repo := &launchAPIKeyRepoStub{listByUserIDResult: []APIKey{{
		ID:        401,
		UserID:    42,
		Key:       "sk-secret-user-key",
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		CreatedAt: now.Add(-1 * time.Hour),
	}}}
	store := &launchTokenStoreStub{}
	svc := &LaunchService{apiKeyRepo: repo, settingService: settings, tokenStore: store}

	result, err := svc.CreateLobeHubLaunch(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.RedirectURL)
	require.NotContains(t, result.RedirectURL, "sk-secret-user-key")

	redirectURL, parseErr := url.Parse(result.RedirectURL)
	require.NoError(t, parseErr)
	require.Equal(t, "https", redirectURL.Scheme)
	require.Equal(t, "lobe.example.com", redirectURL.Host)
	require.Equal(t, "/api/sub2api/launch", redirectURL.Path)
	require.NotEmpty(t, redirectURL.Query().Get("token"))
	require.Len(t, redirectURL.Query(), 1)
	require.Len(t, store.storeCalls, 1)
	require.Equal(t, store.storeCalls[0].token, redirectURL.Query().Get("token"))
	require.Equal(t, int64(42), store.storeCalls[0].data.UserID)
	require.Equal(t, int64(401), store.storeCalls[0].data.APIKeyID)
}

func TestLaunchService_CreateLobeHubLaunch_ReturnsServiceUnavailableWhenDisabled(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyLobeHubEnabled: "false",
	}}, &config.Config{})
	svc := &LaunchService{settingService: settings}

	result, err := svc.CreateLobeHubLaunch(context.Background(), 42)
	require.Error(t, err)
	require.Nil(t, result)

	var appErr *infraerrors.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "LOBEHUB_DISABLED", appErr.Reason)
}

func TestLaunchService_CreateMediaPlaygroundLaunch_ReturnsRedirectURLWithAPISettings(t *testing.T) {
	t.Setenv("MEDIA_PLAYGROUND_EXCHANGE_SECRET", "exchange-secret")
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "https://media.example.com/")
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyLaunchTokenTTLSeconds: "60",
	}}, &config.Config{})
	now := time.Now()
	repo := &launchAPIKeyRepoStub{listByUserIDResult: []APIKey{{
		ID:        401,
		UserID:    42,
		Key:       "sk-user-image-key",
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		CreatedAt: now.Add(-1 * time.Hour),
	}}}
	store := &launchTokenStoreStub{}
	svc := &LaunchService{apiKeyRepo: repo, settingService: settings, tokenStore: store}

	result, err := svc.CreateMediaPlaygroundLaunch(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotContains(t, result.RedirectURL, "sk-user-image-key")

	redirectURL, parseErr := url.Parse(result.RedirectURL)
	require.NoError(t, parseErr)
	require.Equal(t, "https", redirectURL.Scheme)
	require.Equal(t, "media.example.com", redirectURL.Host)
	require.Equal(t, "/api/sub2api/launch", redirectURL.Path)
	require.NotEmpty(t, redirectURL.Query().Get("token"))
	require.Len(t, redirectURL.Query(), 1)
	require.Len(t, store.storeCalls, 1)
	require.Equal(t, store.storeCalls[0].token, redirectURL.Query().Get("token"))
	require.Equal(t, 60*time.Second, store.storeCalls[0].ttl)
	require.Equal(t, int64(42), store.storeCalls[0].data.UserID)
	require.Equal(t, int64(401), store.storeCalls[0].data.APIKeyID)
	require.Equal(t, []int64{42}, repo.listByUserIDCalls)
}

func TestLaunchService_CreateMediaPlaygroundLaunch_UsesEnvBaseURLOverride(t *testing.T) {
	t.Setenv("MEDIA_PLAYGROUND_EXCHANGE_SECRET", "exchange-secret")
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "http://127.0.0.1:5173/")
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	now := time.Now()
	repo := &launchAPIKeyRepoStub{listByUserIDResult: []APIKey{{
		ID:        402,
		UserID:    42,
		Key:       "sk-user-local-image-key",
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		CreatedAt: now.Add(-1 * time.Hour),
	}}}
	store := &launchTokenStoreStub{}
	svc := &LaunchService{apiKeyRepo: repo, settingService: settings, tokenStore: store}

	result, err := svc.CreateMediaPlaygroundLaunch(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotContains(t, result.RedirectURL, "sk-user-local-image-key")

	redirectURL, parseErr := url.Parse(result.RedirectURL)
	require.NoError(t, parseErr)
	require.Equal(t, "http", redirectURL.Scheme)
	require.Equal(t, "127.0.0.1:5173", redirectURL.Host)
	require.Equal(t, "/api/sub2api/launch", redirectURL.Path)
	require.NotEmpty(t, redirectURL.Query().Get("token"))
	require.Len(t, redirectURL.Query(), 1)
	require.Len(t, store.storeCalls, 1)
	require.Equal(t, store.storeCalls[0].token, redirectURL.Query().Get("token"))
}

func TestMediaPlaygroundLaunchBaseURL_DefaultsToInfiniteCanvas(t *testing.T) {
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "")
	require.Equal(t, "https://media.xiaoni-ai.top/", mediaPlaygroundLaunchBaseURL())
}

func TestLaunchService_CreateMediaPlaygroundLaunch_ReturnsServiceUnavailableWhenExchangeSecretMissing(t *testing.T) {
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	svc := &LaunchService{settingService: settings}

	result, err := svc.CreateMediaPlaygroundLaunch(context.Background(), 42)
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
}

func TestLaunchService_CreateMediaPlaygroundLaunch_ReturnsConflictWhenNoEligibleAPIKeyExists(t *testing.T) {
	t.Setenv("MEDIA_PLAYGROUND_EXCHANGE_SECRET", "exchange-secret")
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	repo := &launchAPIKeyRepoStub{listByUserIDResult: []APIKey{}}
	svc := &LaunchService{apiKeyRepo: repo, settingService: settings}

	result, err := svc.CreateMediaPlaygroundLaunch(context.Background(), 42)
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrLaunchNoEligibleAPIKey)
	require.Equal(t, []int64{42}, repo.listByUserIDCalls)
}

func TestLaunchService_ConsumeLobeHubLaunch_ReturnsUnauthorizedWhenTokenEmpty(t *testing.T) {
	store := &launchTokenStoreStub{}
	svc := &LaunchService{tokenStore: store}

	result, err := svc.ConsumeLobeHubLaunch(context.Background(), "")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsUnauthorized(err))
	require.Equal(t, 0, store.consumeCalls)
	require.Empty(t, store.consumeToken)
}

func TestLaunchService_ConsumeLobeHubLaunch_InvalidatesTokenAfterSuccessfulConsumption(t *testing.T) {
	store := &launchTokenStoreStub{consumeResult: &LaunchTokenData{UserID: 42, APIKeyID: 1001}}
	repo := &launchAPIKeyRepoStub{getByIDResult: &APIKey{
		ID:        1001,
		UserID:    42,
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
	}}
	svc := &LaunchService{apiKeyRepo: repo, tokenStore: store}

	result, err := svc.ConsumeLobeHubLaunch(context.Background(), "  single-use-token  ")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(42), result.UserID)
	require.Equal(t, int64(1001), result.APIKeyID)
	require.Equal(t, 1, store.consumeCalls)
	require.Equal(t, "single-use-token", store.consumeToken)
}

func TestLaunchService_ConsumeLobeHubLaunch_ReturnsServiceUnavailableWhenAPIKeyRepoMissing(t *testing.T) {
	store := &launchTokenStoreStub{consumeResult: &LaunchTokenData{UserID: 42, APIKeyID: 1001}}
	svc := &LaunchService{tokenStore: store}

	result, err := svc.ConsumeLobeHubLaunch(context.Background(), "valid-token")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, 1, store.consumeCalls)
	require.Equal(t, "valid-token", store.consumeToken)
}

func TestLaunchService_ConsumeLobeHubLaunch_ReturnsConflictWhenBoundAPIKeyNoLongerEligible(t *testing.T) {
	store := &launchTokenStoreStub{consumeResult: &LaunchTokenData{UserID: 42, APIKeyID: 1001}}
	repo := &launchAPIKeyRepoStub{getByIDResult: &APIKey{
		ID:        1001,
		UserID:    42,
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 10,
	}}
	svc := &LaunchService{apiKeyRepo: repo, tokenStore: store}

	result, err := svc.ConsumeLobeHubLaunch(context.Background(), "quota-exhausted-token")
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrLaunchNoEligibleAPIKey)
	require.Equal(t, []int64{1001}, repo.getByIDCalls)
	require.Equal(t, 1, store.consumeCalls)
	require.Equal(t, "quota-exhausted-token", store.consumeToken)
}

func TestLaunchService_ConsumeLobeHubLaunch_ReturnsExchangePayloadForEligibleBoundAPIKey(t *testing.T) {
	store := &launchTokenStoreStub{consumeResult: &LaunchTokenData{UserID: 42, APIKeyID: 1001}}
	repo := &launchAPIKeyRepoStub{getByIDResult: &APIKey{
		ID:        1001,
		UserID:    42,
		Key:       "sk-user-key",
		Status:    StatusActive,
		Quota:     10,
		QuotaUsed: 1,
		User: &User{
			ID:       42,
			Email:    "user@example.com",
			Username: "launch-user",
			Role:     RoleAdmin,
		},
	}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyAPIBaseURL:        "https://sub2api.example.com",
		SettingKeyDefaultTextModel:  "gpt-5.5-mini",
		SettingKeyDefaultImageModel: "gpt-image-2",
	}}, &config.Config{})
	svc := &LaunchService{apiKeyRepo: repo, settingService: settings, tokenStore: store}

	result, err := svc.ConsumeLobeHubLaunch(context.Background(), "valid-token")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(42), result.UserID)
	require.Equal(t, "user@example.com", result.Email)
	require.Equal(t, "launch-user", result.Username)
	require.Equal(t, RoleAdmin, result.Role)
	require.Equal(t, int64(1001), result.APIKeyID)
	require.Equal(t, "sk-user-key", result.APIKey)
	require.Equal(t, "https://sub2api.example.com/v1", result.APIBaseURL)
	require.Equal(t, "gpt-5.5-mini", result.TextModelName)
	require.Equal(t, "gpt-image-2", result.ImageModelName)
}

func TestNormalizeOpenAICompatibleAPIBaseURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "loopback gets v1 path",
			input:    "http://127.0.0.1:8080",
			expected: "http://127.0.0.1:8080/v1",
		},
		{
			name:     "localhost gets v1 path",
			input:    "http://localhost:8080/",
			expected: "http://localhost:8080/v1",
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
			require.Equal(t, tc.expected, normalizeOpenAICompatibleAPIBaseURL(tc.input))
		})
	}
}

func TestLaunchService_ConsumeLobeHubLaunch_ReturnsUnauthorizedWhenTokenDataMissing(t *testing.T) {
	store := &launchTokenStoreStub{consumeResult: nil}
	svc := &LaunchService{tokenStore: store}

	result, err := svc.ConsumeLobeHubLaunch(context.Background(), "nil-data-token")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsUnauthorized(err))
	require.Equal(t, "nil-data-token", store.consumeToken)
}

func TestLaunchService_ConsumeLobeHubLaunch_DeletesTokenWhenTokenDataHasInvalidIDs(t *testing.T) {
	store := &launchTokenStoreStub{consumeResult: &LaunchTokenData{UserID: 42, APIKeyID: 0}}
	svc := &LaunchService{tokenStore: store}

	result, err := svc.ConsumeLobeHubLaunch(context.Background(), "invalid-binding-token")
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, infraerrors.IsUnauthorized(err))
	require.Equal(t, "invalid-binding-token", store.consumeToken)
	require.Equal(t, 1, store.consumeCalls)
}

func TestLaunchService_ConsumeLobeHubLaunch_ReturnsUnauthorizedWhenTokenDataHasInvalidIDs(t *testing.T) {
	testCases := []struct {
		name string
		data *LaunchTokenData
	}{
		{
			name: "user id is zero",
			data: &LaunchTokenData{UserID: 0, APIKeyID: 1001},
		},
		{
			name: "api key id is zero",
			data: &LaunchTokenData{UserID: 42, APIKeyID: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := &launchTokenStoreStub{consumeResult: tc.data}
			svc := &LaunchService{tokenStore: store}

			result, err := svc.ConsumeLobeHubLaunch(context.Background(), "invalid-binding-token")
			require.Nil(t, result)
			require.Error(t, err)
			require.True(t, infraerrors.IsUnauthorized(err))
			require.Equal(t, "invalid-binding-token", store.consumeToken)
		})
	}
}

func launchTimePtr(value time.Time) *time.Time {
	return &value
}
