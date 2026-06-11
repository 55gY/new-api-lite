package model

import (
	"errors"
	"fmt"
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
	Enabled      bool   `json:"enabled"`
	Priority     *int64 `json:"priority" gorm:"bigint;default:0;index"`
	Weight       uint   `json:"weight" gorm:"default:0;index"`
	TestStatus   int    `json:"test_status" gorm:"default:0"`
	TestTime     int64  `json:"test_time" gorm:"bigint;default:0"`
	ResponseTime int    `json:"response_time" gorm:"default:0"`
	TestError    string `json:"test_error" gorm:"type:text"`
	TestResponse string `json:"test_response" gorm:"type:text"`
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

func GetEnabledModelChannels() ([]EnabledModelChannel, error) {
	var channels []EnabledModelChannel
	err := DB.Table("abilities").
		Select("abilities.model, channels.id as channel_id, channels.name as channel_name, channels.type as channel_type, channels.model_mapping, abilities.test_status, abilities.test_time, abilities.response_time, abilities.test_error, abilities.test_response").
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

func GetGroupEnabledModels(group string) []string {
	var models []string
	// Find distinct models
	DB.Table("abilities").Where(commonGroupCol+" = ? and enabled = ?", group, true).Distinct("model").Pluck("model", &models)
	return models
}

func GetEnabledModels() []string {
	var models []string
	// Find distinct models
	DB.Table("abilities").Where("enabled = ?", true).Distinct("model").Pluck("model", &models)
	return models
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
	var abilities []Ability

	var err error = nil
	channelQuery, err := getChannelQuery(group, model, retry)
	if err != nil {
		return nil, err
	}
	err = channelQuery.Order("weight DESC").Find(&abilities).Error
	if err != nil {
		return nil, err
	}
	channel := Channel{}
	if len(abilities) > 0 {
		// Randomly choose one
		weightSum := uint(0)
		for _, ability_ := range abilities {
			weightSum += ability_.Weight + 10
		}
		// Randomly choose one
		weight := common.GetRandomInt(int(weightSum))
		for _, ability_ := range abilities {
			weight -= int(ability_.Weight) + 10
			//log.Printf("weight: %d, ability weight: %d", weight, *ability_.Weight)
			if weight <= 0 {
				channel.Id = ability_.ChannelId
				break
			}
		}
	} else {
		return nil, nil
	}
	err = DB.First(&channel, "id = ?", channel.Id).Error
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

	// First delete all abilities of this channel
	err := tx.Where("channel_id = ?", channel.Id).Delete(&Ability{}).Error
	if err != nil {
		if isNewTx {
			tx.Rollback()
		}
		return err
	}

	// Then add new abilities
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
				Enabled:   channel.Status == common.ChannelStatusEnabled,
				Priority:  channel.Priority,
				Weight:    uint(channel.GetWeight()),
			}
			abilities = append(abilities, ability)
		}
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
	return DB.Model(&Ability{}).Where("channel_id = ?", channelId).Select("enabled").Update("enabled", status).Error
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
