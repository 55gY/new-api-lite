package model

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func IsChannelEnabledForGroupModel(group string, modelName string, channelID int) bool {
	if modelName == "" || channelID <= 0 {
		return false
	}
	if !common.MemoryCacheEnabled {
		return isChannelEnabledForGroupModelDB(group, modelName, channelID)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	if group2model2channels == nil {
		return false
	}

	if isChannelIDInList(group2model2channels[group][modelName], channelID) {
		return true
	}
	if group2mappedModel2channels != nil && isChannelIDInList(group2mappedModel2channels[group][modelName], channelID) {
		return true
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	if normalized != "" && normalized != modelName {
		return isChannelIDInList(group2model2channels[group][normalized], channelID) ||
			(group2mappedModel2channels != nil && isChannelIDInList(group2mappedModel2channels[group][normalized], channelID))
	}
	return false
}

func IsChannelEnabledForAnyGroupModel(groups []string, modelName string, channelID int) bool {
	if len(groups) == 0 {
		return false
	}
	for _, g := range groups {
		if IsChannelEnabledForGroupModel(g, modelName, channelID) {
			return true
		}
	}
	return false
}

func isChannelEnabledForGroupModelDB(group string, modelName string, channelID int) bool {
	var count int64
	query := DB.Model(&Ability{}).
		Where("model = ? and channel_id = ? and enabled = ?", modelName, channelID, true)
	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	err := query.Count(&count).Error
	if err == nil && count > 0 {
		return true
	}
	if isChannelMappedForGroupModelDB(group, modelName, channelID) {
		return true
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	if normalized == "" || normalized == modelName {
		return false
	}
	count = 0
	query = DB.Model(&Ability{}).
		Where("model = ? and channel_id = ? and enabled = ?", normalized, channelID, true)
	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	err = query.Count(&count).Error
	return (err == nil && count > 0) || isChannelMappedForGroupModelDB(group, normalized, channelID)
}

func isChannelMappedForGroupModelDB(group string, requestModel string, channelID int) bool {
	type mappedAbility struct {
		Model        string  `gorm:"column:model"`
		ModelMapping *string `gorm:"column:model_mapping"`
	}
	var abilities []mappedAbility
	query := DB.Table("abilities").
		Select("abilities.model, channels.model_mapping").
		Joins("left join channels on abilities.channel_id = channels.id").
		Where("abilities.channel_id = ? and abilities.enabled = ?", channelID, true)
	if group != "" {
		query = query.Where("abilities."+commonGroupCol+" = ?", group)
	}
	if err := query.Scan(&abilities).Error; err != nil {
		return false
	}
	for _, ability := range abilities {
		actualModel := ability.Model
		if parseModelMapping(ability.ModelMapping)[actualModel] == requestModel {
			return true
		}
	}
	return false
}

func isChannelIDInList(list []int, channelID int) bool {
	for _, id := range list {
		if id == channelID {
			return true
		}
	}
	return false
}
