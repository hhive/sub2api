package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// launchTokenKeyPrefix is shared by LobeHub and the media playgrounds.
const launchTokenKeyPrefix = "launch:token:"

var consumeLaunchTokenScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if value then
  redis.call('DEL', KEYS[1])
end
return value
`)

type launchTokenStore struct {
	rdb *redis.Client
}

func NewLaunchTokenStore(rdb *redis.Client) service.LaunchTokenStore {
	return &launchTokenStore{rdb: rdb}
}

func (s *launchTokenStore) StoreLaunchToken(ctx context.Context, token string, data *service.LaunchTokenData, ttl time.Duration) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, launchTokenRedisKey(token), payload, ttl).Err()
}

func (s *launchTokenStore) ConsumeLaunchToken(ctx context.Context, token string) (*service.LaunchTokenData, error) {
	payload, err := consumeLaunchTokenScript.Run(ctx, s.rdb, []string{launchTokenRedisKey(token)}).Text()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var data service.LaunchTokenData
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func launchTokenRedisKey(token string) string {
	return fmt.Sprintf("%s%s", launchTokenKeyPrefix, token)
}
