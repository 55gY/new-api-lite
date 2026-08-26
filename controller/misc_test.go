package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/55gY/new-api-lite/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetStatusOmitsLegacyLiteContractFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	GetStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	for _, key := range []string{
		"enable_task",
		"checkin_enabled",
		"quota_per_unit",
		"display_in_currency",
		"quota_display_type",
		"custom_currency_symbol",
		"custom_currency_exchange_rate",
		"usd_exchange_rate",
		"price",
	} {
		require.NotContainsf(t, response.Data, key, "legacy Lite field %q must not be public", key)
	}
}
