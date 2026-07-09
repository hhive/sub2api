package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var ErrChatAPIKeyNotFound = infraerrors.NotFound("CHAT_API_KEY_NOT_FOUND", "no usable API key found for chat")

type ChatModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type ChatService struct {
	apiKeyService *APIKeyService
}

func NewChatService(apiKeyService *APIKeyService) *ChatService {
	return &ChatService{apiKeyService: apiKeyService}
}

func (s *ChatService) SelectDefaultAPIKey(ctx context.Context, userID int64) (*APIKey, error) {
	keys, _, err := s.apiKeyService.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100, SortBy: "id", SortOrder: pagination.SortOrderDesc}, APIKeyListFilters{})
	if err != nil {
		return nil, err
	}
	for i := range keys {
		key := keys[i]
		if key.Status != StatusAPIKeyActive || key.GroupID == nil {
			continue
		}
		loaded, err := s.apiKeyService.GetByKey(ctx, key.Key)
		if err != nil {
			return nil, err
		}
		if !isDefaultChatAPIKeyUsable(loaded) {
			continue
		}
		return loaded, nil
	}
	return nil, ErrChatAPIKeyNotFound
}

func isDefaultChatAPIKeyUsable(key *APIKey) bool {
	if key == nil || key.Status != StatusAPIKeyActive || key.IsExpired() || key.IsQuotaExhausted() {
		return false
	}
	if key.User == nil || !key.User.IsActive() {
		return false
	}
	if key.GroupID != nil {
		if key.Group == nil || !key.Group.IsActive() {
			return false
		}
	}
	return true
}

func (s *ChatService) ListModels(ctx context.Context, userID int64) ([]ChatModel, error) {
	key, err := s.SelectDefaultAPIKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	model := "gpt-4o-mini"
	provider := "openai"
	if key.Group != nil {
		if key.Group.DefaultMappedModel != "" {
			model = key.Group.DefaultMappedModel
		}
		if key.Group.Platform != "" {
			provider = key.Group.Platform
		}
	}
	return []ChatModel{{ID: model, Name: model, Provider: provider}}, nil
}
