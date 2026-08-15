package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/model"

	"gorm.io/gorm"
)

const (
	channelCheckinSchedulerInterval = 5 * time.Minute
	channelCheckinResponseLimit     = 64 * 1024
)

var (
	channelCheckinTaskSchedulerOnce sync.Once
	channelCheckinTaskRunLock       sync.Mutex
)

type ChannelCheckinTaskInput struct {
	Name            string `json:"name"`
	Status          int    `json:"status"`
	RequestURL      string `json:"request_url"`
	RequestMethod   string `json:"request_method"`
	AuthType        string `json:"auth_type"`
	Secret          string `json:"secret"`
	APIUser         string `json:"api_user"`
	ProxyURL        string `json:"proxy_url"`
	ClearProxy      bool   `json:"clear_proxy"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	RetryCount      int    `json:"retry_count"`
	IntervalMinutes int    `json:"interval_minutes"`
}

type ChannelCheckinRunResult struct {
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status"`
	Message    string `json:"message"`
	Manual     bool   `json:"manual"`
}

func ValidateChannelCheckinTaskInput(input *ChannelCheckinTaskInput) error {
	if input == nil {
		return fmt.Errorf("invalid check-in task")
	}
	input.Name = strings.TrimSpace(input.Name)
	input.RequestURL = strings.TrimSpace(input.RequestURL)
	input.APIUser = strings.TrimSpace(input.APIUser)
	input.ProxyURL = strings.TrimSpace(input.ProxyURL)
	input.AuthType = strings.TrimSpace(input.AuthType)
	input.RequestMethod = strings.ToUpper(strings.TrimSpace(input.RequestMethod))
	if input.Name == "" || len(input.Name) > 128 {
		return fmt.Errorf("invalid task name")
	}
	if input.RequestURL == "" {
		return fmt.Errorf("check-in request URL is required")
	}
	parsedURL, err := url.Parse(input.RequestURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return fmt.Errorf("check-in request URL must be a valid HTTPS URL")
	}
	if err := common.DefaultSSRFProtection.ValidateURL(input.RequestURL); err != nil {
		return fmt.Errorf("unsafe check-in request URL: %w", err)
	}
	if input.RequestMethod == "" {
		input.RequestMethod = http.MethodPost
	}
	if input.RequestMethod != http.MethodPost && input.RequestMethod != http.MethodGet {
		return fmt.Errorf("check-in request method must be GET or POST")
	}
	if input.AuthType != model.ChannelCheckinAuthTypeSystemAccessToken && input.AuthType != model.ChannelCheckinAuthTypeCookie {
		return fmt.Errorf("unsupported check-in authentication type")
	}
	if input.APIUser == "" || len(input.APIUser) > 64 {
		return fmt.Errorf("valid api user is required")
	}
	if input.ProxyURL != "" {
		proxyURL, err := url.Parse(input.ProxyURL)
		if err != nil || proxyURL.Host == "" {
			return fmt.Errorf("invalid fixed proxy URL")
		}
		switch proxyURL.Scheme {
		case "http", "https", "socks5", "socks5h":
		default:
			return fmt.Errorf("unsupported fixed proxy URL scheme")
		}
	}
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 20
	}
	if input.TimeoutSeconds < 5 || input.TimeoutSeconds > 120 {
		return fmt.Errorf("timeout must be between 5 and 120 seconds")
	}
	if input.RetryCount < 0 || input.RetryCount > 2 {
		return fmt.Errorf("retry count must be between 0 and 2")
	}
	if input.IntervalMinutes == 0 {
		input.IntervalMinutes = 1440
	}
	if input.IntervalMinutes < 60 {
		return fmt.Errorf("interval must be at least 60 minutes")
	}
	if input.Status != model.ChannelCheckinTaskStatusDisabled && input.Status != model.ChannelCheckinTaskStatusEnabled {
		return fmt.Errorf("invalid task status")
	}
	return nil
}

func ApplyChannelCheckinTaskInput(task *model.ChannelCheckinTask, input *ChannelCheckinTaskInput, updateSecret bool) error {
	if err := ValidateChannelCheckinTaskInput(input); err != nil {
		return err
	}
	previousStatus := task.Status
	previousIntervalMinutes := task.IntervalMinutes
	isExistingTask := task.Id > 0
	if updateSecret {
		if strings.TrimSpace(input.Secret) == "" {
			return fmt.Errorf("check-in credential is required")
		}
		encryptedSecret, err := common.EncryptSecret(input.Secret)
		if err != nil {
			return fmt.Errorf("encrypt check-in credential: %w", err)
		}
		task.EncryptedSecret = encryptedSecret
	}
	task.Name = input.Name
	task.Status = input.Status
	task.RequestURL = input.RequestURL
	task.RequestMethod = input.RequestMethod
	task.AuthType = input.AuthType
	task.APIUser = input.APIUser
	if input.ClearProxy {
		task.EncryptedProxyURL = ""
		task.HasProxyURL = false
	} else if input.ProxyURL != "" {
		encryptedProxyURL, err := common.EncryptSecret(input.ProxyURL)
		if err != nil {
			return fmt.Errorf("encrypt fixed proxy URL: %w", err)
		}
		task.EncryptedProxyURL = encryptedProxyURL
		task.HasProxyURL = true
	}
	task.TimeoutSeconds = input.TimeoutSeconds
	task.RetryCount = input.RetryCount
	task.IntervalMinutes = input.IntervalMinutes
	if task.Status == model.ChannelCheckinTaskStatusDisabled {
		task.NextRunAt = 0
	} else if !isExistingTask || previousStatus != task.Status || previousIntervalMinutes != task.IntervalMinutes || task.NextRunAt == 0 {
		// 启用、重新启用或修改计划间隔后，从当前时间重新计算下一次执行时间。
		task.NextRunAt = 0
	}
	if previousStatus == model.ChannelCheckinTaskStatusDisabled && task.Status == model.ChannelCheckinTaskStatusEnabled {
		// 管理员人工处理后重新启用任务时，清除旧的阻断状态，保留历史日志供追溯。
		task.LastRunStatus = ""
		task.LastRunMessage = ""
	}
	task.NormalizeSchedule(time.Now())
	return nil
}

func StartChannelCheckinTaskScheduler() {
	channelCheckinTaskSchedulerOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			common.SysLog("channel check-in task scheduler started")
			runDueChannelCheckinTasks()
			ticker := time.NewTicker(channelCheckinSchedulerInterval)
			defer ticker.Stop()
			for range ticker.C {
				runDueChannelCheckinTasks()
			}
		}()
	})
}

func runDueChannelCheckinTasks() {
	now := time.Now().Unix()
	tasks := make([]model.ChannelCheckinTask, 0)
	if err := model.DB.Where("status = ? AND next_run_at > 0 AND next_run_at <= ?", model.ChannelCheckinTaskStatusEnabled, now).
		Order("next_run_at ASC").Limit(20).Find(&tasks).Error; err != nil {
		common.SysError("load due channel check-in tasks failed: " + err.Error())
		return
	}
	for _, task := range tasks {
		if _, err := RunChannelCheckinTask(task.Id); err != nil {
			common.SysError(fmt.Sprintf("channel check-in task #%d failed: %s", task.Id, err.Error()))
		}
	}
}

func RunChannelCheckinTask(taskID int) (*ChannelCheckinRunResult, error) {
	channelCheckinTaskRunLock.Lock()
	defer channelCheckinTaskRunLock.Unlock()

	task, err := model.GetChannelCheckinTaskByID(taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != model.ChannelCheckinTaskStatusEnabled {
		return nil, fmt.Errorf("check-in task is disabled")
	}
	secret, err := common.DecryptSecret(task.EncryptedSecret)
	if err != nil {
		result := &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusConfigError, Message: "凭据无法解密，请重新保存任务凭据"}
		persistChannelCheckinTaskResult(task, result, true)
		return result, nil
	}

	result := executeChannelCheckinTask(task, secret)
	persistChannelCheckinTaskResult(task, result, shouldSuspendChannelCheckinTask(result))
	return result, nil
}

func shouldSuspendChannelCheckinTask(result *ChannelCheckinRunResult) bool {
	if result == nil {
		return false
	}
	return result.Manual ||
		result.Status == model.ChannelCheckinRunStatusAuthFailed ||
		result.Status == model.ChannelCheckinRunStatusConfigError
}

func executeChannelCheckinTask(task *model.ChannelCheckinTask, secret string) *ChannelCheckinRunResult {
	attempts := task.RetryCount + 1
	var result *ChannelCheckinRunResult
	for attempt := 0; attempt < attempts; attempt++ {
		result = executeChannelCheckinRequest(task, secret)
		if result.Status != model.ChannelCheckinRunStatusNetworkFailed || attempt+1 >= attempts {
			return result
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return result
}

func executeChannelCheckinRequest(task *model.ChannelCheckinTask, secret string) *ChannelCheckinRunResult {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(task.TimeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, task.RequestMethod, task.RequestURL, nil)
	if err != nil {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusConfigError, Message: "签到请求配置无效"}
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("New-Api-User", task.APIUser)
	switch task.AuthType {
	case model.ChannelCheckinAuthTypeSystemAccessToken:
		req.Header.Set("Authorization", "Bearer "+secret)
	case model.ChannelCheckinAuthTypeCookie:
		req.Header.Set("Cookie", secret)
	}

	proxyURL := ""
	if task.EncryptedProxyURL != "" {
		proxyURL, err = common.DecryptSecret(task.EncryptedProxyURL)
		if err != nil {
			return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusConfigError, Message: "固定代理凭据无法解密，请重新保存"}
		}
	}
	baseClient, err := GetHttpClientWithProxy(proxyURL)
	if err != nil {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusConfigError, Message: "固定代理配置无效"}
	}
	client := *baseClient
	client.Timeout = time.Duration(task.TimeoutSeconds) * time.Second
	response, err := client.Do(req)
	if err != nil {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusNetworkFailed, Message: "网络请求失败"}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, channelCheckinResponseLimit))
	if err != nil {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusNetworkFailed, HTTPStatus: response.StatusCode, Message: "读取签到响应失败"}
	}
	return classifyChannelCheckinResponse(response.StatusCode, body)
}

func classifyChannelCheckinResponse(httpStatus int, body []byte) *ChannelCheckinRunResult {
	lowerBody := strings.ToLower(string(body))
	manualIndicators := []string{"captcha", "turnstile", "cloudflare", "challenge", "access denied", "cf-error", "验证码", "人机验证", "安全验证"}
	for _, indicator := range manualIndicators {
		if strings.Contains(lowerBody, indicator) {
			return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusManualAction, HTTPStatus: httpStatus, Message: "站点要求人工验证", Manual: true}
		}
	}
	if httpStatus == http.StatusUnauthorized {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusAuthFailed, HTTPStatus: httpStatus, Message: "签到凭据已失效"}
	}
	if httpStatus == http.StatusForbidden || httpStatus == http.StatusTooManyRequests {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusManualAction, HTTPStatus: httpStatus, Message: "站点拒绝请求，需人工处理", Manual: true}
	}
	if httpStatus >= http.StatusInternalServerError {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusNetworkFailed, HTTPStatus: httpStatus, Message: "站点服务暂时不可用"}
	}
	if httpStatus < http.StatusOK || httpStatus >= http.StatusMultipleChoices {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusFailed, HTTPStatus: httpStatus, Message: "签到请求失败"}
	}
	if strings.Contains(lowerBody, "already") || strings.Contains(lowerBody, "已签到") || strings.Contains(lowerBody, "今日已签") {
		return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusAlreadyDone, HTTPStatus: httpStatus, Message: "今日已完成签到"}
	}
	return &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusSuccess, HTTPStatus: httpStatus, Message: "签到请求已成功执行"}
}

func persistChannelCheckinTaskResult(task *model.ChannelCheckinTask, result *ChannelCheckinRunResult, disableTask bool) {
	if task == nil || result == nil {
		return
	}
	now := time.Now()
	nextRunAt := now.Add(time.Duration(task.IntervalMinutes) * time.Minute).Unix()
	updates := map[string]any{
		"last_run_at":      now.Unix(),
		"next_run_at":      nextRunAt,
		"last_run_status":  result.Status,
		"last_run_message": result.Message,
		"updated_at":       now.Unix(),
	}
	if disableTask {
		updates["status"] = model.ChannelCheckinTaskStatusDisabled
		updates["next_run_at"] = int64(0)
	}
	if err := model.DB.Model(&model.ChannelCheckinTask{}).Where("id = ?", task.Id).Updates(updates).Error; err != nil {
		common.SysError(fmt.Sprintf("update channel check-in task #%d result failed: %s", task.Id, err.Error()))
	}
	logEntry := model.ChannelCheckinTaskLog{
		TaskId:     task.Id,
		ChannelId:  model.FindChannelCheckinTaskChannelID(task.Id),
		Status:     result.Status,
		HTTPStatus: result.HTTPStatus,
		Message:    result.Message,
		CreatedAt:  now.Unix(),
	}
	if err := model.DB.Create(&logEntry).Error; err != nil {
		common.SysError(fmt.Sprintf("create channel check-in task log #%d failed: %s", task.Id, err.Error()))
	}
}

func DeleteChannelCheckinTask(id int) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Channel{}).Where("checkin_task_id = ?", id).Update("checkin_task_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", id).Delete(&model.ChannelCheckinTaskLog{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.ChannelCheckinTask{}, id).Error
	})
}
