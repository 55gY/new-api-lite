package model

import (
	"testing"

	"github.com/55gY/new-api-lite/common"
)

func TestUpdateOptionMapUpdatesEnableRateLimitFlags(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()

	originalAPI := common.GlobalApiRateLimitEnable
	originalWeb := common.GlobalWebRateLimitEnable
	originalCritical := common.CriticalRateLimitEnable
	originalSearch := common.SearchRateLimitEnable
	t.Cleanup(func() {
		common.GlobalApiRateLimitEnable = originalAPI
		common.GlobalWebRateLimitEnable = originalWeb
		common.CriticalRateLimitEnable = originalCritical
		common.SearchRateLimitEnable = originalSearch
	})

	common.GlobalApiRateLimitEnable = true
	common.GlobalWebRateLimitEnable = true
	common.CriticalRateLimitEnable = true
	common.SearchRateLimitEnable = true

	for _, key := range []string{
		"GlobalApiRateLimitEnable",
		"GlobalWebRateLimitEnable",
		"CriticalRateLimitEnable",
		"SearchRateLimitEnable",
	} {
		if err := updateOptionMap(key, "false"); err != nil {
			t.Fatalf("updateOptionMap(%s) returned error: %v", key, err)
		}
	}

	if common.GlobalApiRateLimitEnable || common.GlobalWebRateLimitEnable || common.CriticalRateLimitEnable || common.SearchRateLimitEnable {
		t.Fatal("rate limit enable flags were not disabled by updateOptionMap")
	}
}
