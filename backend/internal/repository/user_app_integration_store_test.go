//go:build unit

package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestUserAppIntegrationStore(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := NewUserAppIntegrationStore(client)
	ctx := context.Background()
	grant := &service.UserAppIntegrationGrant{UserID: 42, Scopes: []string{service.UserAppScopeKeysRead}}
	require.NoError(t, store.PutTicket(ctx, "first", "digest", grant, time.Minute))
	data, err := store.ConsumeTicket(ctx, "other", "digest")
	require.NoError(t, err)
	require.Nil(t, data)

	var wg sync.WaitGroup
	var consumed atomic.Int64
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := store.ConsumeTicket(ctx, "first", "digest")
			if err != nil {
				t.Error(err)
			}
			if data != nil {
				consumed.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, consumed.Load())

	require.NoError(t, store.PutTicket(ctx, "first", "next-digest", grant, time.Minute))
	server.FastForward(time.Minute)
	data, err = store.ConsumeTicket(ctx, "first", "next-digest")
	require.NoError(t, err)
	require.Nil(t, data)

	require.NoError(t, store.PutGrant(ctx, "first", "digest", grant, time.Hour))
	require.NoError(t, store.RevokeGrant(ctx, "other", "digest"))
	data, err = store.GetGrant(ctx, "first", "digest")
	require.NoError(t, err)
	require.Equal(t, grant, data)
	server.FastForward(time.Hour)
	data, err = store.GetGrant(ctx, "first", "digest")
	require.NoError(t, err)
	require.Nil(t, data)
	require.NoError(t, store.RevokeGrant(ctx, "first", "digest"))
}
