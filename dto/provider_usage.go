package dto

const (
	ProviderUsageSourceClaudeMessages = "claude_messages"
	ProviderUsageSourceGeminiChat     = "gemini_chat"
	ProviderUsageSourceOAIChat        = "oai_chat"
	ProviderUsageSourceOAIResponses   = "oai_responses"

	ProviderUsageSemanticAnthropic = "anthropic"
	ProviderUsageSemanticGemini    = "gemini"
	ProviderUsageSemanticOpenAI    = "openai"
)

type ProviderUsage struct {
	Source              string               `json:"source,omitempty"`
	Semantic            string               `json:"semantic,omitempty"`
	Estimated           bool                 `json:"estimated,omitempty"`
	OpenAIUsage         *Usage               `json:"openai_usage,omitempty"`
	ClaudeUsage         *ClaudeUsage         `json:"claude_usage,omitempty"`
	GeminiUsageMetadata *GeminiUsageMetadata `json:"gemini_usage_metadata,omitempty"`
}

func NewClaudeMessagesProviderUsage(usage *ClaudeUsage) *ProviderUsage {
	if !HasClaudeUsageTokens(usage) {
		return nil
	}
	return &ProviderUsage{
		Source:      ProviderUsageSourceClaudeMessages,
		Semantic:    ProviderUsageSemanticAnthropic,
		ClaudeUsage: cloneClaudeUsage(usage),
	}
}

// HasClaudeUsageTokens mirrors HasOpenAIUsageTokens/HasGeminiUsageMetadataTokens:
// an all-zero ClaudeUsage must not become a ProviderUsage, otherwise it would take
// precedence during settlement and zero out a non-zero top-level usage.
func HasClaudeUsageTokens(usage *ClaudeUsage) bool {
	if usage == nil {
		return false
	}
	if usage.InputTokens != 0 ||
		usage.OutputTokens != 0 ||
		usage.CacheCreationInputTokens != 0 ||
		usage.CacheReadInputTokens != 0 ||
		usage.ClaudeCacheCreation5mTokens != 0 ||
		usage.ClaudeCacheCreation1hTokens != 0 {
		return true
	}
	if usage.CacheCreation != nil &&
		(usage.CacheCreation.Ephemeral5mInputTokens != 0 || usage.CacheCreation.Ephemeral1hInputTokens != 0) {
		return true
	}
	return false
}

func NewOpenAIChatProviderUsage(usage *Usage) *ProviderUsage {
	return newOpenAIProviderUsage(ProviderUsageSourceOAIChat, usage)
}

func NewOpenAIResponsesProviderUsage(usage *Usage) *ProviderUsage {
	return newOpenAIProviderUsage(ProviderUsageSourceOAIResponses, usage)
}

func newOpenAIProviderUsage(source string, usage *Usage) *ProviderUsage {
	if !HasOpenAIUsageTokens(usage) {
		return nil
	}
	return &ProviderUsage{
		Source:      source,
		Semantic:    ProviderUsageSemanticOpenAI,
		OpenAIUsage: cloneOpenAIUsage(usage),
	}
}

func HasOpenAIUsageTokens(usage *Usage) bool {
	if usage == nil {
		return false
	}
	if usage.PromptTokens != 0 ||
		usage.CompletionTokens != 0 ||
		usage.TotalTokens != 0 ||
		usage.InputTokens != 0 ||
		usage.OutputTokens != 0 ||
		usage.PromptCacheHitTokens != 0 ||
		usage.ClaudeCacheCreation5mTokens != 0 ||
		usage.ClaudeCacheCreation1hTokens != 0 {
		return true
	}
	if usage.PromptTokensDetails.CachedTokens != 0 ||
		usage.PromptTokensDetails.CachedCreationTokens != 0 ||
		usage.PromptTokensDetails.CacheWriteTokens != 0 ||
		usage.PromptTokensDetails.TextTokens != 0 ||
		usage.PromptTokensDetails.ImageTokens != 0 ||
		usage.PromptTokensDetails.AudioTokens != 0 {
		return true
	}
	if usage.CompletionTokenDetails.ReasoningTokens != 0 ||
		usage.CompletionTokenDetails.TextTokens != 0 ||
		usage.CompletionTokenDetails.ImageTokens != 0 ||
		usage.CompletionTokenDetails.AudioTokens != 0 {
		return true
	}
	return usage.InputTokensDetails != nil
}

func NewGeminiChatProviderUsage(metadata *GeminiUsageMetadata) *ProviderUsage {
	return newGeminiChatProviderUsage(metadata, false)
}

func NewEstimatedGeminiChatProviderUsage(usage *Usage) *ProviderUsage {
	if usage == nil {
		return nil
	}
	totalTokens := usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return newGeminiChatProviderUsage(&GeminiUsageMetadata{
		PromptTokenCount:     usage.PromptTokens,
		CandidatesTokenCount: usage.CompletionTokens,
		TotalTokenCount:      totalTokens,
	}, true)
}

func newGeminiChatProviderUsage(metadata *GeminiUsageMetadata, estimated bool) *ProviderUsage {
	if !HasGeminiUsageMetadataTokens(metadata) {
		return nil
	}
	usageMetadata := cloneGeminiUsageMetadata(*metadata)
	return &ProviderUsage{
		Source:              ProviderUsageSourceGeminiChat,
		Semantic:            ProviderUsageSemanticGemini,
		Estimated:           estimated,
		GeminiUsageMetadata: &usageMetadata,
	}
}

func CloneProviderUsage(usage *ProviderUsage) *ProviderUsage {
	if usage == nil {
		return nil
	}
	clone := *usage
	clone.OpenAIUsage = cloneOpenAIUsage(usage.OpenAIUsage)
	clone.ClaudeUsage = cloneClaudeUsage(usage.ClaudeUsage)
	if usage.GeminiUsageMetadata != nil {
		metadata := cloneGeminiUsageMetadata(*usage.GeminiUsageMetadata)
		clone.GeminiUsageMetadata = &metadata
	}
	return &clone
}

func cloneOpenAIUsage(usage *Usage) *Usage {
	if usage == nil {
		return nil
	}
	clone := *usage
	clone.ProviderUsage = nil
	if usage.InputTokensDetails != nil {
		inputTokensDetails := *usage.InputTokensDetails
		clone.InputTokensDetails = &inputTokensDetails
	}
	return &clone
}

func cloneClaudeUsage(usage *ClaudeUsage) *ClaudeUsage {
	if usage == nil {
		return nil
	}
	clone := *usage
	clone.ProviderUsage = nil
	if usage.CacheCreation != nil {
		cacheCreation := *usage.CacheCreation
		clone.CacheCreation = &cacheCreation
	}
	if usage.ServerToolUse != nil {
		serverToolUse := *usage.ServerToolUse
		clone.ServerToolUse = &serverToolUse
	}
	return &clone
}

func cloneGeminiUsageMetadata(metadata GeminiUsageMetadata) GeminiUsageMetadata {
	metadata.PromptTokensDetails = append([]GeminiPromptTokensDetails{}, metadata.PromptTokensDetails...)
	metadata.ToolUsePromptTokensDetails = append([]GeminiPromptTokensDetails{}, metadata.ToolUsePromptTokensDetails...)
	metadata.CandidatesTokensDetails = append([]GeminiPromptTokensDetails{}, metadata.CandidatesTokensDetails...)
	metadata.ProviderUsage = nil
	return metadata
}

func HasGeminiUsageMetadataTokens(metadata *GeminiUsageMetadata) bool {
	if metadata == nil {
		return false
	}
	if metadata.PromptTokenCount != 0 ||
		metadata.ToolUsePromptTokenCount != 0 ||
		metadata.CandidatesTokenCount != 0 ||
		metadata.TotalTokenCount != 0 ||
		metadata.ThoughtsTokenCount != 0 ||
		metadata.CachedContentTokenCount != 0 {
		return true
	}
	for _, detail := range metadata.PromptTokensDetails {
		if detail.TokenCount != 0 {
			return true
		}
	}
	for _, detail := range metadata.ToolUsePromptTokensDetails {
		if detail.TokenCount != 0 {
			return true
		}
	}
	for _, detail := range metadata.CandidatesTokensDetails {
		if detail.TokenCount != 0 {
			return true
		}
	}
	return false
}
