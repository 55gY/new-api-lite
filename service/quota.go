package service

import (
	"fmt"
	"time"

	"github.com/55gY/new-api-lite/dto"
	"github.com/55gY/new-api-lite/logger"
	"github.com/55gY/new-api-lite/model"
	relaycommon "github.com/55gY/new-api-lite/relay/common"

	"github.com/gin-gonic/gin"
)

type TokenDetails struct {
	TextTokens  int
	AudioTokens int
}

func PreWssConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.RealtimeUsage) error {
	return nil
}

func PostWssConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, modelName string,
	usage *dto.RealtimeUsage, extraContent string) {

	useTimeSeconds := time.Now().Unix() - relayInfo.StartTime.Unix()
	tokenName := ctx.GetString("token_name")

	totalTokens := usage.TotalTokens
	logContent := ""

	if totalTokens == 0 {
		logContent = "可能是上游超时"
		logger.LogError(ctx, fmt.Sprintf("total tokens is 0, userId %d, channelId %d, tokenId %d, model %s", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, modelName))
	} else {
		model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, 0)
		model.RecordChannelUsedQuota(relayInfo.ChannelId, totalTokens)
	}

	logModel := modelName
	if extraContent != "" {
		logContent += ", " + extraContent
	}
	other := GenerateWssOtherInfo(ctx, relayInfo, usage)
	model.RecordUsageLog(ctx, relayInfo.UserId, model.RecordUsageLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		ModelName:        logModel,
		TokenName:        tokenName,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeSeconds:   int(useTimeSeconds),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
	})
}
