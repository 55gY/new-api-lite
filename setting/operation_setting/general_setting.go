package operation_setting

import "github.com/55gY/new-api-lite/setting/config"

type GeneralSetting struct {
	PingIntervalEnabled bool `json:"ping_interval_enabled"`
	PingIntervalSeconds int  `json:"ping_interval_seconds"`
}

var generalSetting = GeneralSetting{
	PingIntervalEnabled: false,
	PingIntervalSeconds: 60,
}

func init() {
	config.GlobalConfig.Register("general_setting", &generalSetting)
}

func GetGeneralSetting() *GeneralSetting {
	return &generalSetting
}
