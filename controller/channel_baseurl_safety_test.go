package controller

import (
	"strings"
	"testing"
)

// 渠道 BaseURL 安全校验：只拦截云实例元数据/链路本地地址，
// 不得误伤自建上游（局域网 IP、回环地址、非标准端口）。
func TestValidateChannelBaseURLSafetyAllowsSelfHostedUpstreams(t *testing.T) {
	allowed := []string{
		"",                                // 未填写
		"https://api.openai.com/v1",       // 公网标准端口
		"http://127.0.0.1:3000",           // 本机自建上游
		"http://127.0.0.1:9999/v1",        // 本机非标准端口
		"http://192.168.1.50:3000/v1",     // 局域网自建上游
		"http://10.0.0.8:8317",            // 内网非标准端口
		"http://172.16.5.4:23000/v1",      // 内网非标准端口
		"https://relay.example.com:23000", // 公网域名 + 非标准端口
		"http://localhost:3000",           // localhost
		"not a url",                       // 非法 URL 交由后续流程处理
	}
	for _, u := range allowed {
		if err := validateChannelBaseURLSafety(u); err != nil {
			t.Fatalf("BaseURL %q 属于合法自建/常规用法，不应被拦截，实际报错：%v", u, err)
		}
	}
}

func TestValidateChannelBaseURLSafetyBlocksMetadataEndpoints(t *testing.T) {
	blocked := []string{
		"http://169.254.169.254/latest/meta-data/", // AWS/阿里云等实例元数据
		"http://169.254.169.254",
		"http://[fe80::1]/v1",                  // IPv6 链路本地
		"http://metadata.google.internal/v1",   // GCP 元数据域名
		"http://metadata.goog/computeMetadata", // GCP 备用域名
	}
	for _, u := range blocked {
		err := validateChannelBaseURLSafety(u)
		if err == nil {
			t.Fatalf("BaseURL %q 指向实例元数据/链路本地地址，应被拦截", u)
		}
		if !strings.Contains(err.Error(), "元数据") && !strings.Contains(err.Error(), "链路本地") {
			t.Fatalf("BaseURL %q 的报错应说明原因，实际：%v", u, err)
		}
	}
}
