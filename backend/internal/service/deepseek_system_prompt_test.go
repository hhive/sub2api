package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestInjectOpenAIChatSystemPromptAppendsToExistingSystem(t *testing.T) {
	body := []byte(`{"model":"deepseek-chat","messages":[{"role":"system","content":"client"},{"role":"user","content":"hi"}]}`)

	got := injectOpenAIChatSystemPrompt(body, "gateway")

	messages := gjson.GetBytes(got, "messages").Array()
	require.Len(t, messages, 2)
	require.Equal(t, "client\n\ngateway", messages[0].Get("content").String())
}

func TestInjectOpenAIChatSystemPromptCreatesSystemWhenMissing(t *testing.T) {
	body := []byte(`{"model":"deepseek-chat","messages":[{"role":"user","content":"hi"}]}`)

	got := injectOpenAIChatSystemPrompt(body, "gateway")

	messages := gjson.GetBytes(got, "messages").Array()
	require.Len(t, messages, 2)
	require.Equal(t, "system", messages[0].Get("role").String())
	require.Equal(t, "gateway", messages[0].Get("content").String())
}

func TestInjectResponsesInstructionsAppendsExistingInstructions(t *testing.T) {
	body := []byte(`{"model":"deepseek-reasoner","instructions":"client","input":"hi"}`)

	got := injectResponsesInstructions(body, "gateway")

	require.Equal(t, "client\n\ngateway", gjson.GetBytes(got, "instructions").String())
}

func TestInjectAnthropicSystemPromptAppendsStringSystem(t *testing.T) {
	body := []byte(`{"model":"deepseek-chat","system":"client","messages":[{"role":"user","content":"hi"}]}`)

	got := injectAnthropicSystemPrompt(body, "gateway")

	system := gjson.GetBytes(got, "system").Array()
	require.Len(t, system, 2)
	require.Equal(t, "client", system[0].Get("text").String())
	require.Equal(t, "gateway", system[1].Get("text").String())
}

func TestInjectAnthropicSystemPromptSetsMissingSystem(t *testing.T) {
	body := []byte(`{"model":"deepseek-chat","messages":[{"role":"user","content":"hi"}]}`)

	got := injectAnthropicSystemPrompt(body, "gateway")

	system := gjson.GetBytes(got, "system").Array()
	require.Len(t, system, 1)
	require.Equal(t, "gateway", system[0].Get("text").String())
}

func TestOpenAIGatewayDeepSeekSystemPromptOnlyAppliesToDeepSeek(t *testing.T) {
	svc := &OpenAIGatewayService{settingService: &SettingService{}}
	body := []byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)

	got := svc.injectDeepSeekSystemPromptForChatCompletions(
		nil,
		&Account{Platform: PlatformOpenAI},
		body,
	)

	require.JSONEq(t, string(body), string(got))
}
