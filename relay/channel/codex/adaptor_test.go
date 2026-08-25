package codex

import (
	"encoding/json"
	"testing"

	"github.com/55gY/new-api-lite/dto"
	relaycommon "github.com/55gY/new-api-lite/relay/common"
	relayconstant "github.com/55gY/new-api-lite/relay/constant"
)

func TestConvertOpenAIResponsesRequestDropsPenalties(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{},
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:            "gpt-5-codex",
		Input:            json.RawMessage(`"hello"`),
		FrequencyPenalty: json.RawMessage(`1.5`),
		PresencePenalty:  json.RawMessage(`0`),
	})
	if err != nil {
		t.Fatalf("convert request: %v", err)
	}
	request, ok := converted.(dto.OpenAIResponsesRequest)
	if !ok {
		t.Fatalf("converted type = %T, want dto.OpenAIResponsesRequest", converted)
	}
	if request.FrequencyPenalty != nil || request.PresencePenalty != nil {
		t.Fatalf("Codex request retained penalties: frequency=%s presence=%s", request.FrequencyPenalty, request.PresencePenalty)
	}
}
