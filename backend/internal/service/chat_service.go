package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var ErrChatAPIKeyNotFound = infraerrors.NotFound("CHAT_API_KEY_NOT_FOUND", "no usable API key found for chat")

type ChatModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type ChatService struct {
	apiKeyService  *APIKeyService
	gatewayService *GatewayService
}

func NewChatService(apiKeyService *APIKeyService, gatewayService *GatewayService) *ChatService {
	return &ChatService{apiKeyService: apiKeyService, gatewayService: gatewayService}
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
		return loaded, nil
	}
	return nil, ErrChatAPIKeyNotFound
}

func (s *ChatService) ListModels(ctx context.Context, userID int64) ([]ChatModel, error) {
	return s.ListChatModels(ctx, userID)
}

func (s *ChatService) ListChatModels(ctx context.Context, userID int64) ([]ChatModel, error) {
	key, err := s.SelectDefaultAPIKey(ctx, userID)
	if err != nil {
		return nil, err
	}

	if key.Group != nil && s.gatewayService != nil {
		groupID := key.Group.ID
		if models := s.gatewayService.GetAvailableModels(ctx, &groupID, ""); len(models) > 0 {
			return chatModelsFromIDs(models, key.Group.Platform, false), nil
		}
	}

	if key.Group != nil {
		switch key.Group.Platform {
		case PlatformOpenAI:
			return chatModelsFromOpenAI(openai.DefaultModels, false), nil
		case PlatformAnthropic:
			return chatModelsFromClaude(claude.DefaultModels), nil
		}
		if key.Group.DefaultMappedModel != "" {
			return []ChatModel{{ID: key.Group.DefaultMappedModel, Name: key.Group.DefaultMappedModel, Provider: key.Group.Platform}}, nil
		}
	}

	return chatModelsFromOpenAI(openai.DefaultModels, false), nil
}

func (s *ChatService) ListImageModels(ctx context.Context, userID int64) ([]ChatModel, error) {
	key, err := s.SelectDefaultAPIKey(ctx, userID)
	if err != nil {
		return nil, err
	}

	if key.Group != nil && s.gatewayService != nil {
		groupID := key.Group.ID
		if models := s.gatewayService.GetAvailableModels(ctx, &groupID, ""); len(models) > 0 {
			return chatModelsFromIDs(models, key.Group.Platform, true), nil
		}
	}

	if key.Group != nil && key.Group.Platform == PlatformOpenAI {
		return chatModelsFromOpenAI(openai.DefaultModels, true), nil
	}

	return nil, nil
}

func chatModelsFromIDs(ids []string, provider string, imageModels bool) []ChatModel {
	models := make([]ChatModel, 0, len(ids))
	for _, id := range ids {
		if provider == PlatformOpenAI && isOpenAIImageModelID(id) != imageModels {
			continue
		}
		models = append(models, ChatModel{ID: id, Name: id, Provider: provider})
	}
	return models
}

func chatModelsFromOpenAI(models []openai.Model, imageModels bool) []ChatModel {
	out := make([]ChatModel, 0, len(models))
	for _, model := range models {
		if isOpenAIImageModelID(model.ID) != imageModels {
			continue
		}
		name := model.DisplayName
		if name == "" {
			name = model.ID
		}
		out = append(out, ChatModel{ID: model.ID, Name: name, Provider: PlatformOpenAI})
	}
	return out
}

func isOpenAIImageModelID(modelID string) bool {
	return strings.HasPrefix(strings.ToLower(modelID), "gpt-image-")
}

func chatModelsFromClaude(models []claude.Model) []ChatModel {
	out := make([]ChatModel, 0, len(models))
	for _, model := range models {
		name := model.DisplayName
		if name == "" {
			name = model.ID
		}
		out = append(out, ChatModel{ID: model.ID, Name: name, Provider: PlatformAnthropic})
	}
	return out
}
