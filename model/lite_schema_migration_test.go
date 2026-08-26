package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/55gY/new-api-lite/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type legacyLiteUser struct {
	Id              int            `gorm:"primaryKey"`
	Username        string         `gorm:"unique;index"`
	Password        string         `gorm:"not null"`
	DisplayName     string         `gorm:"index"`
	Role            int            `gorm:"default:1"`
	Status          int            `gorm:"default:1"`
	Email           string         `gorm:"index"`
	AccessToken     *string        `gorm:"column:access_token;uniqueIndex"`
	RequestCount    int            `gorm:"default:0"`
	Group           string         `gorm:"default:'default'"`
	AffCode         string         `gorm:"column:aff_code;uniqueIndex"`
	AffCount        int            `gorm:"column:aff_count;default:0"`
	InviterId       int            `gorm:"column:inviter_id;index"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
	Setting         string         `gorm:"column:setting"`
	Remark          string
	CreatedAt       int64 `gorm:"column:created_at"`
	LastLoginAt     int64 `gorm:"column:last_login_at"`
	Quota           int
	UsedQuota       int
	AffQuota        int
	AffHistoryQuota int `gorm:"column:aff_history"`
}

func (legacyLiteUser) TableName() string { return "users" }

type legacyLiteToken struct {
	Id             int `gorm:"primaryKey"`
	Key            string
	RemainQuota    int
	UnlimitedQuota bool
	UsedQuota      int
}

func (legacyLiteToken) TableName() string { return "tokens" }

type legacyLiteLog struct {
	Id    int `gorm:"primaryKey"`
	Quota int
}

func (legacyLiteLog) TableName() string { return "logs" }

type legacyLiteQuotaData struct {
	Id        int `gorm:"primaryKey"`
	TokenUsed int
	Count     int
	Quota     int
}

func (legacyLiteQuotaData) TableName() string { return "quota_data" }

type legacyLiteChannel struct {
	Id                 int `gorm:"primaryKey"`
	Key                string
	UsedQuota          int64
	Balance            float64
	BalanceUpdatedTime int64
}

func (legacyLiteChannel) TableName() string { return "channels" }

type legacyLiteOption struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

func (legacyLiteOption) TableName() string { return "options" }

func TestMigrateLegacyLiteSchemaPreservesTokensAndCreatesBackup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&legacyLiteUser{},
		&legacyLiteToken{},
		&legacyLiteLog{},
		&legacyLiteQuotaData{},
		&legacyLiteChannel{},
		&legacyLiteOption{},
	))
	require.NoError(t, db.Create(&legacyLiteUser{Id: 1, Username: "legacy-user", Password: "password", Quota: 100, UsedQuota: 99, AffQuota: 7, AffHistoryQuota: 8}).Error)
	require.NoError(t, db.Create(&legacyLiteToken{Id: 1, Key: "preserve-token", RemainQuota: 10, UnlimitedQuota: true, UsedQuota: 5}).Error)
	require.NoError(t, db.Create(&legacyLiteLog{Id: 1, Quota: 42}).Error)
	require.NoError(t, db.Create(&legacyLiteQuotaData{Id: 1, TokenUsed: 321, Count: 2, Quota: 42}).Error)
	require.NoError(t, db.Create(&legacyLiteChannel{Id: 1, Key: "legacy-channel-key", UsedQuota: 654, Balance: 12.34, BalanceUpdatedTime: 123456}).Error)
	for _, key := range legacyLiteOptionKeys {
		require.NoError(t, db.Create(&legacyLiteOption{Key: key, Value: "retired"}).Error)
	}
	for _, prefix := range legacyLiteOptionPrefixes {
		require.NoError(t, db.Create(&legacyLiteOption{Key: prefix + "retired", Value: "retired"}).Error)
	}
	require.NoError(t, db.Create(&legacyLiteOption{Key: "SystemName", Value: "New API Lite"}).Error)

	require.NoError(t, migrateLegacyLiteSchemaFor(db, dbPath))

	backups, err := filepath.Glob(dbPath + ".pre-lite-schema-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backupInfo, err := os.Stat(backups[0])
	require.NoError(t, err)
	require.NotZero(t, backupInfo.Size())

	for table, columns := range legacyLiteColumns {
		for _, column := range columns {
			require.Falsef(t, db.Migrator().HasColumn(table, column), "%s.%s should be removed", table, column)
		}
	}
	require.False(t, db.Migrator().HasTable("quota_data"))
	require.True(t, db.Migrator().HasTable("usage_data"))
	require.True(t, db.Migrator().HasColumn("channels", "used_tokens"))

	var token struct{ Key string }
	require.NoError(t, db.Table("tokens").Where("id = ?", 1).First(&token).Error)
	require.Equal(t, "preserve-token", token.Key)
	var usage struct {
		TokenUsed int
		Count     int
	}
	require.NoError(t, db.Table("usage_data").Where("id = ?", 1).First(&usage).Error)
	require.Equal(t, 321, usage.TokenUsed)
	require.Equal(t, 2, usage.Count)
	var usedTokens int64
	require.NoError(t, db.Table("channels").Select("used_tokens").Where("id = ?", 1).Scan(&usedTokens).Error)
	require.Equal(t, int64(654), usedTokens)
	var optionCount int64
	for _, key := range legacyLiteOptionKeys {
		optionCount = 0
		require.NoErrorf(t, db.Table("options").Where("`key` = ?", key).Count(&optionCount).Error, "count option %s", key)
		require.Zerof(t, optionCount, "legacy option %s should be removed", key)
	}
	for _, prefix := range legacyLiteOptionPrefixes {
		optionCount = 0
		require.NoErrorf(t, db.Table("options").Where("`key` LIKE ?", prefix+"%").Count(&optionCount).Error, "count options with prefix %s", prefix)
		require.Zerof(t, optionCount, "legacy option prefix %s should be removed", prefix)
	}
	require.NoError(t, db.Table("options").Where("`key` = ?", "SystemName").Count(&optionCount).Error)
	require.Equal(t, int64(1), optionCount)

	require.NoError(t, migrateLegacyLiteSchemaFor(db, dbPath))
	backups, err = filepath.Glob(dbPath + ".pre-lite-schema-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
}

func TestMigrateDBCompletesLegacyLiteSchemaMigration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy-startup.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&legacyLiteUser{},
		&legacyLiteToken{},
		&legacyLiteLog{},
		&legacyLiteQuotaData{},
		&legacyLiteChannel{},
		&legacyLiteOption{},
	))
	require.NoError(t, db.Create(&legacyLiteUser{Id: 1, Username: "legacy-user", Password: "password", Quota: 100, UsedQuota: 99, AffQuota: 7, AffHistoryQuota: 8}).Error)
	require.NoError(t, db.Create(&legacyLiteToken{Id: 1, Key: "startup-preserve-token", RemainQuota: 10, UnlimitedQuota: true, UsedQuota: 5}).Error)
	require.NoError(t, db.Create(&legacyLiteLog{Id: 1, Quota: 42}).Error)
	require.NoError(t, db.Create(&legacyLiteQuotaData{Id: 1, TokenUsed: 321, Count: 2, Quota: 42}).Error)
	require.NoError(t, db.Create(&legacyLiteChannel{Id: 1, Key: "legacy-channel-key", UsedQuota: 654, Balance: 12.34, BalanceUpdatedTime: 123456}).Error)
	for _, key := range legacyLiteOptionKeys {
		require.NoError(t, db.Create(&legacyLiteOption{Key: key, Value: "retired"}).Error)
	}
	require.NoError(t, db.Create(&legacyLiteOption{Key: "SystemName", Value: "New API Lite"}).Error)

	originalDB, originalLogDB, originalSQLitePath := DB, LOG_DB, common.SQLitePath
	DB, LOG_DB, common.SQLitePath = db, db, dbPath
	t.Cleanup(func() {
		DB, LOG_DB, common.SQLitePath = originalDB, originalLogDB, originalSQLitePath
		closeDB(db)
	})

	require.NoError(t, migrateDB())
	backups, err := filepath.Glob(dbPath + ".pre-lite-schema-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	for table, columns := range legacyLiteColumns {
		for _, column := range columns {
			require.Falsef(t, db.Migrator().HasColumn(table, column), "%s.%s should be removed after full migration", table, column)
		}
	}
	require.True(t, db.Migrator().HasTable(&UsageData{}))
	require.True(t, db.Migrator().HasColumn(&Channel{}, "used_tokens"))
	require.True(t, db.Migrator().HasColumn(&User{}, "request_count"))
	require.True(t, db.Migrator().HasColumn(&Token{}, "model_limits"))
	var key string
	require.NoError(t, db.Table("tokens").Select("key").Where("id = ?", 1).Scan(&key).Error)
	require.Equal(t, "startup-preserve-token", key)
	var usedTokens int64
	require.NoError(t, db.Table("channels").Select("used_tokens").Where("id = ?", 1).Scan(&usedTokens).Error)
	require.Equal(t, int64(654), usedTokens)
	var tokenUsed, count int
	require.NoError(t, db.Table("usage_data").Select("token_used", "count").Where("id = ?", 1).Row().Scan(&tokenUsed, &count))
	require.Equal(t, 321, tokenUsed)
	require.Equal(t, 2, count)
	var optionCount int64
	for _, key := range legacyLiteOptionKeys {
		require.NoError(t, db.Table("options").Where("`key` = ?", key).Count(&optionCount).Error)
		require.Zero(t, optionCount)
	}
	require.NoError(t, db.Table("options").Where("`key` = ?", "SystemName").Count(&optionCount).Error)
	require.Equal(t, int64(1), optionCount)
}

func TestMigrateLegacyLiteSchemaSkipsBackupForCurrentSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "current.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyLiteOption{}))
	require.NoError(t, db.Create(&legacyLiteOption{Key: "SystemName", Value: "New API Lite"}).Error)

	require.NoError(t, migrateLegacyLiteSchemaFor(db, dbPath))
	backups, err := filepath.Glob(dbPath + ".pre-lite-schema-*.bak")
	require.NoError(t, err)
	require.Empty(t, backups)
}
