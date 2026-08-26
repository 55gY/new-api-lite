package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/55gY/new-api-lite/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetStatusExposesCoreLiteContractFields(t *testing.T) {
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
		"version",
		"system_name",
		"register_enabled",
		"password_login_enabled",
		"enable_data_export",
	} {
		require.Containsf(t, response.Data, key, "core Lite field %q must be public", key)
	}
}
