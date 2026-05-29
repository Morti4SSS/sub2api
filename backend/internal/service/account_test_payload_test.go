package service

import "testing"

func TestCreateOpenAITestPayloadUsesPrompt(t *testing.T) {
	payload := createOpenAITestPayload("gpt-5.4", true, "random probe prompt")

	input := payload["input"].([]map[string]any)
	content := input[0]["content"].([]map[string]any)
	if got := content[0]["text"]; got != "random probe prompt" {
		t.Fatalf("prompt text = %v, want random probe prompt", got)
	}
}

func TestCreateOpenAITestPayloadDefaultsPrompt(t *testing.T) {
	payload := createOpenAITestPayload("gpt-5.4", true, " ")

	input := payload["input"].([]map[string]any)
	content := input[0]["content"].([]map[string]any)
	if got := content[0]["text"]; got != "hi" {
		t.Fatalf("prompt text = %v, want hi", got)
	}
}
