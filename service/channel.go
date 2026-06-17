package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

func formatNotifyType(channelId int, status int) string {
	return fmt.Sprintf("%s_%d_%d", dto.NotifyTypeChannelUpdate, channelId, status)
}

// disable & notify
func DisableChannel(channelError types.ChannelError, reason string) {
	common.SysLog(fmt.Sprintf("通道「%s」（#%d）发生错误，准备禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, common.LocalLogPreview(reason)))

	// 检查是否启用自动禁用功能
	if !channelError.AutoBan {
		common.SysLog(fmt.Sprintf("通道「%s」（#%d）未启用自动禁用功能，跳过禁用操作", channelError.ChannelName, channelError.ChannelId))
		return
	}

	success := model.UpdateChannelStatus(channelError.ChannelId, channelError.UsingKey, common.ChannelStatusAutoDisabled, reason)
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被禁用", channelError.ChannelName, channelError.ChannelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, reason)
		NotifyRootUser(formatNotifyType(channelError.ChannelId, common.ChannelStatusAutoDisabled), subject, content)
	}
}

func EnableChannel(channelId int, usingKey string, channelName string) {
	success := model.UpdateChannelStatus(channelId, usingKey, common.ChannelStatusEnabled, "")
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		NotifyRootUser(formatNotifyType(channelId, common.ChannelStatusEnabled), subject, content)
	}
}

func ShouldDisableChannel(err *types.NewAPIError) bool {
	if !common.AutomaticDisableChannelEnabled {
		return false
	}
	if err == nil {
		return false
	}
	if types.IsChannelError(err) {
		return true
	}
	if types.IsSkipRetryError(err) {
		return false
	}
	if operation_setting.ShouldDisableByStatusCode(err.StatusCode) {
		return true
	}

	lowerMessage := strings.ToLower(err.Error())
	search, _ := AcSearch(lowerMessage, operation_setting.AutomaticDisableKeywords, true)
	return search
}

func ShouldEnableChannel(newAPIError *types.NewAPIError, status int) bool {
	if !common.AutomaticEnableChannelEnabled {
		return false
	}
	if newAPIError != nil {
		return false
	}
	if status != common.ChannelStatusAutoDisabled {
		return false
	}
	return true
}

// ShouldDisableModel 判断是否为模型级别错误（如模型不存在、模型不可用等）
func ShouldDisableModel(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	// 基于错误码判断
	if err.GetErrorCode() == types.ErrorCodeModelNotFound {
		return true
	}
	// 基于响应体关键字判断
	lowerMessage := strings.ToLower(err.Error())
	modelErrorKeywords := []string{"model not found", "model is not supported", "this model is not available", "does not exist or is not available", "model_not_found"}
	for _, keyword := range modelErrorKeywords {
		if strings.Contains(lowerMessage, keyword) {
			return true
		}
	}
	return false
}

// DisableModel 禁用指定渠道的特定模型（能力级别）
func DisableModel(channel *model.Channel, modelName string, reason string) {
	if channel == nil || modelName == "" {
		return
	}
	modelName = strings.TrimSpace(modelName)
	
	common.SysLog(fmt.Sprintf("渠道「%s」（#%d）模型「%s」发生错误，准备禁用该模型，原因：%s", channel.Name, channel.Id, modelName, common.LocalLogPreview(reason)))

	// 禁用能力表中的模型
	err := model.UpdateAbilityTestResultAndStatus(
		channel.Id,
		modelName,
		model.AbilityTestStatusUnavailable,
		0, // 响应时间
		reason,
		"", // 响应内容
		common.ChannelStatusAutoDisabled,
	)
	if err != nil {
		common.SysError(fmt.Sprintf("禁用模型「%s」失败: %s", modelName, err.Error()))
		return
	}

	// 如果该渠道的所有模型都不可用，则自动禁用整个渠道
	allUnavailable, _ := model.AutoDisableChannelIfAllModelsUnavailable(channel.Id, "all models unavailable")
	if allUnavailable {
		DisableChannel(*types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, "", channel.GetAutoBan()), "all models unavailable")
	}
}
