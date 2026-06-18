package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/constant"
	"github.com/55gY/new-api-lite/logger"
	"github.com/55gY/new-api-lite/setting/ratio_setting"
)

var group2model2channels map[string]map[string][]int // enabled channel
var group2mappedModel2channels map[string]map[string][]int
var channelsIDM map[int]*Channel // all channels include disabled
var channelSyncLock sync.RWMutex

func InitChannelCache() {
	if !common.MemoryCacheEnabled {
		return
	}
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}
	var abilities []*Ability
	DB.Find(&abilities)
	groups := make(map[string]bool)
	for _, ability := range abilities {
		groups[ability.Group] = true
	}
	allModelChannelSeen := make(map[string]map[int]bool)
	allMappedModelChannelSeen := make(map[string]map[int]bool)
	newGroup2model2channels := make(map[string]map[string][]int)
	newGroup2mappedModel2channels := make(map[string]map[string][]int)
	newGroup2model2channels[""] = make(map[string][]int)
	newGroup2mappedModel2channels[""] = make(map[string][]int)
	for group := range groups {
		newGroup2model2channels[group] = make(map[string][]int)
		newGroup2mappedModel2channels[group] = make(map[string][]int)
	}
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue // skip disabled channels
		}
		groups := strings.Split(channel.Group, ",")
		channelModelSet := make(map[string]struct{})
		for _, model := range strings.Split(channel.Models, ",") {
			model = strings.TrimSpace(model)
			if model != "" {
				channelModelSet[model] = struct{}{}
			}
		}
		modelMapping := parseModelMapping(channel.ModelMapping)
		for _, group := range groups {
			group = strings.TrimSpace(group)
			for model := range channelModelSet {
				if _, ok := newGroup2model2channels[""][model]; !ok {
					newGroup2model2channels[""][model] = make([]int, 0)
				}
				if _, ok := allModelChannelSeen[model]; !ok {
					allModelChannelSeen[model] = make(map[int]bool)
				}
				if !allModelChannelSeen[model][channel.Id] {
					newGroup2model2channels[""][model] = append(newGroup2model2channels[""][model], channel.Id)
					allModelChannelSeen[model][channel.Id] = true
				}
				if _, ok := newGroup2model2channels[group][model]; !ok {
					newGroup2model2channels[group][model] = make([]int, 0)
				}
				newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], channel.Id)
			}
			for actualModel, requestModelsValue := range modelMapping {
				if _, ok := channelModelSet[actualModel]; !ok {
					continue
				}
				for _, requestModel := range common.SplitModelMappingValues(requestModelsValue) {
					if _, ok := newGroup2mappedModel2channels[""][requestModel]; !ok {
						newGroup2mappedModel2channels[""][requestModel] = make([]int, 0)
					}
					if _, ok := allMappedModelChannelSeen[requestModel]; !ok {
						allMappedModelChannelSeen[requestModel] = make(map[int]bool)
					}
					if !allMappedModelChannelSeen[requestModel][channel.Id] {
						newGroup2mappedModel2channels[""][requestModel] = append(newGroup2mappedModel2channels[""][requestModel], channel.Id)
						allMappedModelChannelSeen[requestModel][channel.Id] = true
					}
					if _, ok := newGroup2mappedModel2channels[group][requestModel]; !ok {
						newGroup2mappedModel2channels[group][requestModel] = make([]int, 0)
					}
					newGroup2mappedModel2channels[group][requestModel] = append(newGroup2mappedModel2channels[group][requestModel], channel.Id)
				}
			}
		}
	}

	// sort by priority
	for group, model2channels := range newGroup2model2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return newChannelId2channel[channels[i]].GetPriority() > newChannelId2channel[channels[j]].GetPriority()
			})
			newGroup2model2channels[group][model] = channels
		}
	}
	for group, model2channels := range newGroup2mappedModel2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return newChannelId2channel[channels[i]].GetPriority() > newChannelId2channel[channels[j]].GetPriority()
			})
			newGroup2mappedModel2channels[group][model] = channels
		}
	}

	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	group2mappedModel2channels = newGroup2mappedModel2channels
	//channelsIDM = newChannelId2channel
	for i, channel := range newChannelId2channel {
		if channel.ChannelInfo.IsMultiKey {
			channel.Keys = channel.GetKeys()
			if channel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
				if oldChannel, ok := channelsIDM[i]; ok {
					// 存在旧的渠道，如果是多key且轮询，保留轮询索引信息
					if oldChannel.ChannelInfo.IsMultiKey && oldChannel.ChannelInfo.MultiKeyMode == constant.MultiKeyModePolling {
						channel.ChannelInfo.MultiKeyPollingIndex = oldChannel.ChannelInfo.MultiKeyPollingIndex
					}
				}
			}
		}
	}
	channelsIDM = newChannelId2channel
	channelSyncLock.Unlock()
	common.SysLog("channels synced from database")
}

func SyncChannelCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func GetRandomSatisfiedChannel(group string, model string, retry int, isMappedPhase bool) (*Channel, error) {
	// if memory cache is disabled, get channel directly from database
	if !common.MemoryCacheEnabled {
		return GetChannel(group, model, retry)
	}

	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	if isMappedPhase {
		// 映射模型阶段：直接从 group2mappedModel2channels 获取，使用独立的 mappedRetry
		mappedChannels := getChannelsForModel(group2mappedModel2channels, group, model)
		mappedChannels = filterDisabledModels(mappedChannels, model)
		return getRandomChannelFromIDs(mappedChannels, retry, group, model)
	}

	// 实际模型阶段：从 group2model2channels 获取，使用 actualRetry
	directChannels := getChannelsForModel(group2model2channels, group, model)
	directChannels = filterDisabledModels(directChannels, model)
	directPriorityCount, err := countChannelPriorities(directChannels)
	if err != nil {
		return nil, err
	}
	if retry < directPriorityCount {
		return getRandomChannelFromIDs(directChannels, retry, group, model)
	}
	// 实际模型重试耗尽，返回 nil 触发切换到映射模型阶段
	return nil, nil
}

// filterDisabledModels 过滤掉已禁用或自动禁用的模型（基于 ability 表状态）
func filterDisabledModels(channelIDs []int, model string) []int {
	if len(channelIDs) == 0 {
		return channelIDs
	}

	filtered := make([]int, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		channel, ok := channelsIDM[channelID]
		if !ok {
			continue
		}
		// 渠道已禁用，跳过
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}
		// 检查 ability 表中该模型的状态
		ability := getAbilityStatus(channelID, model)
		if ability != nil && ability.TestStatus == AbilityTestStatusUnavailable {
			// 该模型已标记为不可用（自动禁用），跳过
			continue
		}
		filtered = append(filtered, channelID)
	}
	return filtered
}

// getAbilityStatus 获取指定渠道和模型的能力状态（从缓存或数据库）
func getAbilityStatus(channelID int, model string) *Ability {
	// 简化实现：直接查询数据库
	var ability Ability
	err := DB.Where("channel_id = ? AND model = ?", channelID, model).First(&ability).Error
	if err != nil {
		return nil
	}
	return &ability
}

func getChannelsForModel(groupModelChannels map[string]map[string][]int, group string, model string) []int {
	if groupModelChannels == nil || groupModelChannels[group] == nil {
		return nil
	}
	channels := groupModelChannels[group][model]
	if len(channels) != 0 {
		return channels
	}
	normalizedModel := ratio_setting.FormatMatchingModelName(model)
	if normalizedModel == "" || normalizedModel == model {
		return nil
	}
	return groupModelChannels[group][normalizedModel]
}

func countChannelPriorities(channels []int) (int, error) {
	uniquePriorities := make(map[int]bool)
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			uniquePriorities[int(channel.GetPriority())] = true
		} else {
			return 0, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}
	return len(uniquePriorities), nil
}

func getRandomChannelFromIDs(channels []int, retry int, group string, model string) (*Channel, error) {
	if len(channels) == 0 {
		return nil, nil
	}

	if len(channels) == 1 {
		if retry > 0 {
			return nil, nil
		}
		if channel, ok := channelsIDM[channels[0]]; ok {
			return channel, nil
		}
		return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channels[0])
	}

	uniquePriorities := make(map[int]bool)
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			uniquePriorities[int(channel.GetPriority())] = true
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}
	var sortedUniquePriorities []int
	for priority := range uniquePriorities {
		sortedUniquePriorities = append(sortedUniquePriorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedUniquePriorities)))

	if retry >= len(uniquePriorities) {
		return nil, nil
	}
	targetPriority := int64(sortedUniquePriorities[retry])

	// get the priority for the given retry number
	var sumWeight = 0
	var targetChannels []*Channel
	for _, channelId := range channels {
		if channel, ok := channelsIDM[channelId]; ok {
			if channel.GetPriority() == targetPriority {
				sumWeight += channel.GetWeight()
				targetChannels = append(targetChannels, channel)
			}
		} else {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
	}

	if len(targetChannels) == 0 {
		return nil, errors.New(fmt.Sprintf("no channel found, group: %s, model: %s, priority: %d", group, model, targetPriority))
	}

	// smoothing factor and adjustment
	smoothingFactor := 1
	smoothingAdjustment := 0

	if sumWeight == 0 {
		// when all channels have weight 0, set sumWeight to the number of channels and set smoothing adjustment to 100
		// each channel's effective weight = 100
		sumWeight = len(targetChannels) * 100
		smoothingAdjustment = 100
	} else if sumWeight/len(targetChannels) < 10 {
		// when the average weight is less than 10, set smoothing factor to 100
		smoothingFactor = 100
	}

	// Calculate the total weight of all channels up to endIdx
	totalWeight := sumWeight * smoothingFactor

	// Generate a random value in the range [0, totalWeight)
	randomWeight := rand.Intn(totalWeight)

	// Find a channel based on its weight
	for _, channel := range targetChannels {
		randomWeight -= channel.GetWeight()*smoothingFactor + smoothingAdjustment
		if randomWeight < 0 {
			return channel, nil
		}
	}
	// return null if no channel is not found
	return nil, errors.New("channel not found")
}

func CacheGetChannel(id int) (*Channel, error) {
	if !common.MemoryCacheEnabled {
		return GetChannelById(id, true)
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return c, nil
}

func CacheGetChannelInfo(id int) (*ChannelInfo, error) {
	if !common.MemoryCacheEnabled {
		channel, err := GetChannelById(id, true)
		if err != nil {
			return nil, err
		}
		return &channel.ChannelInfo, nil
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()

	c, ok := channelsIDM[id]
	if !ok {
		return nil, fmt.Errorf("渠道# %d，已不存在", id)
	}
	return &c.ChannelInfo, nil
}

func CacheUpdateChannelStatus(id int, status int) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel, ok := channelsIDM[id]; ok {
		channel.Status = status
	}
	if status != common.ChannelStatusEnabled {
		// delete the channel from group2model2channels
		for group, model2channels := range group2model2channels {
			for model, channels := range model2channels {
				for i, channelId := range channels {
					if channelId == id {
						// remove the channel from the slice
						group2model2channels[group][model] = append(channels[:i], channels[i+1:]...)
						break
					}
				}
			}
		}
		for group, model2channels := range group2mappedModel2channels {
			for model, channels := range model2channels {
				for i, channelId := range channels {
					if channelId == id {
						group2mappedModel2channels[group][model] = append(channels[:i], channels[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func CacheUpdateChannel(channel *Channel) {
	if !common.MemoryCacheEnabled {
		return
	}
	channelSyncLock.Lock()
	defer channelSyncLock.Unlock()
	if channel == nil {
		return
	}

	if channelsIDM == nil {
		channelsIDM = make(map[int]*Channel)
	}
	if oldChannel, ok := channelsIDM[channel.Id]; ok {
		logger.LogDebug(nil, "CacheUpdateChannel before: id=%d, name=%s, status=%d, polling_index=%d", channel.Id, channel.Name, channel.Status, oldChannel.ChannelInfo.MultiKeyPollingIndex)
	}
	channelsIDM[channel.Id] = channel
	logger.LogDebug(nil, "CacheUpdateChannel after: id=%d, name=%s, status=%d, polling_index=%d", channel.Id, channel.Name, channel.Status, channel.ChannelInfo.MultiKeyPollingIndex)
}
