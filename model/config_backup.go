package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/constant"
	"gorm.io/gorm"
)

const (
	ConfigBackupFormat  = "new-api-lite.config-backup"
	ConfigBackupVersion = 1

	ConfigBackupCategorySystem      = "system"
	ConfigBackupCategoryModel       = "model"
	ConfigBackupCategoryCatalog     = "catalog"
	ConfigBackupCategoryOperation   = "operation"
	ConfigBackupCategoryConsole     = "console"
	ConfigBackupCategoryPerformance = "performance"
	ConfigBackupCategorySecurity    = "security"
	ConfigBackupCategoryCredentials = "credentials"
	ConfigBackupCategoryChannels    = "channels"
)

var configBackupCategoryOrder = []string{
	ConfigBackupCategorySystem,
	ConfigBackupCategoryModel,
	ConfigBackupCategoryCatalog,
	ConfigBackupCategoryOperation,
	ConfigBackupCategoryConsole,
	ConfigBackupCategoryPerformance,
	ConfigBackupCategorySecurity,
	ConfigBackupCategoryCredentials,
	ConfigBackupCategoryChannels,
}

var configBackupCredentialKeys = map[string]struct{}{
	"SMTPServer":         {},
	"SMTPFrom":           {},
	"SMTPAccount":        {},
	"SMTPToken":          {},
	"TurnstileSiteKey":   {},
	"TurnstileSecretKey": {},
	"WorkerUrl":          {},
	"WorkerValidKey":     {},
}

type ConfigBackupCategoryInfo struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sensitive   bool   `json:"sensitive"`
	Destructive bool   `json:"destructive"`
	ItemCount   int    `json:"item_count"`
}

type ConfigurationBackup struct {
	Format     string                    `json:"format"`
	Version    int                       `json:"version"`
	CreatedAt  time.Time                 `json:"created_at"`
	Categories []string                  `json:"categories"`
	Options    map[string]string         `json:"options,omitempty"`
	Catalog    *ConfigurationCatalogData `json:"catalog,omitempty"`
	Channels   *ConfigurationChannelData `json:"channels,omitempty"`
}

type ConfigurationCatalogData struct {
	Vendors []ConfigurationVendor `json:"vendors"`
	Models  []ConfigurationModel  `json:"models"`
}

type ConfigurationVendor struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Status      int    `json:"status"`
}

type ConfigurationModel struct {
	Id           int    `json:"id"`
	ModelName    string `json:"model_name"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	Tags         string `json:"tags"`
	VendorID     int    `json:"vendor_id"`
	Endpoints    string `json:"endpoints"`
	Status       int    `json:"status"`
	SyncOfficial int    `json:"sync_official"`
	NameRule     int    `json:"name_rule"`
}

type ConfigurationChannelData struct {
	Channels  []ConfigurationChannel `json:"channels"`
	Abilities []ConfigurationAbility `json:"abilities"`
}

type ConfigurationChannel struct {
	Id                int                   `json:"id"`
	Type              int                   `json:"type"`
	Key               string                `json:"key"`
	TestModel         *string               `json:"test_model"`
	Status            int                   `json:"status"`
	Name              string                `json:"name"`
	Weight            *uint                 `json:"weight"`
	BaseURL           *string               `json:"base_url"`
	Other             string                `json:"other"`
	Models            string                `json:"models"`
	Group             string                `json:"group"`
	ModelMapping      *string               `json:"model_mapping"`
	StatusCodeMapping *string               `json:"status_code_mapping"`
	Priority          *int64                `json:"priority"`
	AutoBan           *int                  `json:"auto_ban"`
	DisableAutoTest   *int                  `json:"disable_auto_test"`
	OtherInfo         string                `json:"other_info"`
	Setting           *string               `json:"setting"`
	ParamOverride     *string               `json:"param_override"`
	HeaderOverride    *string               `json:"header_override"`
	Remark            *string               `json:"remark"`
	MultiKey          bool                  `json:"multi_key"`
	MultiKeyMode      constant.MultiKeyMode `json:"multi_key_mode"`
	OtherSettings     string                `json:"settings"`
}

type ConfigurationAbility struct {
	Group     string `json:"group"`
	Model     string `json:"model"`
	ChannelId int    `json:"channel_id"`
	Status    int    `json:"status"`
	Enabled   bool   `json:"enabled"`
	Priority  *int64 `json:"priority"`
	Weight    uint   `json:"weight"`
}

type ConfigurationRestoreResult struct {
	OptionsRestored   int `json:"options_restored"`
	VendorsRestored   int `json:"vendors_restored"`
	ModelsRestored    int `json:"models_restored"`
	ChannelsRestored  int `json:"channels_restored"`
	AbilitiesRestored int `json:"abilities_restored"`
}

func ConfigBackupCategories() ([]ConfigBackupCategoryInfo, error) {
	optionSnapshot := currentOptionSnapshot()
	counts := make(map[string]int, len(configBackupCategoryOrder))
	for key := range optionSnapshot {
		if category, ok := configBackupOptionCategory(key); ok {
			counts[category]++
		}
	}
	var channelCount, vendorCount, modelCount int64
	if err := DB.Model(&Channel{}).Count(&channelCount).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&Vendor{}).Count(&vendorCount).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&Model{}).Count(&modelCount).Error; err != nil {
		return nil, err
	}

	infos := make([]ConfigBackupCategoryInfo, 0, len(configBackupCategoryOrder))
	for _, category := range configBackupCategoryOrder {
		info := ConfigBackupCategoryInfo{Key: category, ItemCount: counts[category]}
		switch category {
		case ConfigBackupCategorySystem:
			info.Name, info.Description = "系统设置", "通用界面、注册、通知和运行设置"
		case ConfigBackupCategoryModel:
			info.Name, info.Description = "模型设置", "全局、Claude、Gemini 和 Qwen 模型配置"
		case ConfigBackupCategoryCatalog:
			info.Name, info.Description = "模型目录", "管理员维护的供应商与模型元数据"
			info.ItemCount = int(vendorCount + modelCount)
		case ConfigBackupCategoryOperation:
			info.Name, info.Description = "运营设置", "运营、监控、Token 上限和渠道亲和配置"
		case ConfigBackupCategoryConsole:
			info.Name, info.Description = "控制台内容", "公告、FAQ、导航和控制台展示配置"
		case ConfigBackupCategoryPerformance:
			info.Name, info.Description = "性能设置", "性能、指标和缓存策略配置"
		case ConfigBackupCategorySecurity:
			info.Name, info.Description = "安全设置", "SSRF、隐私协议和安全策略配置"
		case ConfigBackupCategoryCredentials:
			info.Name, info.Description, info.Sensitive = "敏感凭据", "SMTP、Turnstile、Worker 等密钥和连接配置", true
		case ConfigBackupCategoryChannels:
			info.Name, info.Description, info.Sensitive, info.Destructive = "渠道配置", "上游渠道、密钥、模型映射和路由配置；还原会替换全部渠道", true, true
			info.ItemCount = int(channelCount)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func BuildConfigurationBackup(categories []string) (*ConfigurationBackup, error) {
	selected, err := normalizeConfigBackupCategories(categories)
	if err != nil {
		return nil, err
	}
	selectedSet := categorySet(selected)
	backup := &ConfigurationBackup{
		Format:     ConfigBackupFormat,
		Version:    ConfigBackupVersion,
		CreatedAt:  time.Now().UTC(),
		Categories: selected,
		Options:    make(map[string]string),
	}

	for key, value := range currentOptionSnapshot() {
		category, ok := configBackupOptionCategory(key)
		if ok && selectedSet[category] {
			backup.Options[key] = value
		}
	}
	if len(backup.Options) == 0 {
		backup.Options = nil
	}
	if selectedSet[ConfigBackupCategoryCatalog] {
		catalog, err := exportConfigurationCatalog()
		if err != nil {
			return nil, err
		}
		backup.Catalog = catalog
	}
	if selectedSet[ConfigBackupCategoryChannels] {
		channels, err := exportConfigurationChannels()
		if err != nil {
			return nil, err
		}
		backup.Channels = channels
	}
	return backup, nil
}

func RestoreConfigurationBackup(backup ConfigurationBackup, categories []string) (*ConfigurationRestoreResult, error) {
	if backup.Format != ConfigBackupFormat {
		return nil, errors.New("不支持的配置备份格式")
	}
	if backup.Version != ConfigBackupVersion {
		return nil, fmt.Errorf("不支持的配置备份版本: %d", backup.Version)
	}
	selected, err := normalizeConfigBackupCategories(categories)
	if err != nil {
		return nil, err
	}
	selectedSet := categorySet(selected)
	backupSet := categorySet(backup.Categories)
	for _, category := range selected {
		if !backupSet[category] {
			return nil, fmt.Errorf("备份不包含所选类别: %s", category)
		}
	}

	optionValues, err := validateConfigurationBackupOptions(backup.Options, selectedSet)
	if err != nil {
		return nil, err
	}
	if selectedSet[ConfigBackupCategoryCatalog] {
		if backup.Catalog == nil {
			return nil, errors.New("备份不包含模型目录配置")
		}
		if err := validateConfigurationCatalog(*backup.Catalog); err != nil {
			return nil, err
		}
	}
	if selectedSet[ConfigBackupCategoryChannels] {
		if backup.Channels == nil {
			return nil, errors.New("备份不包含渠道配置")
		}
		if err := validateConfigurationChannels(*backup.Channels); err != nil {
			return nil, err
		}
	}

	result := &ConfigurationRestoreResult{OptionsRestored: len(optionValues)}
	err = DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range optionValues {
			option := Option{Key: key}
			if err := tx.FirstOrCreate(&option, Option{Key: key}).Error; err != nil {
				return err
			}
			option.Value = value
			if err := tx.Save(&option).Error; err != nil {
				return err
			}
		}
		if selectedSet[ConfigBackupCategoryCatalog] {
			if err := tx.Unscoped().Where("1 = 1").Delete(&Model{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("1 = 1").Delete(&Vendor{}).Error; err != nil {
				return err
			}
			for _, item := range backup.Catalog.Vendors {
				vendor := item.toVendor()
				if err := tx.Create(&vendor).Error; err != nil {
					return err
				}
			}
			for _, item := range backup.Catalog.Models {
				model := item.toModel()
				if err := tx.Create(&model).Error; err != nil {
					return err
				}
			}
			result.VendorsRestored = len(backup.Catalog.Vendors)
			result.ModelsRestored = len(backup.Catalog.Models)
		}
		if !selectedSet[ConfigBackupCategoryChannels] {
			return nil
		}
		if err := tx.Where("1 = 1").Delete(&Ability{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&Channel{}).Error; err != nil {
			return err
		}
		for _, item := range backup.Channels.Channels {
			channel := item.toChannel()
			if err := tx.Create(&channel).Error; err != nil {
				return err
			}
		}
		for _, item := range backup.Channels.Abilities {
			ability := item.toAbility()
			if err := tx.Create(&ability).Error; err != nil {
				return err
			}
		}
		result.ChannelsRestored = len(backup.Channels.Channels)
		result.AbilitiesRestored = len(backup.Channels.Abilities)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Rebuild effective runtime config and channel selection cache only after a
	// successful database commit. This keeps failed imports from mutating runtime state.
	InitOptionMap()
	if selectedSet[ConfigBackupCategoryChannels] {
		InitChannelCache()
	}
	return result, nil
}

func exportConfigurationCatalog() (*ConfigurationCatalogData, error) {
	var vendors []Vendor
	if err := DB.Order("id asc").Find(&vendors).Error; err != nil {
		return nil, err
	}
	var models []Model
	if err := DB.Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	data := &ConfigurationCatalogData{
		Vendors: make([]ConfigurationVendor, 0, len(vendors)),
		Models:  make([]ConfigurationModel, 0, len(models)),
	}
	for _, vendor := range vendors {
		data.Vendors = append(data.Vendors, ConfigurationVendor{
			Id:          vendor.Id,
			Name:        vendor.Name,
			Description: vendor.Description,
			Icon:        vendor.Icon,
			Status:      vendor.Status,
		})
	}
	for _, item := range models {
		data.Models = append(data.Models, ConfigurationModel{
			Id:           item.Id,
			ModelName:    item.ModelName,
			Description:  item.Description,
			Icon:         item.Icon,
			Tags:         item.Tags,
			VendorID:     item.VendorID,
			Endpoints:    item.Endpoints,
			Status:       item.Status,
			SyncOfficial: item.SyncOfficial,
			NameRule:     item.NameRule,
		})
	}
	return data, nil
}

func (item ConfigurationVendor) toVendor() Vendor {
	return Vendor{
		Id:          item.Id,
		Name:        item.Name,
		Description: item.Description,
		Icon:        item.Icon,
		Status:      item.Status,
	}
}

func (item ConfigurationModel) toModel() Model {
	return Model{
		Id:           item.Id,
		ModelName:    item.ModelName,
		Description:  item.Description,
		Icon:         item.Icon,
		Tags:         item.Tags,
		VendorID:     item.VendorID,
		Endpoints:    item.Endpoints,
		Status:       item.Status,
		SyncOfficial: item.SyncOfficial,
		NameRule:     item.NameRule,
	}
}

func validateConfigurationCatalog(data ConfigurationCatalogData) error {
	if len(data.Vendors) > 5000 || len(data.Models) > 50000 {
		return errors.New("模型目录配置数量超出限制")
	}
	vendorIDs := make(map[int]struct{}, len(data.Vendors))
	vendorNames := make(map[string]struct{}, len(data.Vendors))
	for _, vendor := range data.Vendors {
		name := strings.TrimSpace(vendor.Name)
		if vendor.Id <= 0 || name == "" {
			return errors.New("供应商配置包含无效的 ID 或名称")
		}
		if _, exists := vendorIDs[vendor.Id]; exists {
			return fmt.Errorf("供应商配置包含重复 ID: %d", vendor.Id)
		}
		if _, exists := vendorNames[name]; exists {
			return fmt.Errorf("供应商配置包含重复名称: %s", name)
		}
		vendorIDs[vendor.Id] = struct{}{}
		vendorNames[name] = struct{}{}
	}
	modelIDs := make(map[int]struct{}, len(data.Models))
	modelNames := make(map[string]struct{}, len(data.Models))
	for _, item := range data.Models {
		modelName := strings.TrimSpace(item.ModelName)
		if item.Id <= 0 || modelName == "" {
			return errors.New("模型配置包含无效的 ID 或名称")
		}
		if _, exists := modelIDs[item.Id]; exists {
			return fmt.Errorf("模型配置包含重复 ID: %d", item.Id)
		}
		if _, exists := modelNames[modelName]; exists {
			return fmt.Errorf("模型配置包含重复名称: %s", modelName)
		}
		if item.VendorID != 0 {
			if _, exists := vendorIDs[item.VendorID]; !exists {
				return fmt.Errorf("模型 %s 引用了不存在的供应商: %d", modelName, item.VendorID)
			}
		}
		if len(item.Endpoints) > 1<<20 {
			return fmt.Errorf("模型 %s 的配置值过大", modelName)
		}
		modelIDs[item.Id] = struct{}{}
		modelNames[modelName] = struct{}{}
	}
	return nil
}

func exportConfigurationChannels() (*ConfigurationChannelData, error) {
	var channels []Channel
	if err := DB.Order("id asc").Find(&channels).Error; err != nil {
		return nil, err
	}
	var abilities []Ability
	if err := DB.Order("channel_id asc").Order(commonGroupCol + " asc").Order("model asc").Find(&abilities).Error; err != nil {
		return nil, err
	}
	data := &ConfigurationChannelData{
		Channels:  make([]ConfigurationChannel, 0, len(channels)),
		Abilities: make([]ConfigurationAbility, 0, len(abilities)),
	}
	for _, channel := range channels {
		data.Channels = append(data.Channels, configurationChannelFrom(channel))
	}
	for _, ability := range abilities {
		data.Abilities = append(data.Abilities, ConfigurationAbility{
			Group:     ability.Group,
			Model:     ability.Model,
			ChannelId: ability.ChannelId,
			Status:    NormalizeAbilityStatus(ability.Status, ability.Enabled),
			Enabled:   ability.Enabled,
			Priority:  ability.Priority,
			Weight:    ability.Weight,
		})
	}
	return data, nil
}

func configurationChannelFrom(channel Channel) ConfigurationChannel {
	return ConfigurationChannel{
		Id:                channel.Id,
		Type:              channel.Type,
		Key:               channel.Key,
		TestModel:         channel.TestModel,
		Status:            channel.Status,
		Name:              channel.Name,
		Weight:            channel.Weight,
		BaseURL:           channel.BaseURL,
		Other:             channel.Other,
		Models:            channel.Models,
		Group:             channel.Group,
		ModelMapping:      channel.ModelMapping,
		StatusCodeMapping: channel.StatusCodeMapping,
		Priority:          channel.Priority,
		AutoBan:           channel.AutoBan,
		DisableAutoTest:   channel.DisableAutoTest,
		OtherInfo:         channel.OtherInfo,
		Setting:           channel.Setting,
		ParamOverride:     channel.ParamOverride,
		HeaderOverride:    channel.HeaderOverride,
		Remark:            channel.Remark,
		MultiKey:          channel.ChannelInfo.IsMultiKey,
		MultiKeyMode:      channel.ChannelInfo.MultiKeyMode,
		OtherSettings:     channel.OtherSettings,
	}
}

func (item ConfigurationChannel) toChannel() Channel {
	return Channel{
		Id:                item.Id,
		Type:              item.Type,
		Key:               item.Key,
		TestModel:         item.TestModel,
		Status:            item.Status,
		Name:              item.Name,
		Weight:            item.Weight,
		BaseURL:           item.BaseURL,
		Other:             item.Other,
		Models:            item.Models,
		Group:             item.Group,
		ModelMapping:      item.ModelMapping,
		StatusCodeMapping: item.StatusCodeMapping,
		Priority:          item.Priority,
		AutoBan:           item.AutoBan,
		DisableAutoTest:   item.DisableAutoTest,
		OtherInfo:         item.OtherInfo,
		Setting:           item.Setting,
		ParamOverride:     item.ParamOverride,
		HeaderOverride:    item.HeaderOverride,
		Remark:            item.Remark,
		ChannelInfo: ChannelInfo{
			IsMultiKey:   item.MultiKey,
			MultiKeyMode: item.MultiKeyMode,
		},
		OtherSettings: item.OtherSettings,
	}
}

func (item ConfigurationAbility) toAbility() Ability {
	status := NormalizeAbilityStatus(item.Status, item.Enabled)
	return Ability{
		Group:     item.Group,
		Model:     item.Model,
		ChannelId: item.ChannelId,
		Status:    status,
		Enabled:   item.Enabled,
		Priority:  item.Priority,
		Weight:    item.Weight,
	}
}

func validateConfigurationBackupOptions(options map[string]string, selected map[string]bool) (map[string]string, error) {
	if len(options) > 10000 {
		return nil, errors.New("备份中的配置项数量超出限制")
	}
	validKeys := currentOptionSnapshot()
	values := make(map[string]string, len(options))
	for key, value := range options {
		category, ok := configBackupOptionCategory(key)
		if !ok {
			return nil, fmt.Errorf("备份包含当前版本不支持的配置项: %s", key)
		}
		if _, exists := validKeys[key]; !exists {
			return nil, fmt.Errorf("备份包含无效配置项: %s", key)
		}
		if !selected[category] {
			// A complete backup can contain several categories while the operator
			// restores only one of them. Ignore valid but unselected categories.
			continue
		}
		if len(value) > 1<<20 {
			return nil, fmt.Errorf("配置项 %s 的值过大", key)
		}
		values[key] = value
	}
	return values, nil
}

func validateConfigurationChannels(data ConfigurationChannelData) error {
	if len(data.Channels) > 5000 || len(data.Abilities) > 100000 {
		return errors.New("渠道配置数量超出限制")
	}
	channelIDs := make(map[int]struct{}, len(data.Channels))
	for _, channel := range data.Channels {
		if channel.Id <= 0 || strings.TrimSpace(channel.Name) == "" {
			return errors.New("渠道配置包含无效的 ID 或名称")
		}
		if _, exists := channelIDs[channel.Id]; exists {
			return fmt.Errorf("渠道配置包含重复 ID: %d", channel.Id)
		}
		if len(channel.Key) > 1<<20 || len(channel.Models) > 1<<20 {
			return fmt.Errorf("渠道 %d 的配置值过大", channel.Id)
		}
		channelIDs[channel.Id] = struct{}{}
	}
	abilityKeys := make(map[string]struct{}, len(data.Abilities))
	for _, ability := range data.Abilities {
		if _, exists := channelIDs[ability.ChannelId]; !exists {
			return fmt.Errorf("能力配置引用了不存在的渠道: %d", ability.ChannelId)
		}
		if strings.TrimSpace(ability.Group) == "" || strings.TrimSpace(ability.Model) == "" {
			return errors.New("能力配置包含空分组或模型")
		}
		key := fmt.Sprintf("%d|%s|%s", ability.ChannelId, ability.Group, ability.Model)
		if _, exists := abilityKeys[key]; exists {
			return fmt.Errorf("能力配置包含重复项: %s", key)
		}
		abilityKeys[key] = struct{}{}
	}
	return nil
}

func currentOptionSnapshot() map[string]string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	result := make(map[string]string, len(common.OptionMap))
	for key, value := range common.OptionMap {
		result[key] = value
	}
	return result
}

func normalizeConfigBackupCategories(categories []string) ([]string, error) {
	if len(categories) == 0 {
		return nil, errors.New("至少选择一个配置类别")
	}
	allowed := categorySet(configBackupCategoryOrder)
	selected := categorySet(nil)
	for _, rawCategory := range categories {
		category := strings.TrimSpace(rawCategory)
		if !allowed[category] {
			return nil, fmt.Errorf("不支持的配置类别: %s", category)
		}
		selected[category] = true
	}
	result := make([]string, 0, len(selected))
	for _, category := range configBackupCategoryOrder {
		if selected[category] {
			result = append(result, category)
		}
	}
	return result, nil
}

func categorySet(categories []string) map[string]bool {
	set := make(map[string]bool, len(categories))
	for _, category := range categories {
		set[category] = true
	}
	return set
}

func configBackupOptionCategory(key string) (string, bool) {
	if _, ok := configBackupCredentialKeys[key]; ok {
		return ConfigBackupCategoryCredentials, true
	}
	switch {
	case strings.HasPrefix(key, "global."), strings.HasPrefix(key, "claude."), strings.HasPrefix(key, "gemini."), strings.HasPrefix(key, "qwen."):
		return ConfigBackupCategoryModel, true
	case strings.HasPrefix(key, "console_setting."):
		return ConfigBackupCategoryConsole, true
	case strings.HasPrefix(key, "performance_setting."), strings.HasPrefix(key, "perf_metrics_setting."), key == "StreamCacheQueueLength":
		return ConfigBackupCategoryPerformance, true
	case strings.HasPrefix(key, "fetch_setting."), strings.HasPrefix(key, "legal."):
		return ConfigBackupCategorySecurity, true
	case strings.HasPrefix(key, "general_setting."), strings.HasPrefix(key, "monitor_setting."), strings.HasPrefix(key, "token_setting."), strings.HasPrefix(key, "channel_affinity_setting."), strings.HasPrefix(key, "Automatic"), key == "ChannelDisableThreshold", key == "RetryTimes", key == "ModelRequestRateLimitCount", key == "ModelRequestRateLimitDurationMinutes", key == "ModelRequestRateLimitSuccessCount", key == "ModelRequestRateLimitGroup", key == "UserUsableGroups", key == "CheckSensitiveEnabled", key == "CheckSensitiveOnPromptEnabled", key == "StopOnSensitiveEnabled", key == "SensitiveWords":
		return ConfigBackupCategoryOperation, true
	default:
		return ConfigBackupCategorySystem, true
	}
}

func SortedConfigBackupCategoryKeys() []string {
	keys := append([]string(nil), configBackupCategoryOrder...)
	sort.Strings(keys)
	return keys
}
