//go:build unit

package service

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type adminAppRegistryStub struct{ apps map[string]AdminExternalApp }

func (r *adminAppRegistryStub) List() []AdminExternalApp {
	out := make([]AdminExternalApp, 0, len(r.apps))
	for _, app := range r.apps {
		out = append(out, app)
	}
	return out
}
func (r *adminAppRegistryStub) Get(appID string) (AdminExternalApp, bool) {
	app, ok := r.apps[appID]
	return app, ok
}

type adminAppTokenStoreStub struct {
	stored       *AdminAppLaunchTokenData
	consumed     *AdminAppLaunchTokenData
	consumeCalls int
	consumeOnce  bool
}

func (s *adminAppTokenStoreStub) StoreAdminAppLaunchToken(_ context.Context, _ string, data *AdminAppLaunchTokenData, _ time.Duration) error {
	s.stored = data
	return nil
}
func (s *adminAppTokenStoreStub) ConsumeAdminAppLaunchToken(_ context.Context, _ string) (*AdminAppLaunchTokenData, error) {
	s.consumeCalls++
	data := s.consumed
	if s.consumeOnce {
		s.consumed = nil
	}
	return data, nil
}

type adminAppUserRepoStub struct {
	UserRepository
	user *User
}

func (r *adminAppUserRepoStub) GetByID(context.Context, int64) (*User, error) { return r.user, nil }

func TestAdminExternalAppService_CreateAndExchangeWithoutAPIKey(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	registry := &adminAppRegistryStub{apps: map[string]AdminExternalApp{"media-management": {
		AppID: "media-management", LabelEN: "Media management", LabelZH: "媒体管理",
		LaunchURL: "https://media.example.com/api/sub2api/admin-launch", ExchangeSecret: "0123456789abcdef0123456789abcdef", Enabled: true,
	}}}
	store := &adminAppTokenStoreStub{}
	repo := &adminAppUserRepoStub{user: &User{ID: 42, Email: "admin@example.com", Username: "admin", Role: RoleAdmin, Status: StatusActive}}
	svc := NewAdminExternalAppService(registry, store, repo)
	svc.now = func() time.Time { return now }

	launch, err := svc.CreateLaunch(context.Background(), "media-management", 42)
	require.NoError(t, err)
	require.Equal(t, 60, launch.ExpiresIn)
	require.Equal(t, "admin_sso", store.stored.Purpose)
	require.Equal(t, "media-management", store.stored.AppID)
	redirect, err := url.Parse(launch.RedirectURL)
	require.NoError(t, err)
	require.Equal(t, "https", redirect.Scheme)
	require.Equal(t, "media.example.com", redirect.Host)
	require.Equal(t, "/api/sub2api/admin-launch", redirect.Path)
	require.Len(t, redirect.Query(), 1)
	require.NotEmpty(t, redirect.Query().Get("token"))
	require.NotContains(t, launch.RedirectURL, "admin@example.com")

	store.consumed = store.stored
	payload, err := svc.Exchange(context.Background(), "media-management", "token")
	require.NoError(t, err)
	require.Equal(t, int64(42), payload.UserID)
	require.Equal(t, RoleAdmin, payload.Role)
	require.Equal(t, "media-management", payload.AppID)
}

func TestAdminExternalAppService_CreateRejectsInvalidAppOrAdministrator(t *testing.T) {
	registry := &adminAppRegistryStub{apps: map[string]AdminExternalApp{
		"disabled": {AppID: "disabled", Enabled: false},
		"enabled":  {AppID: "enabled", LaunchURL: "https://media.example.com/launch", Enabled: true},
	}}
	tests := []struct {
		name  string
		appID string
		user  *User
	}{
		{name: "unknown app", appID: "unknown", user: &User{ID: 42, Role: RoleAdmin, Status: StatusActive}},
		{name: "disabled app", appID: "disabled", user: &User{ID: 42, Role: RoleAdmin, Status: StatusActive}},
		{name: "regular user", appID: "enabled", user: &User{ID: 42, Role: RoleUser, Status: StatusActive}},
		{name: "disabled admin", appID: "enabled", user: &User{ID: 42, Role: RoleAdmin, Status: StatusDisabled}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &adminAppTokenStoreStub{}
			svc := NewAdminExternalAppService(registry, store, &adminAppUserRepoStub{user: tc.user})
			_, err := svc.CreateLaunch(context.Background(), tc.appID, 42)
			require.Error(t, err)
			require.Nil(t, store.stored)
		})
	}
}

func TestAdminExternalAppService_WrongAudienceAndDisabledAdminRejected(t *testing.T) {
	registry := &adminAppRegistryStub{apps: map[string]AdminExternalApp{"media-management": {AppID: "media-management", Enabled: true}}}
	now := time.Now()
	store := &adminAppTokenStoreStub{consumed: &AdminAppLaunchTokenData{Version: 1, Purpose: "admin_sso", AppID: "other", UserID: 42, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()}}
	repo := &adminAppUserRepoStub{user: &User{ID: 42, Role: RoleAdmin, Status: StatusDisabled}}
	svc := NewAdminExternalAppService(registry, store, repo)

	_, err := svc.Exchange(context.Background(), "media-management", "token")
	require.Error(t, err)
	require.Equal(t, 1, store.consumeCalls)

	store.consumed.AppID = "media-management"
	_, err = svc.Exchange(context.Background(), "media-management", "token")
	require.Error(t, err)
}

func TestAdminExternalAppService_RejectsInvalidReplayAndDowngradedTickets(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	registry := &adminAppRegistryStub{apps: map[string]AdminExternalApp{"media-management": {AppID: "media-management", Enabled: true}}}
	valid := AdminAppLaunchTokenData{Version: 1, Purpose: "admin_sso", AppID: "media-management", UserID: 42, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()}
	tests := []struct {
		name string
		edit func(*AdminAppLaunchTokenData)
		user *User
	}{
		{name: "expired", edit: func(data *AdminAppLaunchTokenData) { data.ExpiresAt = now.Add(-time.Second).Unix() }},
		{name: "wrong version", edit: func(data *AdminAppLaunchTokenData) { data.Version = 0 }},
		{name: "wrong purpose", edit: func(data *AdminAppLaunchTokenData) { data.Purpose = "launch" }},
		{name: "future issued", edit: func(data *AdminAppLaunchTokenData) {
			data.IssuedAt = now.Add(time.Minute).Unix()
			data.ExpiresAt = now.Add(2 * time.Minute).Unix()
		}},
		{name: "overlong lifetime", edit: func(data *AdminAppLaunchTokenData) {
			data.IssuedAt = now.Add(-time.Second).Unix()
			data.ExpiresAt = now.Add(10 * time.Minute).Unix()
		}},
		{name: "regular user", user: &User{ID: 42, Role: RoleUser, Status: StatusActive}},
		{name: "disabled admin", user: &User{ID: 42, Role: RoleAdmin, Status: StatusDisabled}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := valid
			if tc.edit != nil {
				tc.edit(&data)
			}
			user := tc.user
			if user == nil {
				user = &User{ID: 42, Role: RoleAdmin, Status: StatusActive}
			}
			store := &adminAppTokenStoreStub{consumed: &data}
			svc := NewAdminExternalAppService(registry, store, &adminAppUserRepoStub{user: user})
			svc.now = func() time.Time { return now }
			_, err := svc.Exchange(context.Background(), "media-management", "token")
			require.Error(t, err)
		})
	}

	store := &adminAppTokenStoreStub{consumed: &valid, consumeOnce: true}
	svc := NewAdminExternalAppService(registry, store, &adminAppUserRepoStub{user: &User{ID: 42, Role: RoleAdmin, Status: StatusActive}})
	svc.now = func() time.Time { return now }
	_, err := svc.Exchange(context.Background(), "media-management", "token")
	require.NoError(t, err)
	_, err = svc.Exchange(context.Background(), "media-management", "token")
	require.Error(t, err)
}
