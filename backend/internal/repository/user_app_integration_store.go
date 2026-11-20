package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type userAppIntegrationStore struct {
	rdb *redis.Client
}

func NewUserAppIntegrationStore(rdb *redis.Client) service.UserAppIntegrationStore {
	return &userAppIntegrationStore{rdb: rdb}
}

const (
	userAppTicketPrefix = "user-app-integration:ticket:"
	userAppGrantPrefix  = "user-app-integration:grant:"
)

func (s *userAppIntegrationStore) PutTicket(ctx context.Context, appID, digest string, grant *service.UserAppIntegrationGrant, ttl time.Duration) error {
	return s.put(ctx, userAppTicketPrefix+appID+":"+digest, grant, ttl)
}

func (s *userAppIntegrationStore) ConsumeTicket(ctx context.Context, appID, digest string) (*service.UserAppIntegrationGrant, error) {
	key := userAppTicketPrefix + appID + ":" + digest
	return decodeUserAppGrant(s.rdb.GetDel(ctx, key).Result())
}

func (s *userAppIntegrationStore) PutGrant(ctx context.Context, appID, digest string, grant *service.UserAppIntegrationGrant, ttl time.Duration) error {
	return s.put(ctx, userAppGrantPrefix+appID+":"+digest, grant, ttl)
}

func (s *userAppIntegrationStore) GetGrant(ctx context.Context, appID, digest string) (*service.UserAppIntegrationGrant, error) {
	return decodeUserAppGrant(s.rdb.Get(ctx, userAppGrantPrefix+appID+":"+digest).Result())
}

func (s *userAppIntegrationStore) RevokeGrant(ctx context.Context, appID, digest string) error {
	return s.rdb.Del(ctx, userAppGrantPrefix+appID+":"+digest).Err()
}

func (s *userAppIntegrationStore) put(ctx context.Context, key string, grant *service.UserAppIntegrationGrant, ttl time.Duration) error {
	payload, err := json.Marshal(grant)
	if err != nil {
		return err
	}
	stored, err := s.rdb.SetNX(ctx, key, payload, ttl).Result()
	if err == nil && !stored {
		return errors.New("user app integration credential collision")
	}
	return err
}

func decodeUserAppGrant(payload string, err error) (*service.UserAppIntegrationGrant, error) {
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var grant service.UserAppIntegrationGrant
	if err := json.Unmarshal([]byte(payload), &grant); err != nil {
		return nil, errors.New("invalid user app integration credential record")
	}
	return &grant, nil
}
