package controller

import (
	"strings"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/setting/system_setting"
)

func paymentReturnPath(suffix string) string {
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	return base + common.ThemeAwarePath(suffix)
}
