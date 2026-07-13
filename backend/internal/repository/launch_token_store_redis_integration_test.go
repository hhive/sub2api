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

func TestLaunchTokenStore_RedisIntegration(t *testing.T) {
	ctx := context.Background()
	rdb := newLaunchRedisIntegrationClient(t)
	store := NewLaunchTokenStore(rdb)

	token := "launch-token-integration"
	require.NoError(t, rdb.Del(ctx, launchTokenRedisKey(token)).Err())
	t.Cleanup(func() {
		_ = rdb.Del(ctx, launchTokenRedisKey(token)).Err()
	})

	data := &service.LaunchTokenData{UserID: 42, APIKeyID: 1001}
	require.NoError(t, store.StoreLaunchToken(ctx, token, data, 60*time.Second))

	ttl, err := rdb.TTL(ctx, launchTokenRedisKey(token)).Result()
	require.NoError(t, err)
	require.GreaterOrEqual(t, ttl, time.Second)
	require.LessOrEqual(t, ttl, 60*time.Second)

	stored, err := store.ConsumeLaunchToken(ctx, token)
	require.NoError(t, err)
	require.Equal(t, data, stored)

	missing, err := store.ConsumeLaunchToken(ctx, token)
	require.NoError(t, err)
	require.Nil(t, missing)
}

func TestLaunchTokenStore_RedisIntegration_InvalidJSONReturnsError(t *testing.T) {
	ctx := context.Background()
	rdb := newLaunchRedisIntegrationClient(t)
	store := NewLaunchTokenStore(rdb)

	token := "launch-token-broken-json"
	require.NoError(t, rdb.Set(ctx, launchTokenRedisKey(token), "{", time.Minute).Err())
	t.Cleanup(func() {
		_ = rdb.Del(ctx, launchTokenRedisKey(token)).Err()
	})

	data, err := store.ConsumeLaunchToken(ctx, token)
	require.Nil(t, data)
	require.Error(t, err)
	missing, err := store.ConsumeLaunchToken(ctx, token)
	require.NoError(t, err)
	require.Nil(t, missing)
}

func newLaunchRedisIntegrationClient(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("LAUNCH_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     os.Getenv("LAUNCH_REDIS_PASSWORD"),
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
		t.Skipf("Redis unavailable at %s; set LAUNCH_REDIS_ADDR to run this test: %v", addr, err)
	}

	return rdb
}
