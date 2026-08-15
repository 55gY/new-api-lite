package model

import (
	"strings"
	"time"
)

const (
	ChannelCheckinTaskStatusDisabled = 0
	ChannelCheckinTaskStatusEnabled  = 1
)

const (
	ChannelCheckinRunStatusSuccess       = "success"
	ChannelCheckinRunStatusAlreadyDone   = "already_done"
	ChannelCheckinRunStatusManualAction  = "manual_action_required"
	ChannelCheckinRunStatusAuthFailed    = "authentication_failed"
	ChannelCheckinRunStatusConfigError   = "configuration_error"
	ChannelCheckinRunStatusNetworkFailed = "network_failed"
	ChannelCheckinRunStatusFailed        = "failed"
)

const (
	ChannelCheckinAuthTypeSystemAccessToken = "system_access_token"
	ChannelCheckinAuthTypeCookie            = "cookie"
)

// ChannelCheckinTask stores an administrator-configured request for a third-party channel check-in.
// Secret is encrypted before persistence and never exposed through JSON responses.
type ChannelCheckinTask struct {
	Id                int    `json:"id"`
	Name              string `json:"name" gorm:"type:varchar(128);not null;index"`
	Status            int    `json:"status" gorm:"default:1;index"`
	RequestURL        string `json:"request_url" gorm:"type:text;not null"`
	RequestMethod     string `json:"request_method" gorm:"type:varchar(8);default:'POST'"`
	AuthType          string `json:"auth_type" gorm:"type:varchar(32);not null"`
	EncryptedSecret   string `json:"-" gorm:"type:text;not null"`
	APIUser           string `json:"api_user" gorm:"type:varchar(64);not null"`
	EncryptedProxyURL string `json:"-" gorm:"type:text"`
	HasProxyURL       bool   `json:"has_proxy_url" gorm:"-"`
	TimeoutSeconds    int    `json:"timeout_seconds" gorm:"default:20"`
	RetryCount        int    `json:"retry_count" gorm:"default:1"`
	IntervalMinutes   int    `json:"interval_minutes" gorm:"default:1440"`
	LastRunAt         int64  `json:"last_run_at" gorm:"bigint"`
	NextRunAt         int64  `json:"next_run_at" gorm:"bigint;index"`
	LastRunStatus     string `json:"last_run_status" gorm:"type:varchar(32)"`
	LastRunMessage    string `json:"last_run_message" gorm:"type:varchar(512)"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint"`
}

func (ChannelCheckinTask) TableName() string {
	return "channel_checkin_tasks"
}

func (task *ChannelCheckinTask) NormalizeSchedule(now time.Time) {
	if task.IntervalMinutes < 60 {
		task.IntervalMinutes = 1440
	}
	if task.TimeoutSeconds < 5 || task.TimeoutSeconds > 120 {
		task.TimeoutSeconds = 20
	}
	if task.RetryCount < 0 {
		task.RetryCount = 0
	}
	if task.RetryCount > 2 {
		task.RetryCount = 2
	}
	task.RequestMethod = strings.ToUpper(strings.TrimSpace(task.RequestMethod))
	if task.RequestMethod == "" {
		task.RequestMethod = "POST"
	}
	if task.NextRunAt == 0 {
		task.NextRunAt = now.Add(time.Duration(task.IntervalMinutes) * time.Minute).Unix()
	}
}

// ChannelCheckinTaskLog intentionally excludes request headers, cookies and tokens.
type ChannelCheckinTaskLog struct {
	Id         int    `json:"id"`
	TaskId     int    `json:"task_id" gorm:"index;not null"`
	ChannelId  *int   `json:"channel_id,omitempty" gorm:"index"`
	Status     string `json:"status" gorm:"type:varchar(32);not null"`
	HTTPStatus int    `json:"http_status"`
	Message    string `json:"message" gorm:"type:varchar(512)"`
	CreatedAt  int64  `json:"created_at" gorm:"bigint;index"`
}

func (ChannelCheckinTaskLog) TableName() string {
	return "channel_checkin_task_logs"
}

func GetChannelCheckinTaskByID(id int) (*ChannelCheckinTask, error) {
	task := &ChannelCheckinTask{}
	if err := DB.First(task, id).Error; err != nil {
		return nil, err
	}
	task.HasProxyURL = task.EncryptedProxyURL != ""
	return task, nil
}

func ListChannelCheckinTasks() ([]ChannelCheckinTask, error) {
	tasks := make([]ChannelCheckinTask, 0)
	if err := DB.Order("id DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	for index := range tasks {
		tasks[index].HasProxyURL = tasks[index].EncryptedProxyURL != ""
	}
	return tasks, nil
}

func ListChannelCheckinTaskLogs(taskID int, limit int) ([]ChannelCheckinTaskLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	logs := make([]ChannelCheckinTaskLog, 0)
	err := DB.Where("task_id = ?", taskID).Order("id DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func FindChannelCheckinTaskChannelID(taskID int) *int {
	if taskID <= 0 {
		return nil
	}
	var channel Channel
	if err := DB.Select("id").Where("checkin_task_id = ?", taskID).First(&channel).Error; err != nil {
		return nil
	}
	return &channel.Id
}

func DisableChannelCheckinTask(taskID int, status, message string, httpStatus int, nextRunAt int64) error {
	return DB.Model(&ChannelCheckinTask{}).Where("id = ?", taskID).Updates(map[string]any{
		"last_run_at":      time.Now().Unix(),
		"next_run_at":      nextRunAt,
		"last_run_status":  status,
		"last_run_message": message,
		"updated_at":       time.Now().Unix(),
	}).Error
}
