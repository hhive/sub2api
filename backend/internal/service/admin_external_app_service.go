package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	adminAppLaunchTokenVersion = 1
	adminAppLaunchPurpose      = "admin_sso"
	adminAppLaunchTTL          = 60 * time.Second
)

type AdminAppLaunchTokenData struct {
	Version   int    `json:"version"`
	Purpose   string `json:"purpose"`
	AppID     string `json:"app_id"`
	UserID    int64  `json:"user_id"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type AdminAppLaunchTokenStore interface {
	StoreAdminAppLaunchToken(context.Context, string, *AdminAppLaunchTokenData, time.Duration) error
	ConsumeAdminAppLaunchToken(context.Context, string) (*AdminAppLaunchTokenData, error)
}

type AdminExternalAppMetadata struct {
	AppID     string `json:"app_id"`
	LabelEN   string `json:"label_en"`
	LabelZH   string `json:"label_zh"`
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}

type AdminExternalAppLaunchResult struct {
	RedirectURL string `json:"redirect_url"`
	ExpiresIn   int    `json:"expires_in"`
}

type AdminExternalAppExchangePayload struct {
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	AppID     string `json:"app_id"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type AdminExternalAppService struct {
	registry   AdminExternalAppRegistry
	tokenStore AdminAppLaunchTokenStore
	userRepo   UserRepository
	now        func() time.Time
}

func NewAdminExternalAppService(registry AdminExternalAppRegistry, tokenStore AdminAppLaunchTokenStore, userRepo UserRepository) *AdminExternalAppService {
	return &AdminExternalAppService{registry: registry, tokenStore: tokenStore, userRepo: userRepo, now: time.Now}
}

func (s *AdminExternalAppService) List() []AdminExternalAppMetadata {
	apps := s.registry.List()
	result := make([]AdminExternalAppMetadata, 0, len(apps))
	for _, app := range apps {
		if app.Enabled {
			result = append(result, AdminExternalAppMetadata{AppID: app.AppID, LabelEN: app.LabelEN, LabelZH: app.LabelZH, Enabled: true, SortOrder: app.SortOrder})
		}
	}
	return result
}

func (s *AdminExternalAppService) CreateLaunch(ctx context.Context, appID string, userID int64) (*AdminExternalAppLaunchResult, error) {
	app, err := s.enabledApp(appID)
	if err != nil {
		return nil, err
	}
	if _, err := s.activeAdmin(ctx, userID); err != nil {
		return nil, err
	}
	token, err := randomAdminAppLaunchToken()
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	data := &AdminAppLaunchTokenData{Version: adminAppLaunchTokenVersion, Purpose: adminAppLaunchPurpose, AppID: app.AppID, UserID: userID, IssuedAt: now.Unix(), ExpiresAt: now.Add(adminAppLaunchTTL).Unix()}
	if err := s.tokenStore.StoreAdminAppLaunchToken(ctx, token, data, adminAppLaunchTTL); err != nil {
		return nil, err
	}
	redirect, err := url.Parse(app.LaunchURL)
	if err != nil {
		return nil, infraerrors.InternalServer("ADMIN_EXTERNAL_APP_CONFIG_INVALID", "external app configuration invalid")
	}
	query := redirect.Query()
	query.Set("token", token)
	redirect.RawQuery = query.Encode()
	return &AdminExternalAppLaunchResult{RedirectURL: redirect.String(), ExpiresIn: int(adminAppLaunchTTL / time.Second)}, nil
}

func (s *AdminExternalAppService) Exchange(ctx context.Context, appID, token string) (*AdminExternalAppExchangePayload, error) {
	if _, err := s.enabledApp(appID); err != nil {
		return nil, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, invalidAdminAppToken()
	}
	data, err := s.tokenStore.ConsumeAdminAppLaunchToken(ctx, token)
	if err != nil {
		return nil, err
	}
	now := s.now().Unix()
	if data == nil || data.Version != adminAppLaunchTokenVersion || data.Purpose != adminAppLaunchPurpose || data.AppID != appID || data.UserID <= 0 || data.IssuedAt <= 0 || data.IssuedAt > now || data.ExpiresAt <= data.IssuedAt || data.ExpiresAt-data.IssuedAt > int64(adminAppLaunchTTL/time.Second) || data.ExpiresAt <= now {
		return nil, invalidAdminAppToken()
	}
	user, err := s.activeAdmin(ctx, data.UserID)
	if err != nil {
		return nil, err
	}
	return &AdminExternalAppExchangePayload{UserID: user.ID, Email: user.Email, Username: user.Username, Role: user.Role, AppID: appID, IssuedAt: data.IssuedAt, ExpiresAt: data.ExpiresAt}, nil
}

func (s *AdminExternalAppService) ValidateExchangeSecret(appID, actual string) error {
	app, err := s.enabledApp(appID)
	if err != nil {
		return infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_SECRET_INVALID", "external app secret invalid")
	}
	if !constantTimeSecretEqual(strings.TrimSpace(actual), app.ExchangeSecret) {
		return infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_SECRET_INVALID", "external app secret invalid")
	}
	return nil
}

func (s *AdminExternalAppService) enabledApp(appID string) (AdminExternalApp, error) {
	app, ok := s.registry.Get(strings.TrimSpace(appID))
	if !ok || !app.Enabled {
		return AdminExternalApp{}, infraerrors.NotFound("ADMIN_EXTERNAL_APP_NOT_FOUND", "external app not found")
	}
	return app, nil
}

func (s *AdminExternalAppService) activeAdmin(ctx context.Context, userID int64) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive() || !user.IsAdmin() {
		return nil, infraerrors.Forbidden("ADMIN_EXTERNAL_APP_ADMIN_REQUIRED", "active admin required")
	}
	return user, nil
}

func randomAdminAppLaunchToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func constantTimeSecretEqual(actual, expected string) bool {
	return actual != "" && subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func invalidAdminAppToken() error {
	return infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_TOKEN_INVALID", "external app token invalid")
}
