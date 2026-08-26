package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/55gY/new-api-lite/dto"
	relaycommon "github.com/55gY/new-api-lite/relay/common"
	"github.com/55gY/new-api-lite/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCalculateTextUsageStatsPreservesClaudeSemantic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	usage := &dto.Usage{
		PromptTokens:                1000,
		CompletionTokens:            200,
		UsageSemantic:               "anthropic",
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}
	info := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-3-7-sonnet",
		StartTime:               time.Now(),
	}

	summary := calculateTextUsageStats(ctx, info, usage)
	require.True(t, summary.IsClaudeUsageSemantic)
	require.Equal(t, "anthropic", summary.UsageSemantic)
	require.Equal(t, 1200, summary.TotalTokens)
	require.Equal(t, 10, summary.CacheCreationTokens5m)
	require.Equal(t, 20, summary.CacheCreationTokens1h)
}

func TestUsageStatsCacheWriteTokens(t *testing.T) {
	require.Equal(t, 50, usageStatsCacheWriteTokens(textUsageStats{
		CacheCreationTokens: 50, CacheCreationTokens5m: 10, CacheCreationTokens1h: 20,
	}))
	require.Equal(t, 30, usageStatsCacheWriteTokens(textUsageStats{
		CacheCreationTokens5m: 10, CacheCreationTokens1h: 20,
	}))
}

func TestCalculateTextUsageStatsPreservesOpenRouterCacheReadTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	usage := &dto.Usage{
		PromptTokens: 2604, CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 2432},
	}
	info := &relaycommon.RelayInfo{OriginModelName: "openai/gpt-4.1", StartTime: time.Now()}

	summary := calculateTextUsageStats(ctx, info, usage)
	require.Equal(t, 2604, summary.PromptTokens)
	require.Equal(t, 2432, summary.CacheTokens)
	require.Equal(t, 2987, summary.TotalTokens)
}
