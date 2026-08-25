package oaichat

import (
	"encoding/json"
	"testing"

	"github.com/55gY/new-api-lite/dto"
)

func TestOpenAIChatRequestToClaudeMessagesNormalizesFunctionToolSchema(t *testing.T) {
	maxTokens := uint(16)
	tests := []struct {
		name       string
		parameters any
		wantSchema map[string]any
	}{
		{
			name:       "omitted parameters",
			parameters: nil,
			wantSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			name: "missing type and properties",
			parameters: map[string]any{
				"additionalProperties": false,
			},
			wantSchema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			name: "non-string type",
			parameters: map[string]any{
				"type":       123,
				"properties": map[string]any{},
			},
			wantSchema: map[string]any{
				"type":       123,
				"properties": map[string]any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OpenAIChatRequestToClaudeMessages(nil, dto.GeneralOpenAIRequest{
				Model:     "claude-test",
				MaxTokens: &maxTokens,
				Messages:  []dto.Message{{Role: "user", Content: "Call the tool."}},
				Tools: []dto.ToolCallRequest{{
					Type: "function",
					Function: dto.FunctionRequest{
						Name:        "get_current_time",
						Description: "Get the current time",
						Parameters:  tt.parameters,
					},
				}},
			})
			if err != nil {
				t.Fatalf("convert request: %v", err)
			}
			tools, ok := got.Tools.([]any)
			if !ok || len(tools) != 1 {
				t.Fatalf("tools = %#v, want one Claude tool", got.Tools)
			}
			tool, ok := tools[0].(*dto.Tool)
			if !ok {
				t.Fatalf("tool type = %T, want *dto.Tool", tools[0])
			}
			if tool.Name != "get_current_time" {
				t.Fatalf("tool name = %q", tool.Name)
			}
			if !schemasEqual(tool.InputSchema, tt.wantSchema) {
				t.Fatalf("input schema = %#v, want %#v", tool.InputSchema, tt.wantSchema)
			}
		})
	}
}

func TestOpenAIChatRequestToClaudeMessagesOmitsEmptyTools(t *testing.T) {
	maxTokens := uint(16)
	got, err := OpenAIChatRequestToClaudeMessages(nil, dto.GeneralOpenAIRequest{
		Model:     "claude-test",
		MaxTokens: &maxTokens,
		Messages:  []dto.Message{{Role: "user", Content: "hello"}},
		Tools:     []dto.ToolCallRequest{},
	})
	if err != nil {
		t.Fatalf("convert request: %v", err)
	}
	if got.Tools != nil {
		t.Fatalf("tools = %#v, want nil", got.Tools)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if _, exists := payload["tools"]; exists {
		t.Fatalf("serialized request unexpectedly contains tools: %s", encoded)
	}
}

func schemasEqual(got map[string]interface{}, want map[string]any) bool {
	gotJSON, err := json.Marshal(got)
	if err != nil {
		return false
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		return false
	}
	return string(gotJSON) == string(wantJSON)
}
