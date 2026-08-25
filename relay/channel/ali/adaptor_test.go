package ali

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/55gY/new-api-lite/dto"
	relaycommon "github.com/55gY/new-api-lite/relay/common"
	"github.com/55gY/new-api-lite/relay/constant"
	"github.com/gin-gonic/gin"
)

func TestMappedAliImageModelUsesUpstreamProtocol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := &relaycommon.RelayInfo{
		RelayMode:       constant.RelayModeImagesGenerations,
		OriginModelName: "customer-image-model",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://dashscope.aliyuncs.com",
			UpstreamModelName: "qwen-image-3.0-pro",
		},
	}

	adaptor := &Adaptor{}
	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("get request URL: %v", err)
	}
	if want := "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"; url != want {
		t.Fatalf("request URL = %q, want %q", url, want)
	}

	header := http.Header{}
	if err := adaptor.SetupRequestHeader(c, &header, info); err != nil {
		t.Fatalf("setup request header: %v", err)
	}
	if got := header.Get("X-DashScope-Async"); got != "" {
		t.Fatalf("X-DashScope-Async = %q, want omitted for synchronous model", got)
	}
}

func TestRequestOpenAI2AliTopP(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	tests := []struct {
		name string
		topP *float64
		want *float64
	}{
		{name: "omitted is not injected", topP: nil, want: nil},
		{name: "in range is preserved", topP: value(0.8), want: value(0.8)},
		{name: "one is clamped", topP: value(1), want: value(0.99)},
		{name: "above one is clamped", topP: value(1.5), want: value(0.99)},
		{name: "zero is clamped", topP: value(0), want: value(0.01)},
		{name: "negative is clamped", topP: value(-0.3), want: value(0.01)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requestOpenAI2Ali(dto.GeneralOpenAIRequest{Model: "qwen-plus", TopP: tt.topP})
			if (got.TopP == nil) != (tt.want == nil) {
				t.Fatalf("top_p = %v, want %v", got.TopP, tt.want)
			}
			if got.TopP != nil && *got.TopP != *tt.want {
				t.Fatalf("top_p = %v, want %v", *got.TopP, *tt.want)
			}
		})
	}
}
