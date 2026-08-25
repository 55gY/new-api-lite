package oaichat

import (
	"encoding/json"
	"testing"

	"github.com/55gY/new-api-lite/dto"
)

func TestChatCompletionsRequestToResponsesRequestPreservesCompatibilityFields(t *testing.T) {
	frequencyPenalty := 0.0
	presencePenalty := 1.5
	promptCacheKey := "session-\"quoted\"\\path\n世界"

	got, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model:            "gpt-test",
		Messages:         []dto.Message{{Role: "user", Content: "hello"}},
		FrequencyPenalty: &frequencyPenalty,
		PresencePenalty:  &presencePenalty,
		PromptCacheKey:   promptCacheKey,
	})
	if err != nil {
		t.Fatalf("convert request: %v", err)
	}
	if string(got.FrequencyPenalty) != "0" {
		t.Fatalf("frequency_penalty = %s, want 0", got.FrequencyPenalty)
	}
	if string(got.PresencePenalty) != "1.5" {
		t.Fatalf("presence_penalty = %s, want 1.5", got.PresencePenalty)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal responses request: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal responses request: %v", err)
	}
	if gotKey, ok := payload["prompt_cache_key"].(string); !ok || gotKey != promptCacheKey {
		t.Fatalf("prompt_cache_key = %#v, want %q", payload["prompt_cache_key"], promptCacheKey)
	}
	if value, exists := payload["frequency_penalty"]; !exists || value.(float64) != 0 {
		t.Fatalf("frequency_penalty = %#v, want explicit zero", value)
	}
	if value, exists := payload["presence_penalty"]; !exists || value.(float64) != 1.5 {
		t.Fatalf("presence_penalty = %#v, want 1.5", value)
	}
}

func TestChatCompletionsRequestToResponsesRequestOmitsUnsetCompatibilityFields(t *testing.T) {
	got, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model:    "gpt-test",
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("convert request: %v", err)
	}
	if got.FrequencyPenalty != nil || got.PresencePenalty != nil || got.PromptCacheKey != nil {
		t.Fatalf("unset fields were injected: frequency=%s presence=%s cache=%s", got.FrequencyPenalty, got.PresencePenalty, got.PromptCacheKey)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal responses request: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal responses request: %v", err)
	}
	for _, key := range []string{"frequency_penalty", "presence_penalty", "prompt_cache_key"} {
		if _, exists := payload[key]; exists {
			t.Fatalf("serialized request unexpectedly contains %s: %s", key, encoded)
		}
	}
}
