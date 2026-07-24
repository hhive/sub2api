//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAdminAppLaunchTokenStore_UsesIndependentPrefix(t *testing.T) {
	require.Equal(t, "admin-app-launch:token:abc", adminAppLaunchTokenRedisKey("abc"))
	require.NotEqual(t, launchTokenRedisKey("abc"), adminAppLaunchTokenRedisKey("abc"))
}

func TestAdminAppLaunchTokenStore_RedisError(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond, ReadTimeout: 50 * time.Millisecond, WriteTimeout: 50 * time.Millisecond})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewAdminAppLaunchTokenStore(rdb)
	err := store.StoreAdminAppLaunchToken(context.Background(), "token", &service.AdminAppLaunchTokenData{UserID: 1}, time.Minute)
	require.Error(t, err)
}

func TestAdminAppLaunchTokenStore_TTLConsumeAndMalformedPayload(t *testing.T) {
	ctx := context.Background()
	miniRedis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewAdminAppLaunchTokenStore(rdb)
	data := &service.AdminAppLaunchTokenData{
		Version: 1, Purpose: "admin_sso", AppID: "media-management", UserID: 42,
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Minute).Unix(),
	}

	require.NoError(t, store.StoreAdminAppLaunchToken(ctx, "valid", data, time.Minute))
	ttl, err := rdb.TTL(ctx, adminAppLaunchTokenRedisKey("valid")).Result()
	require.NoError(t, err)
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, time.Minute)
	consumed, err := store.ConsumeAdminAppLaunchToken(ctx, "valid")
	require.NoError(t, err)
	require.Equal(t, data, consumed)
	consumed, err = store.ConsumeAdminAppLaunchToken(ctx, "valid")
	require.NoError(t, err)
	require.Nil(t, consumed)

	malformedKey := adminAppLaunchTokenRedisKey("malformed")
	require.NoError(t, rdb.Set(ctx, malformedKey, "{", time.Minute).Err())
	consumed, err = store.ConsumeAdminAppLaunchToken(ctx, "malformed")
	require.Error(t, err)
	require.Nil(t, consumed)
	exists, err := rdb.Exists(ctx, malformedKey).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}
