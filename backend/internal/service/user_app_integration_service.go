package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	userAppTicketTTL = time.Minute
	userAppGrantTTL  = 8 * time.Hour
)

var (
	errUserAppForbidden    = infraerrors.Forbidden("USER_APP_INTEGRATION_FORBIDDEN", "app integration or permission unavailable")
	errUserAppUnauthorized = infraerrors.Unauthorized("USER_APP_INTEGRATION_UNAUTHORIZED", "app integration credentials invalid")
)

type UserAppIntegrationGrant struct {
	UserID int64    `json:"user_id"`
	Scopes []string `json:"scopes"`
}

type UserAppIntegrationStore interface {
	PutTicket(ctx context.Context, appID, digest string, grant *UserAppIntegrationGrant, ttl time.Duration) error
	ConsumeTicket(ctx context.Context, appID, digest string) (*UserAppIntegrationGrant, error)
	PutGrant(ctx context.Context, appID, digest string, grant *UserAppIntegrationGrant, ttl time.Duration) error
	GetGrant(ctx context.Context, appID, digest string) (*UserAppIntegrationGrant, error)
	RevokeGrant(ctx context.Context, appID, digest string) error
}

type UserAppIntegrationLaunch struct {
	LaunchURL string `json:"launch_url"`
	Ticket    string `json:"ticket"`
}

type UserAppIntegrationExchange struct {
	UserID     int64    `json:"user_id"`
	Email      string   `json:"email"`
	Username   string   `json:"username"`
	GrantToken string   `json:"grant_token"`
	ExpiresIn  int64    `json:"expires_in"`
	Scopes     []string `json:"scopes"`
}

type UserAppIntegrationService struct {
	registry map[string]userAppIntegration
	store    UserAppIntegrationStore
	users    UserRepository
}

func NewUserAppIntegrationService(store UserAppIntegrationStore, users UserRepository) (*UserAppIntegrationService, error) {
	registry, err := loadUserAppIntegrations(strings.TrimSpace(os.Getenv("USER_APP_INTEGRATIONS_CONFIG_PATH")))
	if err != nil {
		return nil, err
	}
	if chatAppID := strings.TrimSpace(os.Getenv("CHAT_INTEGRATION_APP_ID")); chatAppID != "" && !registry[chatAppID].Enabled {
		return nil, errors.New("CHAT_INTEGRATION_APP_ID must name an enabled user app integration")
	}
	return &UserAppIntegrationService{registry: registry, store: store, users: users}, nil
}

func (s *UserAppIntegrationService) LaunchFormActions() []string {
	actions := make([]string, 0, len(s.registry))
	for _, app := range s.registry {
		if app.Enabled {
			actions = append(actions, app.LaunchURL)
		}
	}
	sort.Strings(actions)
	return slices.Compact(actions)
}

func (s *UserAppIntegrationService) Launch(ctx context.Context, appID string, userID int64) (*UserAppIntegrationLaunch, error) {
	app := s.registry[appID]
	if !app.Enabled {
		return nil, errUserAppForbidden
	}
	if _, err := s.activeUser(ctx, userID); err != nil {
		return nil, err
	}
	ticket, digest, err := newUserAppCredential()
	if err != nil {
		return nil, err
	}
	grant := &UserAppIntegrationGrant{UserID: userID, Scopes: app.Scopes}
	if err := s.store.PutTicket(ctx, appID, digest, grant, userAppTicketTTL); err != nil {
		return nil, err
	}
	return &UserAppIntegrationLaunch{LaunchURL: app.LaunchURL, Ticket: ticket}, nil
}

func (s *UserAppIntegrationService) Exchange(ctx context.Context, appID, secret, ticket string) (*UserAppIntegrationExchange, error) {
	app, err := s.authenticate(appID, secret)
	if err != nil {
		return nil, err
	}
	if len(ticket) != 64 {
		return nil, errUserAppUnauthorized
	}
	grant, err := s.store.ConsumeTicket(ctx, appID, userAppCredentialDigest(ticket))
	if err != nil {
		return nil, err
	}
	if grant == nil {
		return nil, errUserAppUnauthorized
	}
	user, err := s.activeUser(ctx, grant.UserID)
	if err != nil {
		return nil, err
	}
	scopes := make([]string, 0, len(grant.Scopes))
	for _, scope := range grant.Scopes {
		if slices.Contains(app.Scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	token, digest, err := newUserAppCredential()
	if err != nil {
		return nil, err
	}
	grant = &UserAppIntegrationGrant{UserID: user.ID, Scopes: scopes}
	if err := s.store.PutGrant(ctx, appID, digest, grant, userAppGrantTTL); err != nil {
		return nil, err
	}
	return &UserAppIntegrationExchange{
		UserID: user.ID, Email: user.Email, Username: user.Username,
		GrantToken: token, ExpiresIn: int64(userAppGrantTTL / time.Second), Scopes: scopes,
	}, nil
}

func (s *UserAppIntegrationService) Authorize(ctx context.Context, appID, secret, token, scope string) (int64, error) {
	app, err := s.authenticate(appID, secret)
	if err != nil {
		return 0, err
	}
	if len(token) != 64 {
		return 0, errUserAppUnauthorized
	}
	grant, err := s.store.GetGrant(ctx, appID, userAppCredentialDigest(token))
	if err != nil {
		return 0, err
	}
	if grant == nil {
		return 0, errUserAppUnauthorized
	}
	if !slices.Contains(app.Scopes, scope) || !slices.Contains(grant.Scopes, scope) {
		return 0, errUserAppForbidden
	}
	if _, err := s.activeUser(ctx, grant.UserID); err != nil {
		return 0, err
	}
	return grant.UserID, nil
}

func (s *UserAppIntegrationService) Revoke(ctx context.Context, appID, secret, token string) error {
	if _, err := s.authenticate(appID, secret); err != nil {
		return err
	}
	if len(token) != 64 {
		return errUserAppUnauthorized
	}
	return s.store.RevokeGrant(ctx, appID, userAppCredentialDigest(token))
}

func (s *UserAppIntegrationService) authenticate(appID, secret string) (userAppIntegration, error) {
	app := s.registry[appID]
	if !app.Enabled {
		return app, errUserAppForbidden
	}
	digest := sha256.Sum256([]byte(secret))
	if subtle.ConstantTimeCompare(digest[:], app.secret[:]) != 1 {
		return app, errUserAppUnauthorized
	}
	return app, nil
}

func (s *UserAppIntegrationService) activeUser(ctx context.Context, userID int64) (*User, error) {
	if userID <= 0 {
		return nil, errUserAppForbidden
	}
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, ErrUserNotFound) || (err == nil && (user == nil || !user.IsActive())) {
		return nil, errUserAppForbidden
	}
	return user, err
}

func newUserAppCredential() (string, string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(bytes[:])
	return token, userAppCredentialDigest(token), nil
}

func userAppCredentialDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
