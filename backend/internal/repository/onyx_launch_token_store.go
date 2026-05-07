package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const onyxLaunchTokenKeyPrefix = "onyx:launch:"

type onyxLaunchTokenStore struct {
	rdb *redis.Client
}

func NewOnyxLaunchTokenStore(rdb *redis.Client) service.OnyxLaunchTokenStore {
	return &onyxLaunchTokenStore{rdb: rdb}
}

func (s *onyxLaunchTokenStore) StoreLaunchToken(ctx context.Context, token string, data *service.OnyxLaunchTokenData, ttl time.Duration) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, onyxLaunchTokenRedisKey(token), payload, ttl).Err()
}

func (s *onyxLaunchTokenStore) GetLaunchToken(ctx context.Context, token string) (*service.OnyxLaunchTokenData, error) {
	payload, err := s.rdb.Get(ctx, onyxLaunchTokenRedisKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var data service.OnyxLaunchTokenData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *onyxLaunchTokenStore) DeleteLaunchToken(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, onyxLaunchTokenRedisKey(token)).Err()
}

func onyxLaunchTokenRedisKey(token string) string {
	return fmt.Sprintf("%s%s", onyxLaunchTokenKeyPrefix, token)
}
