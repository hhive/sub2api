package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestChatServiceSelectsFirstActiveGroupedAPIKeyForUser(t *testing.T) {
	groupID := int64(9)
	repo := &chatAPIKeyRepoStub{
		list: []APIKey{
			{ID: 1, UserID: 42, Key: "sk-disabled", Status: StatusAPIKeyDisabled, GroupID: &groupID},
			{ID: 2, UserID: 42, Key: "sk-no-group", Status: StatusAPIKeyActive},
			{ID: 3, UserID: 42, Key: "sk-chat", Status: StatusAPIKeyActive, GroupID: &groupID},
		},
		byKey: map[string]*APIKey{
			"sk-chat": {
				ID:      3,
				UserID:  42,
				Key:     "sk-chat",
				Status:  StatusAPIKeyActive,
				GroupID: &groupID,
				User:    &User{ID: 42, Status: StatusActive, Balance: 10, Concurrency: 1},
				Group:   &Group{ID: groupID, Name: "default", Status: StatusActive},
			},
		},
	}

	svc := NewChatService(NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil), nil)

	key, err := svc.SelectDefaultAPIKey(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, int64(3), key.ID)
	require.Equal(t, "sk-chat", key.Key)
	require.Equal(t, int64(42), repo.listUserID)
	require.Equal(t, "sk-chat", repo.loadedKey)
}

func TestChatServiceListModelsFallsBackToOpenAIDefaultModels(t *testing.T) {
	groupID := int64(9)
	repo := &chatAPIKeyRepoStub{
		list: []APIKey{
			{ID: 3, UserID: 42, Key: "sk-chat", Status: StatusAPIKeyActive, GroupID: &groupID},
		},
		byKey: map[string]*APIKey{
			"sk-chat": {
				ID:      3,
				UserID:  42,
				Key:     "sk-chat",
				Status:  StatusAPIKeyActive,
				GroupID: &groupID,
				User:    &User{ID: 42, Status: StatusActive, Balance: 10, Concurrency: 1},
				Group:   &Group{ID: groupID, Name: "default", Platform: PlatformOpenAI, Status: StatusActive},
			},
		},
	}

	svc := NewChatService(NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil), nil)

	models, err := svc.ListModels(context.Background(), 42)

	require.NoError(t, err)
	require.NotEmpty(t, models)
	require.Equal(t, openai.DefaultModels[0].ID, models[0].ID)
	require.NotEqual(t, "gpt-4o-mini", models[0].ID)
}

func TestChatServiceListModelsUsesAvailableModelsFromGroupAccounts(t *testing.T) {
	groupID := int64(9)
	apiKeyRepo := &chatAPIKeyRepoStub{
		list: []APIKey{
			{ID: 3, UserID: 42, Key: "sk-chat", Status: StatusAPIKeyActive, GroupID: &groupID},
		},
		byKey: map[string]*APIKey{
			"sk-chat": {
				ID:      3,
				UserID:  42,
				Key:     "sk-chat",
				Status:  StatusAPIKeyActive,
				GroupID: &groupID,
				User:    &User{ID: 42, Status: StatusActive, Balance: 10, Concurrency: 1},
				Group:   &Group{ID: groupID, Name: "default", Platform: PlatformOpenAI, Status: StatusActive},
			},
		},
	}
	accountRepo := &modelsListAccountRepoStub{
		byGroup: map[int64][]Account{
			groupID: {
				{
					ID:       8,
					Platform: PlatformOpenAI,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"gpt-5.4-mini": "gpt-5.4-mini",
						},
					},
				},
			},
		},
	}
	svc := NewChatService(
		NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, nil),
		&GatewayService{accountRepo: accountRepo},
	)

	models, err := svc.ListModels(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, []ChatModel{{ID: "gpt-5.4-mini", Name: "gpt-5.4-mini", Provider: PlatformOpenAI}}, models)
}

func TestChatServiceReturnsNotFoundWhenUserHasNoUsableAPIKey(t *testing.T) {
	repo := &chatAPIKeyRepoStub{
		list: []APIKey{
			{ID: 1, UserID: 42, Key: "sk-no-group", Status: StatusAPIKeyActive},
		},
	}

	svc := NewChatService(NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil), nil)

	key, err := svc.SelectDefaultAPIKey(context.Background(), 42)

	require.Nil(t, key)
	require.ErrorIs(t, err, ErrChatAPIKeyNotFound)
}

type chatAPIKeyRepoStub struct {
	APIKeyRepository
	list       []APIKey
	byKey      map[string]*APIKey
	listUserID int64
	loadedKey  string
}

func (r *chatAPIKeyRepoStub) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	r.listUserID = userID
	return r.list, &pagination.PaginationResult{Total: int64(len(r.list)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (r *chatAPIKeyRepoStub) GetByKeyForAuth(ctx context.Context, key string) (*APIKey, error) {
	r.loadedKey = key
	if r.byKey == nil || r.byKey[key] == nil {
		return nil, ErrAPIKeyNotFound
	}
	return r.byKey[key], nil
}
