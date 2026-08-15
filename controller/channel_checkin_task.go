package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/model"
	"github.com/55gY/new-api-lite/service"

	"github.com/gin-gonic/gin"
)

type channelCheckinTaskRequest struct {
	Id int `json:"id"`
	service.ChannelCheckinTaskInput
}

func validateChannelCheckinTaskAssociation(taskID *int) error {
	if taskID == nil || *taskID == 0 {
		return nil
	}
	if *taskID < 0 {
		return errInvalidCheckinTask
	}
	_, err := model.GetChannelCheckinTaskByID(*taskID)
	return err
}

var errInvalidCheckinTask = &channelCheckinTaskValidationError{message: "invalid check-in task"}

type channelCheckinTaskValidationError struct {
	message string
}

func (e *channelCheckinTaskValidationError) Error() string {
	return e.message
}

func ListChannelCheckinTasks(c *gin.Context) {
	tasks, err := model.ListChannelCheckinTasks()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": tasks})
}

func GetChannelCheckinTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid check-in task id")
		return
	}
	task, err := model.GetChannelCheckinTaskByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, task)
}

func AddChannelCheckinTask(c *gin.Context) {
	req := channelCheckinTaskRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	task := &model.ChannelCheckinTask{}
	if err := service.ApplyChannelCheckinTaskInput(task, &req.ChannelCheckinTaskInput, true); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	now := time.Now().Unix()
	task.CreatedAt = now
	task.UpdatedAt = now
	if err := model.DB.Create(task).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, task)
}

func UpdateChannelCheckinTask(c *gin.Context) {
	req := channelCheckinTaskRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Id <= 0 {
		common.ApiErrorMsg(c, "invalid check-in task id")
		return
	}
	task, err := model.GetChannelCheckinTaskByID(req.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	updateSecret := strings.TrimSpace(req.Secret) != ""
	if err := service.ApplyChannelCheckinTaskInput(task, &req.ChannelCheckinTaskInput, updateSecret); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	task.UpdatedAt = time.Now().Unix()
	if err := model.DB.Save(task).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, task)
}

func DeleteChannelCheckinTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid check-in task id")
		return
	}
	if err := service.DeleteChannelCheckinTask(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func RunChannelCheckinTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid check-in task id")
		return
	}
	result, err := service.RunChannelCheckinTask(id)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": result.Status == model.ChannelCheckinRunStatusSuccess || result.Status == model.ChannelCheckinRunStatusAlreadyDone,
		"message": result.Message,
		"data":    result,
	})
}

func GetChannelCheckinTaskLogs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid check-in task id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	logs, err := model.ListChannelCheckinTaskLogs(id, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": logs})
}
