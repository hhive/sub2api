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

var ErrOnyxNoEligibleAPIKey = infraerrors.Conflict("ONYX_NO_ELIGIBLE_API_KEY", "no eligible api key")

type OnyxLaunchTokenData struct {
	UserID   int64 `json:"user_id"`
	APIKeyID int64 `json:"api_key_id"`
}

type OnyxLaunchTokenStore interface {
	StoreLaunchToken(ctx context.Context, token string, data *OnyxLaunchTokenData, ttl time.Duration) error
	GetLaunchToken(ctx context.Context, token string) (*OnyxLaunchTokenData, error)
	DeleteLaunchToken(ctx context.Context, token string) error
}

type OnyxLaunchResult struct {
	RedirectURL string `json:"redirect_url"`
}

const defaultImagePlaygroundBaseURL = "https://xiaoni-ai.top/image_playground/"

type OnyxLaunchPayload struct {
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

type OnyxLaunchService struct {
	apiKeyRepo     APIKeyRepository
	settingService *SettingService
	tokenStore     OnyxLaunchTokenStore
}

func NewOnyxLaunchService(apiKeyRepo APIKeyRepository, settingService *SettingService, tokenStore OnyxLaunchTokenStore) *OnyxLaunchService {
	return &OnyxLaunchService{
		apiKeyRepo:     apiKeyRepo,
		settingService: settingService,
		tokenStore:     tokenStore,
	}
}

func (s *OnyxLaunchService) CreateLaunch(ctx context.Context, userID int64) (*OnyxLaunchResult, error) {
	if s.settingService == nil {
		return nil, infraerrors.ServiceUnavailable("ONYX_SETTINGS_UNAVAILABLE", "onyx settings unavailable")
	}

	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.OnyxEnabled {
		return nil, infraerrors.ServiceUnavailable("ONYX_DISABLED", "onyx integration disabled")
	}
	if settings.OnyxBaseURL == "" {
		return nil, infraerrors.ServiceUnavailable("ONYX_BASE_URL_MISSING", "onyx base url not configured")
	}
	if settings.OnyxExchangeSecret == "" {
		return nil, infraerrors.ServiceUnavailable("ONYX_EXCHANGE_SECRET_MISSING", "onyx exchange secret not configured")
	}

	selectedKey, err := s.selectFirstEligibleAPIKey(ctx, userID)
	if err != nil {
		return nil, err
	}

	launchToken, err := randomOnyxLaunchToken()
	if err != nil {
		return nil, err
	}
	if err := s.storeLaunchToken(ctx, launchToken, userID, selectedKey.ID, settings.OnyxLaunchTokenTTLSeconds); err != nil {
		return nil, err
	}
	redirectURL, err := buildOnyxExchangeRedirectURL(settings.OnyxBaseURL, launchToken)
	if err != nil {
		return nil, err
	}

	return &OnyxLaunchResult{RedirectURL: redirectURL}, nil
}

func (s *OnyxLaunchService) CreateImagePlaygroundLaunch(ctx context.Context, userID int64) (*OnyxLaunchResult, error) {
	if s.settingService == nil {
		return nil, infraerrors.ServiceUnavailable("IMAGE_PLAYGROUND_SETTINGS_UNAVAILABLE", "image playground settings unavailable")
	}

	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}
	apiBaseURL := normalizeOnyxOpenAICompatibleAPIBaseURL(settings.APIBaseURL)
	if apiBaseURL == "" {
		return nil, infraerrors.ServiceUnavailable("IMAGE_PLAYGROUND_API_BASE_URL_MISSING", "api base url not configured")
	}

	selectedKey, err := s.selectFirstEligibleAPIKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	redirectURL, err := buildImagePlaygroundRedirectURL(imagePlaygroundLaunchBaseURL(), apiBaseURL, selectedKey.Key)
	if err != nil {
		return nil, err
	}

	return &OnyxLaunchResult{RedirectURL: redirectURL}, nil
}

func (s *OnyxLaunchService) CreateLobeHubLaunch(ctx context.Context, userID int64) (*OnyxLaunchResult, error) {
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

	launchToken, err := randomOnyxLaunchToken()
	if err != nil {
		return nil, err
	}
	if err := s.storeLaunchToken(ctx, launchToken, userID, selectedKey.ID, settings.OnyxLaunchTokenTTLSeconds); err != nil {
		return nil, err
	}
	redirectURL, err := buildLobeHubLaunchRedirectURL(settings.LobeHubBaseURL, launchToken)
	if err != nil {
		return nil, err
	}

	return &OnyxLaunchResult{RedirectURL: redirectURL}, nil
}

func (s *OnyxLaunchService) ConsumeLaunch(ctx context.Context, token string) (*OnyxLaunchPayload, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, infraerrors.Unauthorized("ONYX_LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	data, err := s.getLaunchToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := s.deleteLaunchToken(ctx, token); err != nil {
		return nil, err
	}
	if data.UserID <= 0 || data.APIKeyID <= 0 {
		return nil, infraerrors.Unauthorized("ONYX_LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	if s.apiKeyRepo == nil {
		return nil, infraerrors.ServiceUnavailable("ONYX_API_KEY_REPOSITORY_UNAVAILABLE", "onyx api key repository unavailable")
	}
	apiKey, err := s.apiKeyRepo.GetByID(ctx, data.APIKeyID)
	if err != nil {
		return nil, err
	}
	if apiKey == nil || apiKey.UserID != data.UserID || !isOnyxAPIKeyEligible(*apiKey, time.Now()) {
		return nil, ErrOnyxNoEligibleAPIKey
	}
	payload := &OnyxLaunchPayload{
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
		payload.APIBaseURL = normalizeOnyxOpenAICompatibleAPIBaseURL(settings.APIBaseURL)
		payload.TextModelName = settings.OnyxDefaultTextModel
		payload.ImageModelName = settings.OnyxDefaultImageModel
	}
	return payload, nil
}

func (s *OnyxLaunchService) selectFirstEligibleAPIKey(ctx context.Context, userID int64) (*APIKey, error) {
	keys, _, err := s.apiKeyRepo.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 1000}, APIKeyListFilters{})
	if err != nil {
		return nil, err
	}

	eligible := make([]APIKey, 0, len(keys))
	now := time.Now()
	for _, key := range keys {
		if !isOnyxAPIKeyEligible(key, now) {
			continue
		}
		eligible = append(eligible, key)
	}

	if len(eligible) == 0 {
		return nil, ErrOnyxNoEligibleAPIKey
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

func isOnyxAPIKeyEligible(key APIKey, now time.Time) bool {
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

func (s *OnyxLaunchService) storeLaunchToken(ctx context.Context, token string, userID, apiKeyID int64, ttlSeconds int) error {
	if s.tokenStore == nil {
		return nil
	}
	return s.tokenStore.StoreLaunchToken(ctx, token, &OnyxLaunchTokenData{
		UserID:   userID,
		APIKeyID: apiKeyID,
	}, time.Duration(ttlSeconds)*time.Second)
}

func (s *OnyxLaunchService) getLaunchToken(ctx context.Context, token string) (*OnyxLaunchTokenData, error) {
	if s.tokenStore == nil {
		return nil, infraerrors.Unauthorized("ONYX_LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	data, err := s.tokenStore.GetLaunchToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, infraerrors.Unauthorized("ONYX_LAUNCH_TOKEN_INVALID", "launch token invalid")
	}
	return data, nil
}

func (s *OnyxLaunchService) deleteLaunchToken(ctx context.Context, token string) error {
	if s.tokenStore == nil {
		return nil
	}
	return s.tokenStore.DeleteLaunchToken(ctx, token)
}

func randomOnyxLaunchToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func imagePlaygroundLaunchBaseURL() string {
	if baseURL := strings.TrimSpace(os.Getenv("IMAGE_PLAYGROUND_BASE_URL")); baseURL != "" {
		return baseURL
	}
	return defaultImagePlaygroundBaseURL
}

func normalizeOnyxOpenAICompatibleAPIBaseURL(rawBaseURL string) string {
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

func buildOnyxExchangeRedirectURL(baseURL, token string) (string, error) {
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

func buildImagePlaygroundRedirectURL(baseURL, apiBaseURL, apiKey string) (string, error) {
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
