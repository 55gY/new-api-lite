package controller

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// 真实复现样本：上游被阿里云 WAF 拦截时返回的滑块验证页（状态码 200、Content-Type: text/html）。
const aliyunWAFPageSample = `<!doctypehtml><meta charset="UTF-8"><meta name="aliyun_waf_aa"content="ff926c7f07e45e2e487a29a6197d3460">` +
	`<title></title><script>initAliyunCaptcha(e)</script><div>访问验证</div>`

func newTestResponse(statusCode int, contentType string, body string) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestLocalizeChannelTestErrorTextHTMLResponse(t *testing.T) {
	raw := "invalid character '<' looking for beginning of value"
	got := LocalizeChannelTestErrorText(raw)

	if !strings.Contains(got, "HTML") {
		t.Fatalf("期望说明中包含 HTML 关键信息，实际：%s", got)
	}
	if !strings.Contains(got, "BaseURL") {
		t.Fatalf("期望说明中包含 BaseURL 排查提示，实际：%s", got)
	}
	// 必须保留原始错误，便于进一步排查
	if !strings.Contains(got, raw) {
		t.Fatalf("期望保留原始错误，实际：%s", got)
	}
}

func TestLocalizeChannelTestErrorTextStatusCodes(t *testing.T) {
	cases := map[string]string{
		"bad response status code 401": "401",
		"bad response status code 404": "404",
		"bad response status code 429": "429",
	}
	for raw, want := range cases {
		got := LocalizeChannelTestErrorText(raw)
		if !strings.Contains(got, want) {
			t.Fatalf("原始错误 %q 期望包含 %q，实际：%s", raw, want, got)
		}
		if got == raw {
			t.Fatalf("原始错误 %q 未被中文化", raw)
		}
	}
}

// 真实样本：上游应用层拒绝“非指定客户端”的调用（状态码 401，但原因不是密钥无效）。
func TestLocalizeChannelTestErrorTextUnauthorizedClient(t *testing.T) {
	raw := `bad response status code 401, message: unauthorized client detected, contact support for assistance at https://discord.gg/aYq5B4RW3, ` +
		`body: {"error":{"message":"unauthorized client detected, contact support for assistance at https://discord.gg/aYq5B4RW3"},` +
		`"message":"UNAUTHENTICATED","success":false,"type":"unauthorized_client_error"}`

	got := LocalizeChannelTestErrorText(raw)

	if !strings.Contains(got, "拒绝了调用方客户端") {
		t.Fatalf("期望识别为客户端被上游限制，实际：%s", got)
	}
	// 关键：不得被更靠后的 401 规则误判为“密钥无效”
	if strings.Contains(got, "密钥无效") {
		t.Fatalf("不应误判为密钥无效，实际：%s", got)
	}
	if !strings.Contains(got, "中转") {
		t.Fatalf("期望说明中转/网关不被接受，实际：%s", got)
	}
	if !strings.Contains(got, raw) {
		t.Fatalf("期望保留原始错误，实际：%s", got)
	}
}

func TestLocalizeChannelTestErrorTextKeepsUnknownAndEmpty(t *testing.T) {
	if got := LocalizeChannelTestErrorText(""); got != "" {
		t.Fatalf("空错误应返回空字符串，实际：%q", got)
	}
	// 已是中文的自有报错不应被再次包装
	raw := "暂不支持测试 Midjourney 类型的渠道"
	if got := LocalizeChannelTestErrorText(raw); got != raw {
		t.Fatalf("未知/已中文化的错误应原样返回，实际：%s", got)
	}
}

func TestDetectNonJSONUpstreamResponseAliyunWAF(t *testing.T) {
	// 关键点：状态码是 200，只有 Content-Type 与响应体能暴露问题
	resp := newTestResponse(http.StatusOK, "text/html; charset=utf-8", aliyunWAFPageSample)

	err := detectNonJSONUpstreamResponse(resp)
	if err == nil {
		t.Fatal("期望识别出网页响应并返回错误，实际为 nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "网页内容") {
		t.Fatalf("期望提示返回的是网页内容，实际：%s", msg)
	}
	if !strings.Contains(msg, "阿里云 WAF") {
		t.Fatalf("期望识别出阿里云 WAF 验证页，实际：%s", msg)
	}
	if !strings.Contains(msg, "响应片段") {
		t.Fatalf("期望附带响应片段以便排查，实际：%s", msg)
	}
}

func TestDetectNonJSONUpstreamResponseAllowsValidTypes(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
	}{
		{"JSON", "application/json", `{"id":"x","choices":[]}`},
		{"SSE 流式", "text/event-stream", "data: {\"id\":\"x\"}\n\ndata: [DONE]\n"},
		{"无 Content-Type", "", `{"id":"x"}`},
		{"纯文本", "text/plain; charset=utf-8", `{"id":"x"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := newTestResponse(http.StatusOK, tc.contentType, tc.body)
			if err := detectNonJSONUpstreamResponse(resp); err != nil {
				t.Fatalf("%s 响应不应被判定为网页内容，实际报错：%v", tc.name, err)
			}
		})
	}
}

func TestDetectNonJSONUpstreamResponseNilSafe(t *testing.T) {
	if err := detectNonJSONUpstreamResponse(nil); err != nil {
		t.Fatalf("nil 响应应安全返回 nil，实际：%v", err)
	}
}

func TestDescribeKnownInterceptPage(t *testing.T) {
	cases := map[string]string{
		aliyunWAFPageSample:                      "阿里云 WAF",
		`<html><head><title>404 Not Found`:       "404",
		`<html><body>502 Bad Gateway</body>`:     "网关错误页",
		`<html><body><h1>nginx</h1></body>`:      "反向代理",
		`<div>Please complete the captcha</div>`: "人机验证",
	}
	for page, want := range cases {
		if got := describeKnownInterceptPage(page); !strings.Contains(got, want) {
			t.Fatalf("页面片段 %q 期望识别为 %q，实际：%s", page[:min(40, len(page))], want, got)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
