package model

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/logger"
)

const AutoModelName = "auto"

// autoModelMapping 返回不落库的自动映射视图。
// 现有 model_mapping 的方向是“实际模型 -> 请求模型”；自动候选只作为
// 运行时视图加入，不覆盖管理员已有的手动映射，也不把 auto 作为实际模型。
func autoModelMapping(channel *Channel) map[string]string {
	mapping := parseModelMapping(channel.ModelMapping)
	if mapping == nil {
		mapping = make(map[string]string)
	}
	for _, modelName := range channel.GetModels() {
		modelName = strings.TrimSpace(modelName)
		if !isConcreteChannelModel(channel, modelName) {
			continue
		}
		values := common.SplitModelMappingValues(mapping[modelName])
		if !common.StringsContains(values, AutoModelName) {
			values = append(values, AutoModelName)
		}
		mapping[modelName] = strings.Join(values, ",")
	}
	return mapping
}

// autoModelMappingJSON 返回供当前请求上下文使用的自动映射 JSON。
// 该 JSON 仅用于运行时，不会写入数据库，避免自动同步覆盖手动配置。
func GetRuntimeModelMappingJSON(channel *Channel) string {
	mapping := autoModelMapping(channel)
	if len(mapping) == 0 {
		return "{}"
	}
	data, err := common.Marshal(mapping)
	if err != nil {
		return channel.GetModelMapping()
	}
	return string(data)
}

// GetRuntimeSelectedModelMappingJSON 为本次已选的实际模型生成单候选运行时映射。
// 这样 auto 在同一渠道拥有多个实际模型时不会因反向映射键覆盖而随机落到错误模型。
func GetRuntimeSelectedModelMappingJSON(channel *Channel, selectedModel string) string {
	mapping := parseModelMapping(channel.ModelMapping)
	if mapping == nil {
		mapping = make(map[string]string)
	}
	for actualModel, requestModels := range mapping {
		values := common.SplitModelMappingValues(requestModels)
		filtered := make([]string, 0, len(values))
		for _, value := range values {
			if value != AutoModelName {
				filtered = append(filtered, value)
			}
		}
		if len(filtered) == 0 {
			delete(mapping, actualModel)
		} else {
			mapping[actualModel] = strings.Join(filtered, ",")
		}
	}
	selectedModel = strings.TrimSpace(selectedModel)
	if isConcreteModelName(selectedModel) {
		mapping[selectedModel] = AutoModelName
	}
	data, err := common.Marshal(mapping)
	if err != nil {
		return channel.GetModelMapping()
	}
	return string(data)
}

// isConcreteModelName 判断候选是否为真实上游模型，防止 auto 或其他逻辑
// 映射名再次作为 auto 的最终目标。
func isConcreteModelName(modelName string) bool {
	modelName = strings.TrimSpace(modelName)
	return modelName != "" && !strings.EqualFold(modelName, AutoModelName)
}

// isConcreteChannelModel excludes a name that is only a request-side alias in
// model_mapping. A mapping key remains an actual model even if it also appears
// in another mapping value, which preserves explicit channel configuration.
func isConcreteChannelModel(channel *Channel, modelName string) bool {
	if channel == nil || !isConcreteModelName(modelName) {
		return false
	}
	mapping := parseModelMapping(channel.ModelMapping)
	if _, ok := mapping[modelName]; ok {
		return true
	}
	for _, requestModels := range mapping {
		for _, requestModel := range common.SplitModelMappingValues(requestModels) {
			if strings.EqualFold(strings.TrimSpace(requestModel), strings.TrimSpace(modelName)) {
				return false
			}
		}
	}
	return true
}

type autoCandidate struct {
	Model     string
	ChannelId int
	Priority  int64
	Weight    uint
}

// GetRandomAutoChannel selects a concrete channel model for an auto request.
// The current implementation deliberately reuses the existing priority/weight
// semantics; the candidate snapshot/cache layer can replace the query later
// without changing the service API.
func GetRandomAutoChannel(group string, retry int, excluded map[string]struct{}) (*Channel, string, error) {
	var candidates []autoCandidate
	group = strings.TrimSpace(group)
	if common.MemoryCacheEnabled {
		channelSyncLock.RLock()
		candidates = append(candidates, group2autoCandidates[group]...)
		channelSyncLock.RUnlock()
	} else {
		query := DB.Table("abilities").
			Select("abilities.model, abilities.channel_id, abilities.priority, abilities.weight").
			Joins("join channels on abilities.channel_id = channels.id").
			Where("abilities.enabled = ? AND abilities.status = ? AND abilities.test_status != ? AND channels.status = ?", true, common.ChannelStatusEnabled, AbilityTestStatusUnavailable, common.ChannelStatusEnabled)
		if group != "" {
			query = query.Where("abilities."+commonGroupCol+" = ?", group)
		}
		if err := query.Scan(&candidates).Error; err != nil {
			return nil, "", err
		}
	}
	filtered := make([]autoCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate.Model = strings.TrimSpace(candidate.Model)
		if !isConcreteModelName(candidate.Model) {
			continue
		}
		key := fmt.Sprintf("%d:%s", candidate.ChannelId, candidate.Model)
		if _, ok := excluded[key]; ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, candidate)
	}
	if len(filtered) == 0 {
		return nil, "", nil
	}
	recentHit := false
	if recent, ok := getRecentAutoSuccess(group); ok {
		recentCandidates := make([]autoCandidate, 0, 1)
		for _, candidate := range filtered {
			if candidate.ChannelId == recent.ChannelID && candidate.Model == recent.Model {
				recentCandidates = append(recentCandidates, candidate)
			}
		}
		if len(recentCandidates) > 0 {
			filtered = recentCandidates
			recentHit = true
		}
	}
	priorities := make([]int64, 0)
	prioritySet := make(map[int64]struct{})
	for _, candidate := range filtered {
		if _, ok := prioritySet[candidate.Priority]; !ok {
			prioritySet[candidate.Priority] = struct{}{}
			priorities = append(priorities, candidate.Priority)
		}
	}
	sort.Slice(priorities, func(i, j int) bool { return priorities[i] > priorities[j] })
	if retry < 0 || retry >= len(priorities) {
		return nil, "", nil
	}
	targetPriority := priorities[retry]
	selected := make([]autoCandidate, 0)
	var weightSum uint
	for _, candidate := range filtered {
		if candidate.Priority != targetPriority {
			continue
		}
		selected = append(selected, candidate)
		weightSum += candidate.Weight + 10
	}
	if len(selected) == 0 {
		return nil, "", nil
	}
	logger.LogDebug(context.Background(), "auto 路由选择: group=%s candidates=%d recent_hit=%t retry=%d", group, len(filtered), recentHit, retry)
	pick := common.GetRandomInt(int(weightSum))
	var chosen autoCandidate
	for _, candidate := range selected {
		pick -= int(candidate.Weight) + 10
		if pick <= 0 {
			chosen = candidate
			break
		}
	}
	if chosen.ChannelId == 0 {
		return nil, "", nil
	}
	var channel Channel
	if err := DB.First(&channel, "id = ?", chosen.ChannelId).Error; err != nil {
		return nil, "", err
	}
	logger.LogDebug(context.Background(), "auto 路由命中: channel_id=%d model=%s priority=%d", chosen.ChannelId, chosen.Model, chosen.Priority)
	return &channel, chosen.Model, nil
}
