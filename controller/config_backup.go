package controller

import (
	"net/http"
	"time"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/model"
	"github.com/gin-gonic/gin"
)

const configBackupMaxPayloadBytes = 16 << 20

type ConfigBackupExportRequest struct {
	Categories       []string `json:"categories"`
	IncludeSensitive bool     `json:"include_sensitive"`
}

type ConfigBackupRestoreRequest struct {
	Backup           model.ConfigurationBackup `json:"backup"`
	Categories       []string                  `json:"categories"`
	ConfirmSensitive bool                      `json:"confirm_sensitive"`
	ConfirmChannels  bool                      `json:"confirm_channels"`
}

func GetConfigBackupCategories(c *gin.Context) {
	categories, err := model.ConfigBackupCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    categories,
	})
}

func ExportConfigBackup(c *gin.Context) {
	var request ConfigBackupExportRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的备份请求"})
		return
	}
	if configBackupContainsSensitiveCategory(request.Categories) && !request.IncludeSensitive {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "导出敏感凭据或渠道配置前必须确认包含敏感信息",
		})
		return
	}
	backup, err := model.BuildConfigurationBackup(request.Categories)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	filename := "new-api-lite-config-backup-" + time.Now().UTC().Format("20060102T150405Z") + ".json"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, backup)
}

func RestoreConfigBackup(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, configBackupMaxPayloadBytes)
	var request ConfigBackupRestoreRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效或过大的配置备份文件"})
		return
	}
	if configBackupContainsSensitiveCategory(request.Categories) && !request.ConfirmSensitive {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "还原敏感凭据前必须确认覆盖当前敏感配置",
		})
		return
	}
	if configBackupContainsCategory(request.Categories, model.ConfigBackupCategoryChannels) && !request.ConfirmChannels {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "还原渠道配置会替换全部渠道，必须确认后才能继续",
		})
		return
	}
	result, err := model.RestoreConfigurationBackup(request.Backup, request.Categories)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置还原成功",
		"data":    result,
	})
}

func configBackupContainsSensitiveCategory(categories []string) bool {
	return configBackupContainsCategory(categories, model.ConfigBackupCategoryCredentials) ||
		configBackupContainsCategory(categories, model.ConfigBackupCategoryChannels)
}

func configBackupContainsCategory(categories []string, target string) bool {
	for _, category := range categories {
		if category == target {
			return true
		}
	}
	return false
}
