package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func (s *OpenAIGatewayService) injectDeepSeekSystemPromptForChatCompletions(ctx context.Context, account *Account, body []byte) []byte {
	if account == nil || account.Platform != PlatformDeepseek {
		return body
	}
	prompt := s.deepSeekSystemPrompt(ctx)
	if prompt == "" {
		return body
	}
	return injectOpenAIChatSystemPrompt(body, prompt)
}

func (s *OpenAIGatewayService) injectDeepSeekSystemPromptForResponses(ctx context.Context, account *Account, body []byte) []byte {
	if account == nil || account.Platform != PlatformDeepseek {
		return body
	}
	prompt := s.deepSeekSystemPrompt(ctx)
	if prompt == "" {
		return body
	}
	return injectResponsesInstructions(body, prompt)
}

func (s *OpenAIGatewayService) injectDeepSeekSystemPromptForAnthropic(ctx context.Context, account *Account, body []byte) []byte {
	if account == nil || account.Platform != PlatformDeepseek {
		return body
	}
	prompt := s.deepSeekSystemPrompt(ctx)
	if prompt == "" {
		return body
	}
	return injectAnthropicSystemPrompt(body, prompt)
}

func (s *OpenAIGatewayService) deepSeekSystemPrompt(ctx context.Context) string {
	if s == nil || s.settingService == nil {
		return ""
	}
	return strings.TrimSpace(s.settingService.GetDeepSeekSystemPrompt(ctx))
}

func injectOpenAIChatSystemPrompt(body []byte, prompt string) []byte {
	prompt = strings.TrimSpace(prompt)
	messages := gjson.GetBytes(body, "messages")
	if prompt == "" || !messages.IsArray() {
		return body
	}
	items := make([][]byte, 0, len(messages.Array())+1)
	systemAppended := false
	messages.ForEach(func(_, item gjson.Result) bool {
		if !systemAppended && item.Get("role").String() == "system" {
			next, err := appendOpenAIChatSystemContent([]byte(item.Raw), item.Get("content"), prompt)
			if err == nil {
				items = append(items, next)
				systemAppended = true
				return true
			}
		}
		items = append(items, []byte(item.Raw))
		return true
	})
	if !systemAppended {
		msg, err := json.Marshal(map[string]any{
			"role":    "system",
			"content": prompt,
		})
		if err != nil {
			return body
		}
		items = append([][]byte{msg}, items...)
	}
	if next, ok := setJSONRawBytes(body, "messages", buildJSONArrayRaw(items)); ok {
		return next
	}
	return body
}

func appendOpenAIChatSystemContent(message []byte, content gjson.Result, prompt string) ([]byte, error) {
	switch content.Type {
	case gjson.String, gjson.Null:
		existing := content.String()
		if existing != "" {
			existing += "\n\n"
		}
		return sjson.SetBytes(message, "content", existing+prompt)
	case gjson.JSON:
		if !content.IsArray() {
			return nil, errors.New("system content must be a string or array")
		}
		textBlock, err := json.Marshal(map[string]any{"type": "text", "text": prompt})
		if err != nil {
			return nil, err
		}
		parts := make([][]byte, 0, len(content.Array())+1)
		content.ForEach(func(_, part gjson.Result) bool {
			parts = append(parts, []byte(part.Raw))
			return true
		})
		parts = append(parts, textBlock)
		return sjson.SetRawBytes(message, "content", buildJSONArrayRaw(parts))
	default:
		return nil, errors.New("system content must be a string or array")
	}
}

func injectResponsesInstructions(body []byte, prompt string) []byte {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return body
	}
	existing := strings.TrimSpace(gjson.GetBytes(body, "instructions").String())
	nextInstructions := existing
	if existing != "" {
		nextInstructions += "\n\n"
	}
	nextInstructions += prompt
	next, err := sjson.SetBytes(body, "instructions", nextInstructions)
	if err != nil {
		return body
	}
	return next
}

func injectAnthropicSystemPrompt(body []byte, prompt string) []byte {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return body
	}
	block, err := json.Marshal(map[string]any{
		"type": "text",
		"text": prompt,
	})
	if err != nil {
		return body
	}
	system := gjson.GetBytes(body, "system")
	switch {
	case !system.Exists():
		if next, ok := setJSONRawBytes(body, "system", buildJSONArrayRaw([][]byte{block})); ok {
			return next
		}
	case system.IsArray():
		items := make([][]byte, 0, len(system.Array())+1)
		system.ForEach(func(_, item gjson.Result) bool {
			items = append(items, []byte(item.Raw))
			return true
		})
		items = append(items, block)
		if next, ok := setJSONRawBytes(body, "system", buildJSONArrayRaw(items)); ok {
			return next
		}
	case system.Type == gjson.String:
		existing, err := json.Marshal(map[string]any{
			"type": "text",
			"text": system.String(),
		})
		if err != nil {
			return body
		}
		if next, ok := setJSONRawBytes(body, "system", buildJSONArrayRaw([][]byte{existing, block})); ok {
			return next
		}
	}
	return body
}
