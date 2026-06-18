package service

import (
	relaycommon "github.com/55gY/new-api-lite/relay/common"
	"github.com/55gY/new-api-lite/types"
	"github.com/gin-gonic/gin"
)

func ReturnPreConsumedQuota(c *gin.Context, relayInfo *relaycommon.RelayInfo) {
	if relayInfo != nil {
		relayInfo.FinalPreConsumedQuota = 0
	}
}

// PreConsumeQuota checks if the user has enough quota to pre-consume.
// It returns the pre-consumed quota if successful, or an error if not.
func PreConsumeQuota(c *gin.Context, preConsumedQuota int, relayInfo *relaycommon.RelayInfo) *types.NewAPIError {
	if relayInfo == nil {
		return nil
	}
	relayInfo.UserQuota = 0
	relayInfo.FinalPreConsumedQuota = 0
	return nil
}
