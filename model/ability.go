package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AbilityTestStatusUntested    = 0
	AbilityTestStatusAvailable   = 1
	AbilityTestStatusUnavailable = 2
)

const abilityTestTextMaxLength = 4096

type Ability struct {
	Group        string `json:"group" gorm:"type:varchar(64);primaryKey;autoIncrement:false"`
	Model        string `json:"model" gorm:"type:varchar(255);primaryKey;autoIncrement:false"`
	ChannelId    int    `json:"channel_id" gorm:"primaryKey;autoIncrement:false;index"`
	Status       int    `json:"status" gorm:"default:1"`
	Enabled      bool   `json:"enabled"`
	Priority     *int64 `json:"priority" gorm:"bigint;default:0;index"`
	Weight       uint   `json:"weight" gorm:"default:0;index"`
	TestStatus   int    `json:"test_status" gorm:"default:0"`
	TestTime     int64  `json:"test_time" gorm:"bigint;default:0"`
	ResponseTime int    `json:"response_time" gorm:"default:0"`
	TestError    string `json:"test_error" gorm:"type:text"`
	TestResponse string `json:"test_response" gorm:"type:text"`
}

func NormalizeAbilityStatus(status int, enabled bool) int {
	switch status {
	case common.ChannelStatusEnabled, common.ChannelStatusManuallyDisabled, common.ChannelStatusAutoDisabled:
		return status
	}
	if enabled {
		return common.ChannelStatusEnabled
	}
	return common.ChannelStatusManuallyDisabled
}

func AbilityEnabledForChannelStatus(abilityStatus int, channelStatus int) bool {
	return abilityStatus == common.ChannelStatusEnabled && channelStatus == common.ChannelStatusEnabled
}

type AbilityWithChannel struct {
	Ability
	ChannelType int `json:"channel_type"`
}

type EnabledModelChannel struct {
	Model        string  `json:"model"`
	ChannelId    int     `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	ChannelType  int     `json:"channel_type"`
	ModelMapping *string `json:"model_mapping"`
	Status       int     `json:"status"`
	Enabled      bool    `json:"enabled"`
	TestStatus   int     `json:"test_status"`
	TestTime     int64   `json:"test_time"`
	ResponseTime int     `json:"response_time"`
	TestError    string  `json:"test_error"`
	TestResponse string  `json:"test_response"`
}

func GetAllEnableAbilityWithChannels() ([]AbilityWithChannel, error) {
	var abilities []AbilityWithChannel
	err := DB.Table("abilities").
		Select("abilities.*, channels.type as channel_type").
		Joins("left join channels on abilities.channel_id = channels.id").
		Where("abilities.enabled = ?", true).
		Scan(&abilities).Error
	return abilities, err
}

func GetModelChannels() ([]EnabledModelChannel, error) {
	var channels []EnabledModelChannel
	err := DB.Table("abilities").
		Select("abilities.model, channels.id as channel_id, channels.name as channel_name, channels.type as channel_type, channels.model_mapping, abilities.status, abilities.enabled, abilities.test_status, abilities.test_time, abilities.response_time, abilities.test_error, abilities.test_response").
		Joins("left join channels on abilities.channel_id = channels.id").
		Where("channels.status != ?", common.ChannelStatusManuallyDisabled).
		Scan(&channels).Error
	return channels, err
}

func GetEnabledModelChannels() ([]EnabledModelChannel, error) {
	var channels []EnabledModelChannel
	err := DB.Table("abilities").
		Select("abilities.model, channels.id as channel_id, channels.name as channel_name, channels.type as channel_type, channels.model_mapping, abilities.status, abilities.enabled, abilities.test_status, abilities.test_time, abilities.response_time, abilities.test_error, abilities.test_response").
		Joins("left join channels on abilities.channel_id = channels.id").
		Where("abilities.enabled = ?", true).
		Scan(&channels).Error
	return channels, err
}

func truncateAbilityTestText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= abilityTestTextMaxLength {
		return text
	}
	runes := []rune(text)
	if len(runes) <= abilityTestTextMaxLength {
		return text
	}
	return string(runes[:abilityTestTextMaxLength])
}

func UpdateAbilityTestResult(channelId int, modelName string, status int, responseTime int, testError string, testResponse string) error {
	modelName = strings.TrimSpace(modelName)
	if channelId <= 0 || modelName == "" {
		return nil
	}
	updates := map[string]interface{}{
		"test_status":   status,
		"test_time":     time.Now().Unix(),
		"response_time": responseTime,
		"test_error":    truncateAbilityTestText(testError),
		"test_response": truncateAbilityTestText(testResponse),
	}
	return DB.Model(&Ability{}).Where("channel_id = ? AND model = ?", channelId, modelName).Updates(updates).Error
}

func UpdateAbilityTestResultAndStatus(channelId int, modelName string, testStatus int, responseTime int, testError string, testResponse string, abilityStatus int) error {
	modelName = strings.TrimSpace(modelName)
	if channelId <= 0 || modelName == "" {
		return nil
	}
	channelStatus := common.ChannelStatusEnabled
	var channel Channel
	if err := DB.Select("status").First(&channel, "id = ?", channelId).Error; err == nil {
		channelStatus = channel.Status
	}
	abilityStatus = NormalizeAbilityStatus(abilityStatus, abilityStatus == common.ChannelStatusEnabled)
	updates := map[string]interface{}{
		"status":        abilityStatus,
		"enabled":       AbilityEnabledForChannelStatus(abilityStatus, channelStatus),
		"test_status":   testStatus,
		"test_time":     time.Now().Unix(),
		"response_time": responseTime,
		"test_error":    truncateAbilityTestText(testError),
		"test_response": truncateAbilityTestText(testResponse),
	}
	return DB.Model(&Ability{}).Where("channel_id = ? AND model = ?", channelId, modelName).Updates(updates).Error
}

func IsChannelAllModelsUnavailable(channelId int) (bool, error) {
	if channelId <= 0 {
		return false, nil
	}
	abilities, err := GetChannelAbilities(channelId)
	if err != nil {
		return false, err
	}
	modelUnavailable := make(map[string]bool, len(abilities))
	for _, ability := range abilities {
		modelName := strings.TrimSpace(ability.Model)
		if modelName == "" {
			continue
		}
		unavailable := ability.Status != common.ChannelStatusEnabled || ability.TestStatus == AbilityTestStatusUnavailable
		if current, ok := modelUnavailable[modelName]; ok {
			modelUnavailable[modelName] = current && unavailable
			continue
		}
		modelUnavailable[modelName] = unavailable
	}
	if len(modelUnavailable) == 0 {
		return false, nil
	}
	for _, unavailable := range modelUnavailable {
		if !unavailable {
			return false, nil
		}
	}
	return true, nil
}

func IsChannelAnyModelAvailable(channelId int) (bool, error) {
	if channelId <= 0 {
		return false, nil
	}
	abilities, err := GetChannelAbilities(channelId)
	if err != nil {
		return false, err
	}
	for _, ability := range abilities {
		modelName := strings.TrimSpace(ability.Model)
		if modelName == "" {
			continue
		}
		if ability.Status == common.ChannelStatusEnabled && ability.TestStatus == AbilityTestStatusAvailable {
			return true, nil
		}
	}
	return false, nil
}

func AutoDisableChannelIfAllModelsUnavailable(channelId int, reason string) (bool, error) {
	if channelId <= 0 || !common.AutomaticDisableChannelEnabled {
		return false, nil
	}
	var channel Channel
	if err := DB.Select("status").First(&channel, "id = ?", channelId).Error; err != nil {
		return false, err
	}
	if channel.Status == common.ChannelStatusManuallyDisabled || channel.Status == common.ChannelStatusAutoDisabled {
		return false, nil
	}
	allUnavailable, err := IsChannelAllModelsUnavailable(channelId)
	if err != nil {
		return false, err
	}
	if !allUnavailable {
		return false, nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "All channel models are unavailable"
	}
	return UpdateChannelStatus(channelId, "", common.ChannelStatusAutoDisabled, reason), nil
}

func AutoEnableChannelIfAnyModelAvailable(channelId int, reason string) (bool, error) {
	if channelId <= 0 || !common.AutomaticDisableChannelEnabled {
		return false, nil
	}
	var channel Channel
	if err := DB.Select("status").First(&channel, "id = ?", channelId).Error; err != nil {
		return false, err
	}
	if channel.Status != common.ChannelStatusAutoDisabled {
		return false, nil
	}
	anyAvailable, err := IsChannelAnyModelAvailable(channelId)
	if err != nil {
		return false, err
	}
	if !anyAvailable {
		return false, nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "At least one channel model is available"
	}
	return UpdateChannelStatus(channelId, "", common.ChannelStatusEnabled, reason), nil
}

func GetChannelAbilities(channelId int) ([]Ability, error) {
	var abilities []Ability
	err := DB.Where("channel_id = ?", channelId).Order(commonGroupCol + " asc").Order("model asc").Find(&abilities).Error
	for i := range abilities {
		abilities[i].Status = NormalizeAbilityStatus(abilities[i].Status, abilities[i].Enabled)
	}
	return abilities, err
}

func GetChannelTestableAbilities(channelId int) ([]Ability, error) {
	abilities, err := GetChannelAbilities(channelId)
	if err != nil {
		return nil, err
	}
	testable := make([]Ability, 0, len(abilities))
	seen := make(map[string]struct{}, len(abilities))
	for _, ability := range abilities {
		if ability.Status == common.ChannelStatusManuallyDisabled {
			continue
		}
		modelName := strings.TrimSpace(ability.Model)
		if modelName == "" {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		testable = append(testable, ability)
	}
	return testable, nil
}

func UpdateChannelModelStatus(channelId int, modelName string, group string, status int) error {
	modelName = strings.TrimSpace(modelName)
	group = strings.TrimSpace(group)
	if channelId <= 0 || modelName == "" {
		return nil
	}
	status = NormalizeAbilityStatus(status, status == common.ChannelStatusEnabled)
	channelStatus := common.ChannelStatusEnabled
	var channel Channel
	if err := DB.Select("status").First(&channel, "id = ?", channelId).Error; err == nil {
		channelStatus = channel.Status
	}
	updates := map[string]interface{}{
		"status":  status,
		"enabled": AbilityEnabledForChannelStatus(status, channelStatus),
	}
	query := DB.Model(&Ability{}).Where("channel_id = ? AND model = ?", channelId, modelName)
	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	return query.Updates(updates).Error
}

func GetGroupEnabledModels(group string) []string {
	var models []string
	// Find distinct models
	DB.Table("abilities").Where(commonGroupCol+" = ? and enabled = ?", group, true).Distinct("model").Pluck("model", &models)
	return appendMappedRequestModels(models, group)
}

func GetEnabledModels() []string {
	var models []string
	// Find distinct models
	DB.Table("abilities").Where("enabled = ?", true).Distinct("model").Pluck("model", &models)
	return appendMappedRequestModels(models, "")
}

func GetAllEnableAbilities() []Ability {
	var abilities []Ability
	DB.Find(&abilities, "enabled = ?", true)
	return abilities
}

func getPriority(group string, model string, retry int) (int, error) {

	var priorities []int
	query := DB.Model(&Ability{}).
		Select("DISTINCT(priority)").
		Where("model = ? and enabled = ?", model, true).
		Order("priority DESC") // 按优先级降序排序
	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	err := query.Pluck("priority", &priorities).Error // Pluck用于将查询的结果直接扫描到一个切片中

	if err != nil {
		// 处理错误
		return 0, err
	}

	if len(priorities) == 0 {
		// 如果没有查询到优先级，则返回错误
		return 0, errors.New("数据库一致性被破坏")
	}

	// 确定要使用的优先级
	var priorityToUse int
	if retry >= len(priorities) {
		// 如果重试次数大于优先级数，则使用最小的优先级
		priorityToUse = priorities[len(priorities)-1]
	} else {
		priorityToUse = priorities[retry]
	}
	return priorityToUse, nil
}

func getChannelQuery(group string, model string, retry int) (*gorm.DB, error) {
	maxPrioritySubQuery := DB.Model(&Ability{}).Select("MAX(priority)").Where("model = ? and enabled = ?", model, true)
	if group != "" {
		maxPrioritySubQuery = maxPrioritySubQuery.Where(commonGroupCol+" = ?", group)
	}
	channelQuery := DB.Where("model = ? and enabled = ? and priority = (?)", model, true, maxPrioritySubQuery)
	if group != "" {
		channelQuery = channelQuery.Where(commonGroupCol+" = ?", group)
	}
	if retry != 0 {
		priority, err := getPriority(group, model, retry)
		if err != nil {
			return nil, err
		} else {
			channelQuery = DB.Where("model = ? and enabled = ? and priority = ?", model, true, priority)
			if group != "" {
				channelQuery = channelQuery.Where(commonGroupCol+" = ?", group)
			}
		}
	}

	return channelQuery, nil
}

func GetChannel(group string, model string, retry int) (*Channel, error) {
	directAbilities, err := getCandidateAbilities(group, model)
	if err != nil {
		return nil, err
	}
	directPriorityCount, err := countAbilityPriorities(directAbilities)
	if err != nil {
		return nil, err
	}
	if retry < directPriorityCount {
		return getChannelFromAbilities(directAbilities, retry)
	}

	mappedAbilities, err := getMappedCandidateAbilities(group, model)
	if err != nil {
		return nil, err
	}
	return getChannelFromAbilities(mappedAbilities, retry-directPriorityCount)
}

func parseModelMapping(modelMapping *string) map[string]string {
	if modelMapping == nil {
		return nil
	}
	rawMapping := strings.TrimSpace(*modelMapping)
	if rawMapping == "" || rawMapping == "{}" {
		return nil
	}
	parsed := make(map[string]string)
	if err := common.UnmarshalJsonStr(rawMapping, &parsed); err != nil {
		return nil
	}
	normalized := make(map[string]string, len(parsed))
	for actualModel, requestModel := range parsed {
		actualModel = strings.TrimSpace(actualModel)
		requestModel = strings.TrimSpace(requestModel)
		if actualModel == "" || requestModel == "" {
			continue
		}
		normalized[actualModel] = requestModel
	}
	return normalized
}

func appendMappedRequestModels(models []string, group string) []string {
	modelSet := make(map[string]struct{}, len(models))
	for _, modelName := range models {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		modelSet[modelName] = struct{}{}
	}

	type abilityMapping struct {
		Model        string  `gorm:"column:model"`
		ModelMapping *string `gorm:"column:model_mapping"`
	}
	var mappings []abilityMapping
	query := DB.Table("abilities").
		Select("abilities.model, channels.model_mapping").
		Joins("left join channels on abilities.channel_id = channels.id").
		Where("abilities.enabled = ?", true)
	if group != "" {
		query = query.Where("abilities."+commonGroupCol+" = ?", group)
	}
	if err := query.Scan(&mappings).Error; err != nil {
		return models
	}
	for _, item := range mappings {
		actualModel := strings.TrimSpace(item.Model)
		if actualModel == "" {
			continue
		}
		mapping := parseModelMapping(item.ModelMapping)
		for _, requestModel := range common.SplitModelMappingValues(mapping[actualModel]) {
			modelSet[requestModel] = struct{}{}
		}
	}

	merged := make([]string, 0, len(modelSet))
	for modelName := range modelSet {
		merged = append(merged, modelName)
	}
	sort.Strings(merged)
	return merged
}

func getCandidateAbilities(group string, model string) ([]Ability, error) {
	var abilities []Ability
	query := DB.Model(&Ability{}).Where("model = ? and enabled = ?", model, true).Order("weight DESC")
	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	return abilities, query.Find(&abilities).Error
}

func getMappedCandidateAbilities(group string, requestModel string) ([]Ability, error) {
	type mappedAbility struct {
		Ability
		ModelMapping *string `gorm:"column:model_mapping"`
	}
	var mappedAbilities []mappedAbility
	query := DB.Table("abilities").
		Select("abilities.*, channels.model_mapping").
		Joins("left join channels on abilities.channel_id = channels.id").
		Where("abilities.enabled = ?", true).
		Order("abilities.weight DESC")
	if group != "" {
		query = query.Where("abilities."+commonGroupCol+" = ?", group)
	}
	if err := query.Scan(&mappedAbilities).Error; err != nil {
		return nil, err
	}

	abilities := make([]Ability, 0, len(mappedAbilities))
	for _, item := range mappedAbilities {
		actualModel := strings.TrimSpace(item.Model)
		mapping := parseModelMapping(item.ModelMapping)
		if !common.StringsContains(common.SplitModelMappingValues(mapping[actualModel]), requestModel) {
			continue
		}
		abilities = append(abilities, item.Ability)
	}
	return abilities, nil
}

func abilityPriority(ability Ability) int {
	if ability.Priority == nil {
		return 0
	}
	return int(*ability.Priority)
}

func countAbilityPriorities(abilities []Ability) (int, error) {
	priorities, err := getSortedAbilityPriorities(abilities)
	if err != nil {
		return 0, err
	}
	return len(priorities), nil
}

func getSortedAbilityPriorities(abilities []Ability) ([]int, error) {
	uniquePriorities := make(map[int]struct{})
	for _, ability := range abilities {
		uniquePriorities[abilityPriority(ability)] = struct{}{}
	}
	priorities := make([]int, 0, len(uniquePriorities))
	for priority := range uniquePriorities {
		priorities = append(priorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(priorities)))
	return priorities, nil
}

func getChannelFromAbilities(abilities []Ability, retry int) (*Channel, error) {
	priorities, err := getSortedAbilityPriorities(abilities)
	if err != nil {
		return nil, err
	}
	if retry >= len(priorities) {
		return nil, nil
	}
	targetPriority := priorities[retry]
	targetAbilities := make([]Ability, 0, len(abilities))
	weightSum := uint(0)
	for _, ability := range abilities {
		if abilityPriority(ability) != targetPriority {
			continue
		}
		targetAbilities = append(targetAbilities, ability)
		weightSum += ability.Weight + 10
	}
	if len(targetAbilities) == 0 {
		return nil, nil
	}
	weight := common.GetRandomInt(int(weightSum))
	channelId := 0
	for _, ability := range targetAbilities {
		weight -= int(ability.Weight) + 10
		if weight <= 0 {
			channelId = ability.ChannelId
			break
		}
	}
	if channelId == 0 {
		return nil, errors.New("channel not found")
	}
	channel := Channel{}
	err = DB.First(&channel, "id = ?", channelId).Error
	return &channel, err
}

func (channel *Channel) AddAbilities(tx *gorm.DB) error {
	models_ := strings.Split(channel.Models, ",")
	groups_ := strings.Split(channel.Group, ",")
	abilitySet := make(map[string]struct{})
	abilities := make([]Ability, 0, len(models_))
	for _, model := range models_ {
		for _, group := range groups_ {
			key := group + "|" + model
			if _, exists := abilitySet[key]; exists {
				continue
			}
			abilitySet[key] = struct{}{}
			ability := Ability{
				Group:     group,
				Model:     model,
				ChannelId: channel.Id,
				Status:    common.ChannelStatusEnabled,
				Enabled:   channel.Status == common.ChannelStatusEnabled,
				Priority:  channel.Priority,
				Weight:    uint(channel.GetWeight()),
			}
			abilities = append(abilities, ability)
		}
	}
	if len(abilities) == 0 {
		return nil
	}
	// choose DB or provided tx
	useDB := DB
	if tx != nil {
		useDB = tx
	}
	for _, chunk := range lo.Chunk(abilities, 50) {
		err := useDB.Clauses(clause.OnConflict{DoNothing: true}).Create(&chunk).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) DeleteAbilities() error {
	return DB.Where("channel_id = ?", channel.Id).Delete(&Ability{}).Error
}

// UpdateAbilities updates abilities of this channel.
// Make sure the channel is completed before calling this function.
func (channel *Channel) UpdateAbilities(tx *gorm.DB) error {
	isNewTx := false
	// 如果没有传入事务，创建新的事务
	if tx == nil {
		tx = DB.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		isNewTx = true
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
	}

	var existingAbilities []Ability
	err := tx.Where("channel_id = ?", channel.Id).Find(&existingAbilities).Error
	if err != nil {
		if isNewTx {
			tx.Rollback()
		}
		return err
	}
	existingByKey := make(map[string]Ability, len(existingAbilities))
	for _, ability := range existingAbilities {
		existingByKey[ability.Group+"|"+ability.Model] = ability
	}

	models_ := strings.Split(channel.Models, ",")
	groups_ := strings.Split(channel.Group, ",")
	abilitySet := make(map[string]struct{})
	abilities := make([]Ability, 0, len(models_))
	for _, model := range models_ {
		for _, group := range groups_ {
			key := group + "|" + model
			if _, exists := abilitySet[key]; exists {
				continue
			}
			abilitySet[key] = struct{}{}
			abilityStatus := common.ChannelStatusEnabled
			ability := Ability{
				Group:     group,
				Model:     model,
				ChannelId: channel.Id,
				Status:    abilityStatus,
				Enabled:   AbilityEnabledForChannelStatus(abilityStatus, channel.Status),
				Priority:  channel.Priority,
				Weight:    uint(channel.GetWeight()),
			}
			if existing, ok := existingByKey[key]; ok {
				abilityStatus = NormalizeAbilityStatus(existing.Status, existing.Enabled)
				ability.Status = abilityStatus
				ability.Enabled = AbilityEnabledForChannelStatus(abilityStatus, channel.Status)
				ability.TestStatus = existing.TestStatus
				ability.TestTime = existing.TestTime
				ability.ResponseTime = existing.ResponseTime
				ability.TestError = existing.TestError
				ability.TestResponse = existing.TestResponse
			}
			abilities = append(abilities, ability)
		}
	}

	err = tx.Where("channel_id = ?", channel.Id).Delete(&Ability{}).Error
	if err != nil {
		if isNewTx {
			tx.Rollback()
		}
		return err
	}

	if len(abilities) > 0 {
		for _, chunk := range lo.Chunk(abilities, 50) {
			err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&chunk).Error
			if err != nil {
				if isNewTx {
					tx.Rollback()
				}
				return err
			}
		}
	}

	// 如果是新创建的事务，需要提交
	if isNewTx {
		return tx.Commit().Error
	}

	return nil
}

func UpdateAbilityStatus(channelId int, status bool) error {
	if !status {
		return DB.Model(&Ability{}).Where("channel_id = ?", channelId).Select("enabled").Update("enabled", false).Error
	}
	return DB.Model(&Ability{}).
		Where("channel_id = ? AND (status = ? OR status = ?)", channelId, common.ChannelStatusEnabled, 0).
		Select("enabled").Update("enabled", true).Error
}

var fixLock = sync.Mutex{}

func FixAbility() (int, int, error) {
	lock := fixLock.TryLock()
	if !lock {
		return 0, 0, errors.New("已经有一个修复任务在运行中，请稍后再试")
	}
	defer fixLock.Unlock()

	err := DB.Exec("DELETE FROM abilities").Error
	if err != nil {
		common.SysLog(fmt.Sprintf("Delete abilities failed: %s", err.Error()))
		return 0, 0, err
	}
	var channels []*Channel
	// Find all channels
	err = DB.Model(&Channel{}).Find(&channels).Error
	if err != nil {
		return 0, 0, err
	}
	if len(channels) == 0 {
		return 0, 0, nil
	}
	successCount := 0
	failCount := 0
	for _, chunk := range lo.Chunk(channels, 50) {
		ids := lo.Map(chunk, func(c *Channel, _ int) int { return c.Id })
		// Delete all abilities of this channel
		err = DB.Where("channel_id IN ?", ids).Delete(&Ability{}).Error
		if err != nil {
			common.SysLog(fmt.Sprintf("Delete abilities failed: %s", err.Error()))
			failCount += len(chunk)
			continue
		}
		// Then add new abilities
		for _, channel := range chunk {
			err = channel.AddAbilities(nil)
			if err != nil {
				common.SysLog(fmt.Sprintf("Add abilities for channel %d failed: %s", channel.Id, err.Error()))
				failCount++
			} else {
				successCount++
			}
		}
	}
	InitChannelCache()
	return successCount, failCount, nil
}
