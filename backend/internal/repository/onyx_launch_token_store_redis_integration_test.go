//go:build redis_integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestOnyxLaunchTokenStore_RedisIntegration(t *testing.T) {
	ctx := context.Background()
	rdb := newOnyxRedisIntegrationClient(t)
	store := NewOnyxLaunchTokenStore(rdb)

	token := "onyx-launch-token-integration"
	require.NoError(t, store.DeleteLaunchToken(ctx, token))
	t.Cleanup(func() {
		_ = store.DeleteLaunchToken(ctx, token)
	})

	data := &service.OnyxLaunchTokenData{UserID: 42, APIKeyID: 1001}
	require.NoError(t, store.StoreLaunchToken(ctx, token, data, 60*time.Second))

	stored, err := store.GetLaunchToken(ctx, token)
	require.NoError(t, err)
	require.Equal(t, data, stored)

	ttl, err := rdb.TTL(ctx, onyxLaunchTokenRedisKey(token)).Result()
	require.NoError(t, err)
	require.GreaterOrEqual(t, ttl, time.Second)
	require.LessOrEqual(t, ttl, 60*time.Second)

	require.NoError(t, store.DeleteLaunchToken(ctx, token))

	missing, err := store.GetLaunchToken(ctx, token)
	require.NoError(t, err)
	require.Nil(t, missing)
}

func TestOnyxLaunchTokenStore_RedisIntegration_InvalidJSONReturnsError(t *testing.T) {
	ctx := context.Background()
	rdb := newOnyxRedisIntegrationClient(t)
	store := NewOnyxLaunchTokenStore(rdb)

	token := "onyx-launch-token-broken-json"
	require.NoError(t, rdb.Set(ctx, onyxLaunchTokenRedisKey(token), "{", time.Minute).Err())
	t.Cleanup(func() {
		_ = store.DeleteLaunchToken(ctx, token)
	})

	data, err := store.GetLaunchToken(ctx, token)
	require.Nil(t, data)
	require.Error(t, err)
}

func newOnyxRedisIntegrationClient(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("ONYX_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     os.Getenv("ONYX_REDIS_PASSWORD"),
		DB:           15,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable at %s; set ONYX_REDIS_ADDR to run this test: %v", addr, err)
	}

	return rdb
}
