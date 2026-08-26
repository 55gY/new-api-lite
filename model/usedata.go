package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/55gY/new-api-lite/common"
	"gorm.io/gorm"
)

// UsageData 按小时聚合的 tokens 使用统计。
type UsageData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	Count     int    `json:"count" gorm:"default:0"`
}

func UpdateUsageData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveUsageDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheUsageData = make(map[string]*UsageData)
var CacheUsageDataLock = sync.Mutex{}

func logUsageDataCache(userId int, username string, modelName string, createdAt int64, tokenUsed int) {
	key := fmt.Sprintf("%d-%s-%s-%d", userId, username, modelName, createdAt)
	usageData, ok := CacheUsageData[key]
	if ok {
		usageData.Count += 1
		usageData.TokenUsed += tokenUsed
	} else {
		usageData = &UsageData{
			UserID:    userId,
			Username:  username,
			ModelName: modelName,
			CreatedAt: createdAt,
			Count:     1,
			TokenUsed: tokenUsed,
		}
	}
	CacheUsageData[key] = usageData
}

func LogUsageData(userId int, username string, modelName string, createdAt int64, tokenUsed int) {
	// 只精确到小时
	createdAt = createdAt - (createdAt % 3600)

	CacheUsageDataLock.Lock()
	defer CacheUsageDataLock.Unlock()
	logUsageDataCache(userId, username, modelName, createdAt, tokenUsed)
}

func SaveUsageDataCache() {
	CacheUsageDataLock.Lock()
	defer CacheUsageDataLock.Unlock()
	size := len(CacheUsageData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, usageData := range CacheUsageData {
		usageDataDB := &UsageData{}
		DB.Table("usage_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
			usageData.UserID, usageData.Username, usageData.ModelName, usageData.CreatedAt).First(usageDataDB)
		if usageDataDB.Id > 0 {

			increaseUsageData(usageData.UserID, usageData.Username, usageData.ModelName, usageData.Count, usageData.CreatedAt, usageData.TokenUsed)
		} else {
			DB.Table("usage_data").Create(usageData)
		}
	}
	CacheUsageData = make(map[string]*UsageData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseUsageData(userId int, username string, modelName string, count int, createdAt int64, tokenUsed int) {
	err := DB.Table("usage_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
		userId, username, modelName, createdAt).Updates(map[string]interface{}{
		"count":      gorm.Expr("count + ?", count),
		"token_used": gorm.Expr("token_used + ?", tokenUsed),
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseUsageData error: %s", err))
	}
}

func GetUsageDataByUsername(username string, startTime int64, endTime int64) (usageData []*UsageData, err error) {
	var usageDatas []*UsageData
	// 从usage_data表中查询数据
	err = DB.Table("usage_data").Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).Find(&usageDatas).Error
	return usageDatas, err
}

func GetUsageDataByUserId(userId int, startTime int64, endTime int64) (usageData []*UsageData, err error) {
	var usageDatas []*UsageData
	// 从usage_data表中查询数据
	err = DB.Table("usage_data").Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).Find(&usageDatas).Error
	return usageDatas, err
}

func GetUsageDataGroupByUser(startTime int64, endTime int64) (usageData []*UsageData, err error) {
	var usageDatas []*UsageData
	err = DB.Table("usage_data").
		Select("username, created_at, sum(count) as count, sum(token_used) as token_used").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("username, created_at").
		Find(&usageDatas).Error
	return usageDatas, err
}

func GetAllUsageDates(startTime int64, endTime int64, username string) (usageData []*UsageData, err error) {
	if username != "" {
		return GetUsageDataByUsername(username, startTime, endTime)
	}
	var usageDatas []*UsageData
	// 从usage_data表中查询数据
	// only select model_name, sum(count) as count, model_name, created_at from usage_data group by model_name, created_at;
	//err = DB.Table("usage_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&usageDatas).Error
	err = DB.Table("usage_data").Select("model_name, sum(count) as count, sum(token_used) as token_used, created_at").Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&usageDatas).Error
	return usageDatas, err
}
