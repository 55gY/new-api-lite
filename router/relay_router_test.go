package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/55gY/new-api-lite/constant"
	"github.com/gin-gonic/gin"
)

func TestModelListChannelType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		path    string
		headers map[string]string
		want    int
	}{
		{
			name: "OpenAI bearer request",
			path: "/v1/models",
			headers: map[string]string{
				"Authorization": "Bearer test-key",
			},
			want: constant.ChannelTypeOpenAI,
		},
		{
			name: "Anthropic request",
			path: "/v1/models",
			headers: map[string]string{
				"x-api-key":         "test-key",
				"anthropic-version": "2023-06-01",
			},
			want: constant.ChannelTypeAnthropic,
		},
		{
			name: "Gemini API key header",
			path: "/v1/models",
			headers: map[string]string{
				"x-goog-api-key": "test-key",
			},
			want: constant.ChannelTypeGemini,
		},
		{
			name: "Gemini API key query",
			path: "/v1/models?key=test-key",
			want: constant.ChannelTypeGemini,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, tt.path, nil)
			for key, value := range tt.headers {
				c.Request.Header.Set(key, value)
			}
			if got := modelListChannelType(c); got != tt.want {
				t.Fatalf("model list channel type = %d, want %d", got, tt.want)
			}
		})
	}
}
