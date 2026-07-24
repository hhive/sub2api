package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const adminAppLaunchTokenKeyPrefix = "admin-app-launch:token:"

var consumeAdminAppLaunchTokenScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if value then
  redis.call('DEL', KEYS[1])
end
return value
`)

type adminAppLaunchTokenStore struct{ rdb *redis.Client }

func NewAdminAppLaunchTokenStore(rdb *redis.Client) service.AdminAppLaunchTokenStore {
	return &adminAppLaunchTokenStore{rdb: rdb}
}

func (s *adminAppLaunchTokenStore) StoreAdminAppLaunchToken(ctx context.Context, token string, data *service.AdminAppLaunchTokenData, ttl time.Duration) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, adminAppLaunchTokenRedisKey(token), payload, ttl).Err()
}

func (s *adminAppLaunchTokenStore) ConsumeAdminAppLaunchToken(ctx context.Context, token string) (*service.AdminAppLaunchTokenData, error) {
	payload, err := consumeAdminAppLaunchTokenScript.Run(ctx, s.rdb, []string{adminAppLaunchTokenRedisKey(token)}).Text()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var data service.AdminAppLaunchTokenData
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func adminAppLaunchTokenRedisKey(token string) string {
	return fmt.Sprintf("%s%s", adminAppLaunchTokenKeyPrefix, token)
}
