package model

import (
	"testing"

	"github.com/55gY/new-api-lite/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openAutoMappingTestDB(t *testing.T) {
	t.Helper()
	common.UsingSQLite = true
	common.MemoryCacheEnabled = false
	initCol()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Ability{}, &Channel{}))
	DB = db
	LOG_DB = db
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestRuntimeModelMappingAddsAutoWithoutOverwritingManualMapping(t *testing.T) {
	channel := &Channel{
		Models:       "gpt-5.5,claude-4",
		ModelMapping: stringPtr(`{"gpt-5.5":"preferred", "claude-4":""}`),
	}

	mapping := autoModelMapping(channel)
	require.Equal(t, "preferred,auto", mapping["gpt-5.5"])
	require.Equal(t, "auto", mapping["claude-4"])
	require.NotContains(t, mapping, AutoModelName)
	require.Equal(t, `{"gpt-5.5":"preferred", "claude-4":""}`, channel.GetModelMapping())
}

func TestGetChannelAutoUsesConcreteEnabledAbility(t *testing.T) {
	openAutoMappingTestDB(t)
	priority := int64(10)
	channel := &Channel{
		Id:           1,
		Name:         "auto-channel",
		Key:          "test-key",
		Models:       "gpt-5.5",
		Group:        "default",
		Status:       common.ChannelStatusEnabled,
		Priority:     &priority,
		ModelMapping: stringPtr("{}"),
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "gpt-5.5",
		ChannelId: 1,
		Status:    common.ChannelStatusEnabled,
		Enabled:   true,
		Priority:  &priority,
	}).Error)

	selected, err := GetChannel("default", AutoModelName, 0)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, 1, selected.Id)
}

func stringPtr(value string) *string {
	return &value
}
