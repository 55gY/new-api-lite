package model

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/55gY/new-api-lite/common"
	"gorm.io/gorm"
)

var legacyLiteOptionKeys = []string{
	"TaskEnabled",
	"DisplayInCurrencyEnabled",
	"Price",
	"USDExchangeRate",
	"QuotaForNewUser",
	"QuotaForInviter",
	"QuotaForInvitee",
	"QuotaRemindThreshold",
	"PreConsumedQuota",
	"QuotaPerUnit",
	"ModelRatio",
	"ModelPrice",
	"CacheRatio",
	"CreateCacheRatio",
	"GroupRatio",
	"GroupGroupRatio",
	"CompletionRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
	"ExposeRatioEnabled",
	"general_setting.quota_display_type",
	"general_setting.custom_currency_symbol",
	"general_setting.custom_currency_exchange_rate",
}

var legacyLiteOptionPrefixes = []string{
	"billing_setting.",
	"tool_price_setting.",
}

func migrateLegacyLiteSchema() error {
	return migrateLegacyLiteSchemaFor(DB, common.SQLitePath)
}

func migrateLegacyLiteSchemaFor(db *gorm.DB, sqlitePath string) error {
	if !hasLegacyLiteSchema(db) {
		return nil
	}
	if _, err := backupSQLiteBeforeLiteSchemaMigration(db, sqlitePath); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable("quota_data") && !tx.Migrator().HasTable("usage_data") {
			if err := tx.Migrator().RenameTable("quota_data", "usage_data"); err != nil {
				return err
			}
		}
		if err := renameColumnIfPresent(tx, "channels", "used_quota", "used_tokens"); err != nil {
			return err
		}
		for table, columns := range legacyLiteColumns {
			for _, column := range columns {
				if err := dropColumnIfPresent(tx, table, column); err != nil {
					return err
				}
			}
		}
		if tx.Migrator().HasTable("options") {
			if err := deleteLegacyLiteOptions(tx); err != nil {
				return err
			}
		}
		return nil
	})
}

var legacyLiteColumns = map[string][]string{
	"users":      {"quota", "used_quota", "aff_quota", "aff_history"},
	"tokens":     {"remain_quota", "unlimited_quota", "used_quota"},
	"logs":       {"quota"},
	"quota_data": {"quota"},
	"usage_data": {"quota"},
	"channels":   {"balance", "balance_updated_time"},
}

func hasLegacyLiteSchema(db *gorm.DB) bool {
	if db.Migrator().HasTable("quota_data") {
		return true
	}
	if db.Migrator().HasTable("options") && hasLegacyLiteOptions(db) {
		return true
	}
	if db.Migrator().HasColumn("channels", "used_quota") {
		return true
	}
	for table, columns := range legacyLiteColumns {
		for _, column := range columns {
			if db.Migrator().HasColumn(table, column) {
				return true
			}
		}
	}
	return false
}

func hasLegacyLiteOptions(db *gorm.DB) bool {
	var count int64
	if db.Where("`key` IN ?", legacyLiteOptionKeys).Count(&count).Error == nil && count > 0 {
		return true
	}
	for _, prefix := range legacyLiteOptionPrefixes {
		count = 0
		if db.Where("`key` LIKE ?", prefix+"%").Count(&count).Error == nil && count > 0 {
			return true
		}
	}
	return false
}

func deleteLegacyLiteOptions(db *gorm.DB) error {
	if err := db.Where("`key` IN ?", legacyLiteOptionKeys).Delete(&Option{}).Error; err != nil {
		return err
	}
	for _, prefix := range legacyLiteOptionPrefixes {
		if err := db.Where("`key` LIKE ?", prefix+"%").Delete(&Option{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func renameColumnIfPresent(db *gorm.DB, table, oldName, newName string) error {
	if !isLegacyLiteIdentifier(table) || !isLegacyLiteIdentifier(oldName) || !isLegacyLiteIdentifier(newName) {
		return fmt.Errorf("invalid legacy Lite migration identifier")
	}
	if !db.Migrator().HasTable(table) || !db.Migrator().HasColumn(table, oldName) {
		return nil
	}
	if db.Migrator().HasColumn(table, newName) {
		return dropColumnIfPresent(db, table, oldName)
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE `%s` RENAME COLUMN `%s` TO `%s`", table, oldName, newName)).Error
}

func dropColumnIfPresent(db *gorm.DB, table, column string) error {
	if !isLegacyLiteIdentifier(table) || !isLegacyLiteIdentifier(column) {
		return fmt.Errorf("invalid legacy Lite migration identifier")
	}
	if !db.Migrator().HasTable(table) || !db.Migrator().HasColumn(table, column) {
		return nil
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", table, column)).Error
}

func isLegacyLiteIdentifier(identifier string) bool {
	for _, allowed := range []string{
		"users", "tokens", "logs", "quota_data", "usage_data", "channels",
		"quota", "used_quota", "aff_quota", "aff_history", "remain_quota", "unlimited_quota", "used_tokens",
		"balance", "balance_updated_time",
	} {
		if identifier == allowed {
			return true
		}
	}
	return false
}

func backupSQLiteBeforeLiteSchemaMigration(db *gorm.DB, sqlitePath string) (string, error) {
	if strings.HasPrefix(sqlitePath, "file:") || strings.HasPrefix(sqlitePath, ":memory:") {
		return "", nil
	}
	path := strings.SplitN(sqlitePath, "?", 2)[0]
	if path == "" || path == ":memory:" {
		return "", nil
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("stat SQLite database for legacy schema backup: %w", err)
	}
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		return "", fmt.Errorf("checkpoint SQLite database before backup: %w", err)
	}

	backupPath := fmt.Sprintf("%s.pre-lite-schema-%s.bak", path, time.Now().UTC().Format("20060102T150405Z"))
	source, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()
	destination, err := os.OpenFile(filepath.Clean(backupPath), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	defer destination.Close()
	if _, err = io.Copy(destination, source); err != nil {
		return "", err
	}
	if err = destination.Sync(); err != nil {
		return "", err
	}
	common.SysLog("created SQLite pre-lite-schema backup: " + backupPath)
	return backupPath, nil
}
