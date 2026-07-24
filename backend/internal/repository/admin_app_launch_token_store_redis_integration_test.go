//go:build redis_integration

package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAdminAppLaunchTokenStore_RedisIntegration_AtomicConsumeTTLAndBadJSON(t *testing.T) {
	ctx := context.Background()
	rdb := newLaunchRedisIntegrationClient(t)
	store := NewAdminAppLaunchTokenStore(rdb)
	token := "admin-app-launch-integration"
	key := adminAppLaunchTokenRedisKey(token)
	require.NoError(t, rdb.Del(ctx, key).Err())
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	data := &service.AdminAppLaunchTokenData{Version: 1, Purpose: "admin_sso", AppID: "media-management", UserID: 42, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Minute).Unix()}
	require.NoError(t, store.StoreAdminAppLaunchToken(ctx, token, data, time.Minute))
	ttl, err := rdb.TTL(ctx, key).Result()
	require.NoError(t, err)
	require.Greater(t, ttl, time.Duration(0))
	require.LessOrEqual(t, ttl, time.Minute)

	var wg sync.WaitGroup
	results := make(chan *service.AdminAppLaunchTokenData, 2)
	errors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, consumeErr := store.ConsumeAdminAppLaunchToken(ctx, token)
			results <- result
			errors <- consumeErr
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	consumed := 0
	for consumeErr := range errors {
		require.NoError(t, consumeErr)
	}
	for result := range results {
		if result != nil {
			consumed++
		}
	}
	require.Equal(t, 1, consumed)

	require.NoError(t, rdb.Set(ctx, key, "{", time.Minute).Err())
	result, err := store.ConsumeAdminAppLaunchToken(ctx, token)
	require.Nil(t, result)
	require.Error(t, err)
	exists, err := rdb.Exists(ctx, key).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}
