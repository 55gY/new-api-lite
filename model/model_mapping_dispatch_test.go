package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openModelMappingDispatchTestDB(t *testing.T) {
	t.Helper()

	common.UsingSQLite = true
	common.MemoryCacheEnabled = false
	initCol()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
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

func insertMappingDispatchCandidate(t *testing.T, channelID int, models string, modelMapping string, abilityModel string, priority int64) {
	t.Helper()
	require.NoError(t, DB.Create(&Channel{
		Id:           channelID,
		Name:         fmt.Sprintf("channel-%d", channelID),
		Key:          fmt.Sprintf("key-%d", channelID),
		Models:       models,
		Group:        "default",
		ModelMapping: &modelMapping,
		Status:       common.ChannelStatusEnabled,
		Priority:     &priority,
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     abilityModel,
		ChannelId: channelID,
		Status:    common.ChannelStatusEnabled,
		Enabled:   true,
		Priority:  &priority,
	}).Error)
}

func TestGetChannelPrefersRealModelBeforeMappedModel(t *testing.T) {
	openModelMappingDispatchTestDB(t)

	emptyMapping := "{}"
	insertMappingDispatchCandidate(t, 1, "gpt-5.5", emptyMapping, "gpt-5.5", 1)
	modelMapping := `{"DeepSeek-V4":"gpt-5.5"}`
	insertMappingDispatchCandidate(t, 2, "DeepSeek-V4", modelMapping, "DeepSeek-V4", 100)

	channel, err := GetChannel("default", "gpt-5.5", 0)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 1, channel.Id)

	channel, err = GetChannel("default", "gpt-5.5", 1)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 2, channel.Id)
}

func TestGetChannelMappedFallbackUsesMappedChannelPriority(t *testing.T) {
	openModelMappingDispatchTestDB(t)

	modelMapping := `{"DeepSeek-V4":"gpt-5.5"}`
	insertMappingDispatchCandidate(t, 1, "DeepSeek-V4", modelMapping, "DeepSeek-V4", 10)
	insertMappingDispatchCandidate(t, 2, "DeepSeek-V4", modelMapping, "DeepSeek-V4", 20)

	channel, err := GetChannel("default", "gpt-5.5", 0)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 2, channel.Id)
}
