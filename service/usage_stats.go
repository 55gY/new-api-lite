package service

import (
	"time"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/constant"
	"github.com/55gY/new-api-lite/dto"
	relaycommon "github.com/55gY/new-api-lite/relay/common"
	"github.com/55gY/new-api-lite/types"

	"github.com/gin-gonic/gin"
)

type textUsageStats struct {
	PromptTokens             int
	CompletionTokens         int
	TotalTokens              int
	CacheTokens              int
	CacheCreationTokens      int
	CacheCreationTokens5m    int
	CacheCreationTokens1h    int
	ImageTokens              int
	AudioTokens              int
	ModelName                string
	TokenName                string
	UseTimeSeconds           int64
	IsClaudeUsageSemantic    bool
	UsageSemantic            string
	WebSearchCallCount       int
	ClaudeWebSearchCallCount int
	FileSearchCallCount      int
	Quota                    int
}

func calculateTextUsageStats(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) textUsageStats {
	stats := textUsageStats{
		ModelName:      relayInfo.OriginModelName,
		TokenName:      ctx.GetString("token_name"),
		UseTimeSeconds: time.Now().Unix() - relayInfo.StartTime.Unix(),
		UsageSemantic:  usageSemanticFromUsage(relayInfo, usage),
		Quota:          0,
	}
	stats.IsClaudeUsageSemantic = stats.UsageSemantic == "anthropic"

	if usage == nil {
		usage = &dto.Usage{
			PromptTokens:     relayInfo.GetEstimatePromptTokens(),
			CompletionTokens: 0,
			TotalTokens:      relayInfo.GetEstimatePromptTokens(),
		}
	}

	stats.PromptTokens = usage.PromptTokens
	stats.CompletionTokens = usage.CompletionTokens
	stats.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	stats.CacheTokens = usage.PromptTokensDetails.CachedTokens
	stats.CacheCreationTokens = usage.PromptTokensDetails.CachedCreationTokens
	stats.CacheCreationTokens5m = usage.ClaudeCacheCreation5mTokens
	stats.CacheCreationTokens1h = usage.ClaudeCacheCreation1hTokens
	stats.ImageTokens = usage.PromptTokensDetails.ImageTokens
	stats.AudioTokens = usage.PromptTokensDetails.AudioTokens

	if relayInfo.ResponsesUsageInfo != nil {
		if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists {
			stats.WebSearchCallCount = webSearchTool.CallCount
		}
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch]; exists {
			stats.FileSearchCallCount = fileSearchTool.CallCount
		}
	}
	stats.ClaudeWebSearchCallCount = ctx.GetInt("claude_web_search_requests")

	return stats
}

func usageStatsCacheWriteTokens(stats textUsageStats) int {
	if stats.CacheCreationTokens5m > 0 || stats.CacheCreationTokens1h > 0 {
		splitCacheWriteTokens := stats.CacheCreationTokens5m + stats.CacheCreationTokens1h
		if stats.CacheCreationTokens > splitCacheWriteTokens {
			return stats.CacheCreationTokens
		}
		return splitCacheWriteTokens
	}
	return stats.CacheCreationTokens
}

func buildTextUsageOther(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, stats textUsageStats, usage *dto.Usage) map[string]interface{} {
	var other map[string]interface{}
	if stats.IsClaudeUsageSemantic {
		other = GenerateClaudeOtherInfo(ctx, relayInfo, stats.CacheTokens,
			stats.CacheCreationTokens,
			stats.CacheCreationTokens5m,
			stats.CacheCreationTokens1h)
		other["usage_semantic"] = "anthropic"
	} else {
		other = GenerateTextOtherInfo(ctx, relayInfo, stats.CacheTokens)
	}

	adminRejectReason := common.GetContextKeyString(ctx, constant.ContextKeyAdminRejectReason)
	if adminRejectReason != "" {
		other["reject_reason"] = adminRejectReason
	}
	if stats.ImageTokens != 0 {
		other["image"] = true
		other["image_output"] = stats.ImageTokens
	}
	if stats.WebSearchCallCount > 0 {
		other["web_search"] = true
		other["web_search_call_count"] = stats.WebSearchCallCount
	} else if stats.ClaudeWebSearchCallCount > 0 {
		other["web_search"] = true
		other["web_search_call_count"] = stats.ClaudeWebSearchCallCount
	}
	if stats.FileSearchCallCount > 0 {
		other["file_search"] = true
		other["file_search_call_count"] = stats.FileSearchCallCount
	}
	if stats.AudioTokens > 0 {
		other["audio_input_token_count"] = stats.AudioTokens
	}
	if stats.CacheCreationTokens > 0 {
		other["cache_creation_tokens"] = stats.CacheCreationTokens
	}
	if stats.CacheCreationTokens5m > 0 {
		other["cache_creation_tokens_5m"] = stats.CacheCreationTokens5m
	}
	if stats.CacheCreationTokens1h > 0 {
		other["cache_creation_tokens_1h"] = stats.CacheCreationTokens1h
	}
	if cacheWriteTokens := usageStatsCacheWriteTokens(stats); cacheWriteTokens > 0 {
		other["cache_write_tokens"] = cacheWriteTokens
	}
	if relayInfo.GetFinalRequestRelayFormat() != types.RelayFormatClaude && usage != nil && usage.UsageSource != "" && usage.InputTokens > 0 {
		other["input_tokens_total"] = usage.InputTokens
	}
	return other
}
