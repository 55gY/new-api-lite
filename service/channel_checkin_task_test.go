package service

import (
	"testing"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/model"
)

func TestClassifyChannelCheckinResponse(t *testing.T) {
	testCases := []struct {
		name       string
		httpStatus int
		body       string
		status     string
		manual     bool
	}{
		{name: "success", httpStatus: 200, body: `{"success":true}`, status: model.ChannelCheckinRunStatusSuccess},
		{name: "already checked in", httpStatus: 200, body: `今日已签到`, status: model.ChannelCheckinRunStatusAlreadyDone},
		{name: "captcha requires manual action", httpStatus: 200, body: `<html>captcha challenge</html>`, status: model.ChannelCheckinRunStatusManualAction, manual: true},
		{name: "forbidden requires manual action", httpStatus: 403, body: `forbidden`, status: model.ChannelCheckinRunStatusManualAction, manual: true},
		{name: "unauthorized credential failure", httpStatus: 401, body: `unauthorized`, status: model.ChannelCheckinRunStatusAuthFailed},
		{name: "server error can retry", httpStatus: 503, body: `unavailable`, status: model.ChannelCheckinRunStatusNetworkFailed},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := classifyChannelCheckinResponse(testCase.httpStatus, []byte(testCase.body))
			if result.Status != testCase.status {
				t.Fatalf("status = %q, want %q", result.Status, testCase.status)
			}
			if result.Manual != testCase.manual {
				t.Fatalf("manual = %t, want %t", result.Manual, testCase.manual)
			}
		})
	}
}

func TestChannelCheckinSecretEncryption(t *testing.T) {
	originalSecret := common.CryptoSecret
	common.CryptoSecret = "checkin-task-test-secret"
	t.Cleanup(func() { common.CryptoSecret = originalSecret })

	ciphertext, err := common.EncryptSecret("sensitive-cookie-value")
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	if ciphertext == "sensitive-cookie-value" {
		t.Fatal("ciphertext must not equal plaintext")
	}
	plaintext, err := common.DecryptSecret(ciphertext)
	if err != nil {
		t.Fatalf("decrypt secret: %v", err)
	}
	if plaintext != "sensitive-cookie-value" {
		t.Fatalf("plaintext = %q", plaintext)
	}
}

func TestShouldSuspendChannelCheckinTask(t *testing.T) {
	testCases := []struct {
		name    string
		result  *ChannelCheckinRunResult
		expected bool
	}{
		{name: "success remains scheduled", result: &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusSuccess}, expected: false},
		{name: "network failure remains scheduled", result: &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusNetworkFailed}, expected: false},
		{name: "manual verification suspends", result: &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusManualAction, Manual: true}, expected: true},
		{name: "authentication failure suspends", result: &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusAuthFailed}, expected: true},
		{name: "configuration failure suspends", result: &ChannelCheckinRunResult{Status: model.ChannelCheckinRunStatusConfigError}, expected: true},
		{name: "nil result does not suspend", result: nil, expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if actual := shouldSuspendChannelCheckinTask(testCase.result); actual != testCase.expected {
				t.Fatalf("shouldSuspendChannelCheckinTask() = %t, want %t", actual, testCase.expected)
			}
		})
	}
}
