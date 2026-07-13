package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var ErrLaunchNoEligibleAPIKey = infraerrors.Conflict("LAUNCH_NO_ELIGIBLE_API_KEY", "no eligible api key")

type LaunchTokenData struct {
	UserID   int64 `json:"user_id"`
	APIKeyID int64 `json:"api_key_id"`
}

type LaunchTokenStore interface {
	StoreLaunchToken(ctx context.Context, token string, data *LaunchTokenData, ttl time.Duration) error
	ConsumeLaunchToken(ctx context.Context, token string) (*LaunchTokenData, error)
}

type LaunchResult struct {
	RedirectURL string `json:"redirect_url"`
}

const defaultMediaPlaygroundBaseURL = "https://media.xiaoni-ai.top/"

type LobeHubLaunchPayload struct {
	UserID         int64  `json:"user_id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	Role           string `json:"role"`
	APIKeyID       int64  `json:"api_key_id"`
	APIKey         string `json:"api_key"`
	APIBaseURL     string `json:"api_base_url"`
	TextModelName  string `json:"text_model_name"`
	ImageModelName string `json:"image_model_name"`
}

type MediaPlaygroundLaunchPayload struct {
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	APIKeyID int64  `json:"api_key_id"`
	APIKey   string `json:"api_key"`
	GroupID  *int64 `json:"group_id,omitempty"`
}

type LaunchService struct {
	apiKeyRepo     APIKeyRepository
	settingService *SettingService
	tokenStore     LaunchTokenStore
}

func NewLaunchService(apiKeyRepo APIKeyRepository, settingService *SettingService, tokenStore LaunchTokenStore) *LaunchService {
	return &LaunchService{
		apiKeyRepo:     apiKeyRepo,
		settingService: settingService,
		tokenStore:     tokenStore,
	}
}

func (s *LaunchService) CreateMediaPlaygroundLaunch(ctx context.Context, userID int64) (*LaunchResult, error) {
	if s.settingService == nil {
		return nil, infraerrors.ServiceUnavailable("MEDIA_PLAYGROUND_SETTINGS_UNAVAILABLE", "media playground settings unavailable")
	}
	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}
	baseURL := mediaPlaygroundLaunchBaseURL()
	if baseURL == "" {
		return nil, infraerrors.ServiceUnavailable("MEDIA_PLAYGROUND_BASE_URL_MISSING", "media playground base url not configured")
	}
	if MediaPlaygroundExchangeSecret() == "" {
		return nil, infraerrors.ServiceUnavailable("MEDIA_PLAYGROUND_EXCHANGE_SECRET_MISSING", "media playground exchange secret not configured")
	}
	selectedKey, err := s.selectFirstEligibleAPIKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	launchToken, err := randomLaunchToken()
	if err != nil {
		return nil, err
	}
	if err := s.storeLaunchToken(ctx, launchToken, userID, selectedKey.ID, settings.LaunchTokenTTLSeconds); err != nil {
		return nil, err
	}
	redirectURL, err := buildMediaPlaygroundLaunchRedirectURL(baseURL, launchToken)
	if err != nil {
		return nil, err
	}

	return &LaunchResult{RedirectURL: redirectURL}, nil
}

func (s *LaunchService) CreateLobeHubLaunch(ctx context.Context, userID int64) (*LaunchResult, error) {
	if s.settingService == nil {
		return nil, infraerrors.ServiceUnavailable("LOBEHUB_SETTINGS_UNAVAILABLE", "lobehub settings unavailable")
	}

	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.LobeHubEnabled {
		return nil, infraerrors.ServiceUnavailable("LOBEHUB_DISABLED", "lobehub integration disabled")
	}
	if settings.LobeHubBaseURL == "" {
		return nil, infraerrors.ServiceUnavailable("LOBEHUB_BASE_URL_MISSING", "lobehub base url not configured")
	}
	if settings.LobeHubExchangeSecret == "" {
		return nil, infraerrors.ServiceUnavailable("LOBEHUB_EXCHANGE_SECRET_MISSING", "lobehub exchange secret not configured")
	}

	selectedKey, err := s.selectFirstEligibleAPIKey(ctx, userID)
	if err != nil {
		return nil, err
	}

	launchToken, err := randomLaunchToken()
	if err != nil {
		return nil, err
	}
	if err := s.storeLaunchToken(ctx, launchToken, userID, selectedKey.ID, settings.LaunchTokenTTLSeconds); err != nil {
		return nil, err
	}
	redirectURL, err := buildLobeHubLaunchRedirectURL(settings.LobeHubBaseURL, launchToken)
	if err != nil {
		return nil, err
	}

	return &LaunchResult{RedirectURL: redirectURL}, nil
}

func (s *LaunchService) ConsumeLobeHubLaunch(ctx context.Context, token string) (*LobeHubLaunchPayload, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, infraerrors.Unauthorized("LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	data, err := s.consumeLaunchToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if data.UserID <= 0 || data.APIKeyID <= 0 {
		return nil, infraerrors.Unauthorized("LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	if s.apiKeyRepo == nil {
		return nil, infraerrors.ServiceUnavailable("LAUNCH_API_KEY_REPOSITORY_UNAVAILABLE", "launch api key repository unavailable")
	}
	apiKey, err := s.apiKeyRepo.GetByID(ctx, data.APIKeyID)
	if err != nil {
		return nil, err
	}
	if apiKey == nil || apiKey.UserID != data.UserID || !isLaunchAPIKeyEligible(*apiKey, time.Now()) {
		return nil, ErrLaunchNoEligibleAPIKey
	}
	payload := &LobeHubLaunchPayload{
		UserID:   data.UserID,
		APIKeyID: data.APIKeyID,
	}
	payload.APIKey = apiKey.Key
	if apiKey.User != nil {
		payload.Email = apiKey.User.Email
		payload.Username = apiKey.User.Username
		payload.Role = apiKey.User.Role
	}
	if s.settingService != nil {
		settings, err := s.settingService.GetAllSettings(ctx)
		if err != nil {
			return nil, err
		}
		payload.APIBaseURL = normalizeOpenAICompatibleAPIBaseURL(settings.APIBaseURL)
		payload.TextModelName = settings.DefaultTextModel
		payload.ImageModelName = settings.DefaultImageModel
	}
	return payload, nil
}

func (s *LaunchService) ConsumeMediaPlaygroundLaunch(ctx context.Context, token string) (*MediaPlaygroundLaunchPayload, error) {
	return s.consumePlaygroundLaunch(ctx, token, "MEDIA_PLAYGROUND")
}

func (s *LaunchService) consumePlaygroundLaunch(ctx context.Context, token, codePrefix string) (*MediaPlaygroundLaunchPayload, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, infraerrors.Unauthorized(codePrefix+"_LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	data, err := s.consumeLaunchToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if data.UserID <= 0 || data.APIKeyID <= 0 {
		return nil, infraerrors.Unauthorized(codePrefix+"_LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	if s.apiKeyRepo == nil {
		return nil, infraerrors.ServiceUnavailable(codePrefix+"_API_KEY_REPOSITORY_UNAVAILABLE", "api key repository unavailable")
	}
	apiKey, err := s.apiKeyRepo.GetByID(ctx, data.APIKeyID)
	if err != nil {
		return nil, err
	}
	if apiKey == nil || apiKey.UserID != data.UserID || !isLaunchAPIKeyEligible(*apiKey, time.Now()) {
		return nil, ErrLaunchNoEligibleAPIKey
	}
	payload := &MediaPlaygroundLaunchPayload{UserID: data.UserID, APIKeyID: apiKey.ID, APIKey: apiKey.Key}
	if apiKey.GroupID != nil && *apiKey.GroupID > 0 {
		payload.GroupID = apiKey.GroupID
	}
	if apiKey.User != nil {
		payload.Email = apiKey.User.Email
		payload.Username = apiKey.User.Username
		payload.Role = apiKey.User.Role
	}
	return payload, nil
}

func (s *LaunchService) selectFirstEligibleAPIKey(ctx context.Context, userID int64) (*APIKey, error) {
	keys, _, err := s.apiKeyRepo.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 1000}, APIKeyListFilters{})
	if err != nil {
		return nil, err
	}

	eligible := make([]APIKey, 0, len(keys))
	now := time.Now()
	for _, key := range keys {
		if !isLaunchAPIKeyEligible(key, now) {
			continue
		}
		eligible = append(eligible, key)
	}

	if len(eligible) == 0 {
		return nil, ErrLaunchNoEligibleAPIKey
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		iSupportsImages := apiKeySupportsImageGeneration(eligible[i])
		jSupportsImages := apiKeySupportsImageGeneration(eligible[j])
		if iSupportsImages != jSupportsImages {
			return iSupportsImages
		}
		if eligible[i].CreatedAt.Equal(eligible[j].CreatedAt) {
			return eligible[i].ID < eligible[j].ID
		}
		return eligible[i].CreatedAt.Before(eligible[j].CreatedAt)
	})

	selected := eligible[0]
	return &selected, nil
}

func apiKeySupportsImageGeneration(key APIKey) bool {
	return key.Group != nil && key.Group.AllowImageGeneration
}

func isLaunchAPIKeyEligible(key APIKey, now time.Time) bool {
	if key.Status != StatusActive {
		return false
	}
	if key.ExpiresAt != nil && !key.ExpiresAt.After(now) {
		return false
	}
	if key.Quota <= 0 {
		return true
	}
	return key.QuotaUsed < key.Quota
}

func (s *LaunchService) storeLaunchToken(ctx context.Context, token string, userID, apiKeyID int64, ttlSeconds int) error {
	if s.tokenStore == nil {
		return nil
	}
	return s.tokenStore.StoreLaunchToken(ctx, token, &LaunchTokenData{
		UserID:   userID,
		APIKeyID: apiKeyID,
	}, time.Duration(ttlSeconds)*time.Second)
}

func (s *LaunchService) consumeLaunchToken(ctx context.Context, token string) (*LaunchTokenData, error) {
	if s.tokenStore == nil {
		return nil, infraerrors.Unauthorized("LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	data, err := s.tokenStore.ConsumeLaunchToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, infraerrors.Unauthorized("LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	return data, nil
}

func randomLaunchToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func mediaPlaygroundLaunchBaseURL() string {
	if baseURL := strings.TrimSpace(os.Getenv("MEDIA_PLAYGROUND_BASE_URL")); baseURL != "" {
		return baseURL
	}
	return defaultMediaPlaygroundBaseURL
}

func MediaPlaygroundExchangeSecret() string {
	return strings.TrimSpace(os.Getenv("MEDIA_PLAYGROUND_EXCHANGE_SECRET"))
}

func normalizeOpenAICompatibleAPIBaseURL(rawBaseURL string) string {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	if rawBaseURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawBaseURL
	}

	if strings.TrimRight(parsed.Path, "/") != "/v1" {
		parsed.Path = path.Join(parsed.Path, "/v1")
	}

	return parsed.String()
}

func buildLaunchExchangeRedirectURL(baseURL, token string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, "/api/sub2api/exchange")
	parsed.RawQuery = url.Values{"token": []string{token}}.Encode()
	return parsed.String(), nil
}

func buildLobeHubLaunchRedirectURL(baseURL, token string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, "/api/sub2api/launch")
	parsed.RawQuery = url.Values{"token": []string{token}}.Encode()
	return parsed.String(), nil
}

func buildMediaPlaygroundLaunchRedirectURL(baseURL, token string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, "/api/sub2api/launch")
	parsed.RawQuery = url.Values{"token": []string{token}}.Encode()
	return parsed.String(), nil
}

func buildMediaPlaygroundRedirectURL(baseURL, apiBaseURL, apiKey string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("apiUrl", apiBaseURL)
	query.Set("apiKey", apiKey)
	query.Set("apiMode", "images")
	query.Set("codexCli", "true")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
