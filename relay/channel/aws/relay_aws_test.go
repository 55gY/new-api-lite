package aws

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/55gY/new-api-lite/common"
	relaycommon "github.com/55gY/new-api-lite/relay/common"
	"github.com/55gY/new-api-lite/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNewAwsInvokeContextPropagatesParentCancellation(t *testing.T) {
	originalTimeout := common.RelayTimeout
	t.Cleanup(func() { common.RelayTimeout = originalTimeout })
	common.RelayTimeout = 0

	parent, cancelParent := context.WithCancel(context.Background())
	ctx, cancel := newAwsInvokeContext(parent)
	defer cancel()
	cancelParent()

	select {
	case <-ctx.Done():
		require.ErrorIs(t, ctx.Err(), context.Canceled)
	default:
		t.Fatal("AWS invocation context did not inherit parent cancellation")
	}
}

func TestNewAwsInvokeContextKeepsRelayTimeout(t *testing.T) {
	originalTimeout := common.RelayTimeout
	t.Cleanup(func() { common.RelayTimeout = originalTimeout })
	common.RelayTimeout = 1

	ctx, cancel := newAwsInvokeContext(context.Background())
	defer cancel()
	_, hasDeadline := ctx.Deadline()
	require.True(t, hasDeadline)
}

func TestNewAwsInvokeErrorSkipsRetryAfterClientCancellation(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	err := newAwsInvokeError(canceled, context.Canceled, "InvokeModel")
	require.True(t, types.IsSkipRetryError(err))

	active := context.Background()
	err = newAwsInvokeError(active, context.DeadlineExceeded, "InvokeModel")
	require.False(t, types.IsSkipRetryError(err))
}

func TestDoAwsClientRequest_AppliesRuntimeHeaderOverrideToAnthropicBeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName:           "claude-3-5-sonnet-20240620",
		IsStream:                  false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"anthropic-beta": "computer-use-2025-01-24",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "access-key|secret-key|us-east-1",
			UpstreamModelName: "claude-3-5-sonnet-20240620",
		},
	}

	requestBody := bytes.NewBufferString(`{"messages":[{"role":"user","content":"hello"}],"max_tokens":128}`)
	adaptor := &Adaptor{}

	_, err := doAwsClientRequest(ctx, info, adaptor, requestBody)
	require.NoError(t, err)

	awsReq, ok := adaptor.AwsReq.(*bedrockruntime.InvokeModelInput)
	require.True(t, ok)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(awsReq.Body, &payload))

	anthropicBeta, exists := payload["anthropic_beta"]
	require.True(t, exists)

	values, ok := anthropicBeta.([]any)
	require.True(t, ok)
	require.Equal(t, []any{"computer-use-2025-01-24"}, values)
}
