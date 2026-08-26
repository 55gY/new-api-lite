package model

import (
	"path/filepath"
	"testing"

	"github.com/55gY/new-api-lite/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestConfigurationBackupFiltersCategoriesAndCredentials(t *testing.T) {
	withConfigBackupTestDatabase(t, func(db *gorm.DB) {
		setConfigBackupOptionMap(map[string]string{
			"SystemName":    "Lite Gateway",
			"SMTPToken":     "sensitive-smtp-token",
			"global.reason": "enabled",
		})
		require.NoError(t, db.Create(&Channel{Id: 11, Key: "upstream-secret", Name: "primary", Models: "gpt-test", Group: "default"}).Error)

		backup, err := BuildConfigurationBackup([]string{ConfigBackupCategorySystem})
		require.NoError(t, err)
		require.Equal(t, ConfigBackupFormat, backup.Format)
		require.Equal(t, ConfigBackupVersion, backup.Version)
		require.Equal(t, []string{ConfigBackupCategorySystem}, backup.Categories)
		require.Equal(t, "Lite Gateway", backup.Options["SystemName"])
		require.NotContains(t, backup.Options, "SMTPToken")
		require.NotContains(t, backup.Options, "global.reason")
		require.Nil(t, backup.Channels)

		backup, err = BuildConfigurationBackup([]string{ConfigBackupCategoryCredentials, ConfigBackupCategoryChannels})
		require.NoError(t, err)
		require.Equal(t, "sensitive-smtp-token", backup.Options["SMTPToken"])
		require.NotNil(t, backup.Channels)
		require.Len(t, backup.Channels.Channels, 1)
		require.Equal(t, "upstream-secret", backup.Channels.Channels[0].Key)
	})
}

func TestRestoreConfigurationBackupReplacesChannelsWithoutRuntimeStatistics(t *testing.T) {
	withConfigBackupTestDatabase(t, func(db *gorm.DB) {
		setConfigBackupOptionMap(map[string]string{"SystemName": "Backup Name"})
		backupChannel := Channel{
			Id:           11,
			Key:          "backup-channel-key",
			Name:         "backup-channel",
			Models:       "gpt-test",
			Group:        "default",
			UsedTokens:   987,
			TestTime:     123,
			ResponseTime: 456,
		}
		require.NoError(t, db.Create(&backupChannel).Error)
		require.NoError(t, db.Create(&Ability{
			Group: "default", Model: "gpt-test", ChannelId: 11, Enabled: true,
			Status: AbilityTestStatusUnavailable, TestStatus: AbilityTestStatusUnavailable,
			TestTime: 123, ResponseTime: 456, TestError: "transient test failure",
		}).Error)
		backup, err := BuildConfigurationBackup([]string{ConfigBackupCategorySystem, ConfigBackupCategoryChannels})
		require.NoError(t, err)
		require.Len(t, backup.Channels.Channels, 1)
		require.Len(t, backup.Channels.Abilities, 1)
		require.Zero(t, backup.Channels.Channels[0].Id-11)
		require.Equal(t, AbilityTestStatusUnavailable, backup.Channels.Abilities[0].Status)

		require.NoError(t, db.Create(&Channel{Id: 22, Key: "current-key", Name: "current", Models: "old-model", Group: "default", UsedTokens: 444}).Error)
		setConfigBackupOptionMap(map[string]string{"SystemName": "Current Name"})
		result, err := RestoreConfigurationBackup(*backup, []string{ConfigBackupCategorySystem, ConfigBackupCategoryChannels})
		require.NoError(t, err)
		require.Equal(t, 1, result.OptionsRestored)
		require.Equal(t, 1, result.ChannelsRestored)
		require.Equal(t, 1, result.AbilitiesRestored)

		var channels []Channel
		require.NoError(t, db.Order("id asc").Find(&channels).Error)
		require.Len(t, channels, 1)
		require.Equal(t, 11, channels[0].Id)
		require.Equal(t, "backup-channel-key", channels[0].Key)
		require.Zero(t, channels[0].UsedTokens)
		require.Zero(t, channels[0].TestTime)
		require.Zero(t, channels[0].ResponseTime)

		var abilities []Ability
		require.NoError(t, db.Find(&abilities).Error)
		require.Len(t, abilities, 1)
		require.Equal(t, AbilityTestStatusUnavailable, abilities[0].Status)
		require.Zero(t, abilities[0].TestStatus)
		require.Zero(t, abilities[0].TestTime)
		require.Zero(t, abilities[0].ResponseTime)
		require.Empty(t, abilities[0].TestError)

		var systemName string
		require.NoError(t, db.Model(&Option{}).Where("`key` = ?", "SystemName").Select("value").Scan(&systemName).Error)
		require.Equal(t, "Backup Name", systemName)
	})
}

func TestRestoreConfigurationBackupRestoresModelCatalog(t *testing.T) {
	withConfigBackupTestDatabase(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&Vendor{Id: 7, Name: "backup-vendor", Description: "backup vendor", Status: 1}).Error)
		require.NoError(t, db.Create(&Model{Id: 8, ModelName: "backup-model", VendorID: 7, Endpoints: "chat", Status: 1, SyncOfficial: 1}).Error)
		backup, err := BuildConfigurationBackup([]string{ConfigBackupCategoryCatalog})
		require.NoError(t, err)
		require.NotNil(t, backup.Catalog)
		require.Len(t, backup.Catalog.Vendors, 1)
		require.Len(t, backup.Catalog.Models, 1)
		require.Equal(t, "backup-vendor", backup.Catalog.Vendors[0].Name)
		require.Equal(t, "backup-model", backup.Catalog.Models[0].ModelName)

		require.NoError(t, db.Create(&Vendor{Id: 17, Name: "current-vendor", Status: 1}).Error)
		require.NoError(t, db.Create(&Model{Id: 18, ModelName: "current-model", VendorID: 17, Status: 1}).Error)
		result, err := RestoreConfigurationBackup(*backup, []string{ConfigBackupCategoryCatalog})
		require.NoError(t, err)
		require.Equal(t, 1, result.VendorsRestored)
		require.Equal(t, 1, result.ModelsRestored)

		var vendors []Vendor
		require.NoError(t, db.Order("id asc").Find(&vendors).Error)
		require.Len(t, vendors, 1)
		require.Equal(t, 7, vendors[0].Id)
		var models []Model
		require.NoError(t, db.Order("id asc").Find(&models).Error)
		require.Len(t, models, 1)
		require.Equal(t, 8, models[0].Id)
		require.Equal(t, 7, models[0].VendorID)
	})
}

func TestRestoreConfigurationBackupRestoresOnlySelectedCategories(t *testing.T) {
	withConfigBackupTestDatabase(t, func(db *gorm.DB) {
		setConfigBackupOptionMap(map[string]string{
			"SystemName":    "Current System",
			"global.reason": "Current Model",
		})
		backup := ConfigurationBackup{
			Format:     ConfigBackupFormat,
			Version:    ConfigBackupVersion,
			Categories: []string{ConfigBackupCategorySystem, ConfigBackupCategoryModel},
			Options: map[string]string{
				"SystemName":    "Restored System",
				"global.reason": "Restored Model",
			},
		}
		result, err := RestoreConfigurationBackup(backup, []string{ConfigBackupCategorySystem})
		require.NoError(t, err)
		require.Equal(t, 1, result.OptionsRestored)

		var restoredSystemName string
		require.NoError(t, db.Model(&Option{}).Where("`key` = ?", "SystemName").Select("value").Scan(&restoredSystemName).Error)
		require.Equal(t, "Restored System", restoredSystemName)
		var modelOptionCount int64
		require.NoError(t, db.Model(&Option{}).Where("`key` = ?", "global.reason").Count(&modelOptionCount).Error)
		require.Zero(t, modelOptionCount)
	})
}

func TestRestoreConfigurationBackupRejectsUnknownOption(t *testing.T) {
	withConfigBackupTestDatabase(t, func(_ *gorm.DB) {
		setConfigBackupOptionMap(map[string]string{"SystemName": "Gateway"})
		backup := ConfigurationBackup{
			Format:     ConfigBackupFormat,
			Version:    ConfigBackupVersion,
			Categories: []string{ConfigBackupCategorySystem},
			Options:    map[string]string{"UnknownOption": "value"},
		}
		_, err := RestoreConfigurationBackup(backup, []string{ConfigBackupCategorySystem})
		require.ErrorContains(t, err, "无效配置项")
	})
}

func withConfigBackupTestDatabase(t *testing.T, test func(db *gorm.DB)) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "config-backup.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}, &Channel{}, &Ability{}, &Vendor{}, &Model{}))
	initCol()

	originalDB, originalLogDB := DB, LOG_DB
	originalSystemName := common.SystemName
	originalOptionMap := currentOptionSnapshot()
	DB, LOG_DB = db, db
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
		common.SystemName = originalSystemName
		setConfigBackupOptionMap(originalOptionMap)
		_ = closeDB(db)
	})
	test(db)
}

func setConfigBackupOptionMap(values map[string]string) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	common.OptionMap = make(map[string]string, len(values))
	for key, value := range values {
		common.OptionMap[key] = value
	}
}
