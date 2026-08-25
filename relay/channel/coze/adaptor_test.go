package coze

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCozeChatResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantErr    string
		wantChatID string
		wantConvID string
	}{
		{
			name:    "rejects malformed JSON",
			body:    `{"code":`,
			wantErr: "decode Coze chat response",
		},
		{
			name:    "returns upstream error",
			body:    `{"code":4001,"msg":"invalid bot"}`,
			wantErr: "invalid bot",
		},
		{
			name:       "parses successful response",
			body:       `{"code":0,"data":{"id":"chat_123","conversation_id":"conv_456"}}`,
			wantChatID: "chat_123",
			wantConvID: "conv_456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := parseCozeChatResponse([]byte(tt.body))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, response)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, response)
			require.Equal(t, tt.wantChatID, response.Data.Id)
			require.Equal(t, tt.wantConvID, response.Data.ConversationId)
		})
	}
}
