package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

func isLegacyClaudeDerivedOpenAIUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if relayInfo == nil || usage == nil {
		return false
	}
	if relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return false
	}
	if usage.UsageSource != "" || usage.UsageSemantic != "" {
		return false
	}
	return usage.ClaudeCacheCreation5mTokens > 0 || usage.ClaudeCacheCreation1hTokens > 0
}

func usageSemanticFromUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) string {
	if usage != nil && usage.UsageSemantic != "" {
		return usage.UsageSemantic
	}
	if relayInfo != nil && relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return "anthropic"
	}
	return "openai"
}

func PostTextConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, extraContent []string) {
	originUsage := usage
	if usage == nil {
		extraContent = append(extraContent, "上游没有返回 token 用量")
	}
	if originUsage != nil {
		ObserveChannelAffinityUsageCacheByRelayFormat(ctx, usage, relayInfo.GetFinalRequestRelayFormat())
	}

	stats := calculateTextUsageStats(ctx, relayInfo, usage)

	if stats.WebSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 %d 次", stats.WebSearchCallCount))
	}
	if stats.ClaudeWebSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Claude Web Search 调用 %d 次", stats.ClaudeWebSearchCallCount))
	}
	if stats.FileSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("File Search 调用 %d 次", stats.FileSearchCallCount))
	}
	if stats.AudioTokens > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Audio Input token %d", stats.AudioTokens))
	}

	if stats.TotalTokens == 0 {
		extraContent = append(extraContent, "上游没有返回 token 用量（可能是上游超时）")
		logger.LogError(ctx, fmt.Sprintf("total tokens is 0, userId %d, channelId %d, tokenId %d, model %s", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, stats.ModelName))
	} else {
		model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, 0)
		model.RecordChannelUsedQuota(relayInfo.ChannelId, stats.TotalTokens)
	}

	logModel := stats.ModelName
	if strings.HasPrefix(logModel, "gpt-4-gizmo") {
		logModel = "gpt-4-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", stats.ModelName))
	}
	if strings.HasPrefix(logModel, "gpt-4o-gizmo") {
		logModel = "gpt-4o-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", stats.ModelName))
	}

	logContent := strings.Join(extraContent, ", ")
	other := buildTextUsageOther(ctx, relayInfo, stats, usage)
	model.RecordUsageLog(ctx, relayInfo.UserId, model.RecordUsageLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     stats.PromptTokens,
		CompletionTokens: stats.CompletionTokens,
		ModelName:        logModel,
		TokenName:        stats.TokenName,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeSeconds:   int(stats.UseTimeSeconds),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
	})
	gopool.Go(func() {
		perfmetrics.RecordRelaySample(relayInfo, true, int64(stats.CompletionTokens))
	})
}
