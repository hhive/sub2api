//go:build unit

package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type userAppIntegrationUsers struct {
	service.UserRepository
	active bool
}

func (u *userAppIntegrationUsers) GetByID(_ context.Context, id int64) (*service.User, error) {
	user := &service.User{ID: id}
	if u.active {
		user.Status = service.StatusActive
	}
	return user, nil
}

func TestUserAppIntegrationFlow(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := NewUserAppIntegrationStore(client)
	users := &userAppIntegrationUsers{active: true}
	path := filepath.Join(t.TempDir(), "registry.json")
	t.Setenv("USER_APP_INTEGRATIONS_CONFIG_PATH", path)
	t.Setenv("CHAT_INTEGRATION_APP_ID", "")
	const chatSecret = "synthetic-chat-secret-at-least-32"
	const readSecret = "synthetic-read-secret-at-least-32"
	t.Setenv("CHAT_APP_SECRET", chatSecret)
	t.Setenv("READ_APP_SECRET", readSecret)
	writeConfig := func(chatScope string) {
		t.Helper()
		cfg := fmt.Sprintf(`[
			{"app_id":"example-chat","enabled":true,"launch_url":"https://chat.example.test/api/launch","secret_env":"CHAT_APP_SECRET","scopes":[%q]},
			{"app_id":"example-read","enabled":true,"launch_url":"https://read.example.test/api/launch","secret_env":"READ_APP_SECRET","scopes":["keys:read"]}
		]`, chatScope)
		require.NoError(t, os.WriteFile(path, []byte(cfg), 0600))
	}
	ctx := context.Background()
	disabled, err := service.NewUserAppIntegrationService(store, users)
	require.NoError(t, err)
	_, err = disabled.Launch(ctx, "example-chat", 14)
	require.Equal(t, 403, infraerrors.Code(err))

	writeConfig(service.UserAppScopeChatInvoke)
	integration, err := service.NewUserAppIntegrationService(store, users)
	require.NoError(t, err)
	launch, err := integration.Launch(ctx, "example-chat", 14)
	require.NoError(t, err)
	require.Equal(t, []string{fmt.Sprintf("user-app-integration:ticket:example-chat:%x", sha256.Sum256([]byte(launch.Ticket)))}, server.Keys())
	_, err = integration.Exchange(ctx, "example-read", readSecret, launch.Ticket)
	require.Equal(t, 401, infraerrors.Code(err))
	_, err = integration.Exchange(ctx, "example-chat", readSecret, launch.Ticket)
	require.Equal(t, 401, infraerrors.Code(err))
	exchanged, err := integration.Exchange(ctx, "example-chat", chatSecret, launch.Ticket)
	require.NoError(t, err)
	require.Equal(t, int64(14), exchanged.UserID)
	require.Equal(t, int64(28800), exchanged.ExpiresIn)
	require.Equal(t, []string{service.UserAppScopeChatInvoke}, exchanged.Scopes)
	_, err = integration.Exchange(ctx, "example-chat", chatSecret, launch.Ticket)
	require.Equal(t, 401, infraerrors.Code(err))
	id, err := integration.Authorize(ctx, "example-chat", chatSecret, exchanged.GrantToken, service.UserAppScopeChatInvoke)
	require.NoError(t, err)
	require.Equal(t, int64(14), id)
	_, err = integration.Authorize(ctx, "example-read", readSecret, exchanged.GrantToken, service.UserAppScopeKeysRead)
	require.Equal(t, 401, infraerrors.Code(err))

	writeConfig(service.UserAppScopeKeysRead)
	updated, err := service.NewUserAppIntegrationService(store, users)
	require.NoError(t, err)
	for _, scope := range []string{service.UserAppScopeChatInvoke, service.UserAppScopeKeysRead} {
		_, err = updated.Authorize(ctx, "example-chat", chatSecret, exchanged.GrantToken, scope)
		require.Equal(t, 403, infraerrors.Code(err))
	}
	users.active = false
	_, err = integration.Authorize(ctx, "example-chat", chatSecret, exchanged.GrantToken, service.UserAppScopeChatInvoke)
	require.Equal(t, 403, infraerrors.Code(err))
	users.active = true
	require.NoError(t, integration.Revoke(ctx, "example-chat", chatSecret, exchanged.GrantToken))
	require.NoError(t, integration.Revoke(ctx, "example-chat", chatSecret, exchanged.GrantToken))
	_, err = integration.Authorize(ctx, "example-chat", chatSecret, exchanged.GrantToken, service.UserAppScopeChatInvoke)
	require.Equal(t, 401, infraerrors.Code(err))
}
