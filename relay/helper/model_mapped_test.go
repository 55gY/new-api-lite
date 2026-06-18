package helper

import (
	"testing"

	"github.com/55gY/new-api-lite/relay/common"
	relayconstant "github.com/55gY/new-api-lite/relay/constant"
	"github.com/55gY/new-api-lite/setting/ratio_setting"
	"github.com/55gY/new-api-lite/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelMappedTestRequest struct {
	model string
}

func (r *modelMappedTestRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{}
}

func (r *modelMappedTestRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *modelMappedTestRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.model = modelName
	}
}

func newModelMappedTestContext(modelMapping string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set("model_mapping", modelMapping)
	return c
}

func TestModelMappedHelperReverseMapping(t *testing.T) {
	c := newModelMappedTestContext(`{"a":"b"}`)
	info := &common.RelayInfo{
		OriginModelName: "b",
		ChannelMeta:     &common.ChannelMeta{UpstreamModelName: "b"},
	}
	request := &modelMappedTestRequest{model: "b"}

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "a", info.UpstreamModelName)
	require.Equal(t, "a", request.model)
}

func TestModelMappedHelperDoesNotMapOriginalModelForward(t *testing.T) {
	c := newModelMappedTestContext(`{"a":"b"}`)
	info := &common.RelayInfo{
		OriginModelName: "a",
		ChannelMeta:     &common.ChannelMeta{UpstreamModelName: "a"},
	}
	request := &modelMappedTestRequest{model: "a"}

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	require.False(t, info.IsModelMapped)
	require.Equal(t, "a", info.UpstreamModelName)
	require.Equal(t, "a", request.model)
}

func TestModelMappedHelperReverseChainMapping(t *testing.T) {
	c := newModelMappedTestContext(`{"a":"b","b":"c"}`)
	info := &common.RelayInfo{
		OriginModelName: "c",
		ChannelMeta:     &common.ChannelMeta{UpstreamModelName: "c"},
	}
	request := &modelMappedTestRequest{model: "c"}

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "a", info.UpstreamModelName)
	require.Equal(t, "a", request.model)
}

func TestModelMappedHelperReverseMappingCycle(t *testing.T) {
	c := newModelMappedTestContext(`{"a":"b","b":"a"}`)
	info := &common.RelayInfo{
		OriginModelName: "a",
		ChannelMeta:     &common.ChannelMeta{UpstreamModelName: "a"},
	}
	request := &modelMappedTestRequest{model: "a"}

	err := ModelMappedHelper(c, info, request)

	require.EqualError(t, err, "model_mapping_contains_cycle")
}

func TestModelMappedHelperReverseMappingSelfMapping(t *testing.T) {
	c := newModelMappedTestContext(`{"a":"a"}`)
	info := &common.RelayInfo{
		OriginModelName: "a",
		ChannelMeta:     &common.ChannelMeta{UpstreamModelName: "a"},
	}
	request := &modelMappedTestRequest{model: "a"}

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	require.False(t, info.IsModelMapped)
	require.Equal(t, "a", info.UpstreamModelName)
	require.Equal(t, "a", request.model)
}

func TestModelMappedHelperReverseMappingResponsesCompact(t *testing.T) {
	c := newModelMappedTestContext(`{"a":"b"}`)
	info := &common.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponsesCompact,
		OriginModelName: ratio_setting.WithCompactModelSuffix("b"),
		ChannelMeta:     &common.ChannelMeta{UpstreamModelName: ratio_setting.WithCompactModelSuffix("b")},
	}
	request := &modelMappedTestRequest{model: ratio_setting.WithCompactModelSuffix("b")}

	err := ModelMappedHelper(c, info, request)

	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "a", info.UpstreamModelName)
	require.Equal(t, ratio_setting.WithCompactModelSuffix("a"), info.OriginModelName)
	require.Equal(t, "a", request.model)
}
