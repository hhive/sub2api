package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

type ChatModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

func ChatModelFromService(model service.ChatModel) ChatModel {
	return ChatModel{
		ID:       model.ID,
		Name:     model.Name,
		Provider: model.Provider,
	}
}

func ChatModelsFromService(models []service.ChatModel) []ChatModel {
	out := make([]ChatModel, 0, len(models))
	for _, model := range models {
		out = append(out, ChatModelFromService(model))
	}
	return out
}
