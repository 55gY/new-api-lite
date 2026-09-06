package setting

import (
	"fmt"
	"strings"
	"sync"

	"github.com/55gY/new-api-lite/common"
)

var userUsableGroups = map[string]string{
	"default": "默认分组",
	"vip":     "vip分组",
}
var userUsableGroupsMutex sync.RWMutex

func GetUserUsableGroupsCopy() map[string]string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	copyUserUsableGroups := make(map[string]string)
	for k, v := range userUsableGroups {
		copyUserUsableGroups[k] = v
	}
	return copyUserUsableGroups
}

func UserUsableGroups2JSONString() string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	jsonBytes, err := common.Marshal(userUsableGroups)
	if err != nil {
		common.SysLog("error marshalling user groups: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateUserUsableGroupsByJSONString(jsonStr string) error {
	var nextGroups map[string]string
	if err := common.Unmarshal([]byte(jsonStr), &nextGroups); err != nil {
		return err
	}
	userUsableGroupsMutex.Lock()
	defer userUsableGroupsMutex.Unlock()
	userUsableGroups = nextGroups
	return nil
}

func CheckUserUsableGroups(jsonStr string) error {
	var groups map[string]string
	if err := common.Unmarshal([]byte(jsonStr), &groups); err != nil {
		return fmt.Errorf("用户可用分组必须是有效的 JSON 对象: %w", err)
	}
	if len(groups) > 100 {
		return fmt.Errorf("用户可用分组数量不能超过 100 个")
	}
	for name, description := range groups {
		if strings.TrimSpace(name) == "" || len(name) > 64 {
			return fmt.Errorf("用户可用分组名称不能为空且不能超过 64 个字符")
		}
		if len(description) > 128 {
			return fmt.Errorf("用户可用分组描述不能超过 128 个字符")
		}
	}
	return nil
}

func GetUsableGroupDescription(groupName string) string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	if desc, ok := userUsableGroups[groupName]; ok {
		return desc
	}
	return groupName
}
