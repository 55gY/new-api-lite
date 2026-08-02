package controller

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

// 渠道测试错误信息中文化。
//
// 背景：渠道测试失败时，底层返回的多是 Go 运行时/上游原始英文错误
// （例如 `invalid character '<' looking for beginning of value`），
// 对使用者不友好且难以定位。这里统一把测试相关的英文报错翻译成中文说明，
// 并给出常见原因提示，同时在末尾保留原始错误，便于进一步排查。

// channelTestErrorExplanation 一条错误匹配规则。
// keywords 全部命中（不区分大小写）才算匹配，便于表达“既包含 A 又包含 B”的组合条件。
type channelTestErrorExplanation struct {
	keywords []string
	message  string
}

// channelTestErrorExplanations 按“由具体到宽泛”的顺序匹配，命中第一条即返回。
var channelTestErrorExplanations = []channelTestErrorExplanation{
	// —— 响应体不是 JSON ——
	{
		keywords: []string{"invalid character '<'"},
		message:  "上游返回的是 HTML 页面而不是 JSON 数据，说明请求没有真正到达接口。常见原因：BaseURL 配置错误（多写或少写 /v1、结尾多了斜杠）、上游启用了人机验证/WAF 拦截（返回验证页）、被反向代理挡下返回了 404/502 等错误页。可用 curl 直接请求该地址，查看返回的是否为网页内容。",
	},
	{
		keywords: []string{"looking for beginning of value"},
		message:  "上游返回的内容不是合法 JSON，无法解析。请确认 BaseURL 指向的是 API 接口而非网页，且上游未返回验证页或错误页。",
	},
	{
		keywords: []string{"unexpected end of json input"},
		message:  "上游返回的 JSON 不完整（可能被截断或连接提前中断）。请检查网络稳定性、反向代理超时设置，或上游是否异常断开。",
	},
	{
		keywords: []string{"cannot unmarshal"},
		message:  "上游返回的 JSON 结构与预期不一致（字段类型不符）。可能是该上游并非标准 OpenAI 兼容格式，或端点类型选择有误。",
	},

	// —— 客户端/调用方被上游限制（必须排在 401/403 之前，否则会被误判为密钥问题）——
	{
		keywords: []string{"unauthorized_client"},
		message:  "上游拒绝了调用方客户端（unauthorized client）：该服务只允许其指定的客户端直接调用，不接受经由中转/网关（本项目即属于此类）转发的请求。这不是密钥错误，也不是本网关配置错误。请改用上游文档中支持的客户端直接调用，或联系上游确认是否允许网关中转、能否为你的用途开通权限。",
	},
	{
		keywords: []string{"unauthorized client"},
		message:  "上游拒绝了调用方客户端（unauthorized client）：该服务只允许其指定的客户端直接调用，不接受经由中转/网关转发的请求。这不是密钥错误。请改用上游支持的客户端，或联系上游确认中转授权。",
	},
	{
		keywords: []string{"client not allowed"},
		message:  "上游拒绝了调用方客户端：该服务限制了允许调用的客户端类型，当前以网关中转方式调用被拒绝。请联系上游确认授权方式。",
	},

	// —— HTTP 状态码类 ——
	{
		keywords: []string{"status code 401"},
		message:  "上游返回 401 未授权：密钥无效、已过期或格式不正确。请检查渠道密钥是否填写正确、是否包含多余空格。",
	},
	{
		keywords: []string{"status code 403"},
		message:  "上游返回 403 拒绝访问：密钥无该模型/接口权限，或来源 IP 被上游限制。",
	},
	{
		keywords: []string{"status code 404"},
		message:  "上游返回 404 接口不存在：BaseURL 或端点类型配置有误（例如应填到 /v1 而未填、或重复了 /v1）。",
	},
	{
		keywords: []string{"status code 429"},
		message:  "上游返回 429 请求过于频繁：已触发上游限流，请降低测试频率或稍后重试。",
	},
	{
		keywords: []string{"status code 500"},
		message:  "上游返回 500 内部错误：上游服务自身异常，请稍后重试或联系上游。",
	},
	{
		keywords: []string{"status code 502"},
		message:  "上游返回 502 网关错误：上游或其反向代理不可用。",
	},
	{
		keywords: []string{"status code 503"},
		message:  "上游返回 503 服务不可用：上游暂时过载或维护中。",
	},
	{
		keywords: []string{"status code 504"},
		message:  "上游返回 504 网关超时：上游处理超时。",
	},

	// —— 网络/传输类 ——
	{
		keywords: []string{"no such host"},
		message:  "域名解析失败：BaseURL 中的域名无法解析，请检查域名拼写与服务器 DNS 配置。",
	},
	{
		keywords: []string{"connection refused"},
		message:  "连接被拒绝：目标地址或端口未开放，请检查 BaseURL 的地址与端口、以及上游是否在运行。",
	},
	{
		keywords: []string{"connection reset by peer"},
		message:  "连接被对端重置：可能被上游防火墙/WAF 主动断开，或网络链路不稳定。",
	},
	{
		keywords: []string{"context deadline exceeded"},
		message:  "请求超时：上游在超时时间内未响应。可适当调大超时设置，或检查网络与上游负载。",
	},
	{
		keywords: []string{"client.timeout"},
		message:  "请求超时：等待上游响应超过客户端超时时间。",
	},
	{
		keywords: []string{"i/o timeout"},
		message:  "网络读写超时：与上游之间的连接超时，请检查网络连通性与代理设置。",
	},
	{
		keywords: []string{"tls: handshake"},
		message:  "TLS 握手失败：可能是协议版本/加密套件不兼容，或中间设备拦截了 HTTPS 连接。",
	},
	{
		keywords: []string{"x509"},
		message:  "TLS 证书校验失败：上游证书不受信任或已过期（自签证书需在服务器信任该证书）。",
	},
	{
		keywords: []string{"certificate"},
		message:  "TLS 证书异常：请检查上游证书是否有效、域名是否匹配。",
	},
	{
		keywords: []string{"proxyconnect"},
		message:  "代理连接失败：渠道配置的代理不可用，请检查代理地址与凭据。",
	},
	{
		keywords: []string{"unexpected eof"},
		message:  "连接被意外中断：上游在返回完整响应前关闭了连接。",
	},

	// —— 模型/参数类 ——
	{
		keywords: []string{"model_not_found"},
		message:  "上游报告模型不存在：请确认该模型名在上游确实可用，以及是否需要通过模型重定向映射为上游的真实模型名。",
	},
	{
		keywords: []string{"model", "not found"},
		message:  "上游报告模型不存在或不可用：请确认模型名拼写，以及该密钥是否有权访问此模型。",
	},
	{
		keywords: []string{"does not exist"},
		message:  "上游报告所请求的对象不存在（通常是模型名不正确）。请核对模型名或配置模型重定向。",
	},
	{
		keywords: []string{"invalid_request_error"},
		message:  "上游报告请求参数不合法：请检查模型名、端点类型与参数覆盖设置。",
	},

	// —— 测试流程自身 ——
	{
		keywords: []string{"usage is nil"},
		message:  "上游未返回用量（usage）信息：非流式响应缺少 usage 字段，可能该上游不兼容标准返回格式。",
	},
	{
		keywords: []string{"stream response body is empty"},
		message:  "流式响应内容为空：上游未返回任何流式数据。",
	},
	{
		keywords: []string{"does not contain a valid stream event"},
		message:  "流式响应中没有有效的流事件：返回内容不符合 SSE 流式格式，请确认上游是否支持流式输出。",
	},
	{
		keywords: []string{"upstream error"},
		message:  "上游返回了错误信息。",
	},
}

// LocalizeChannelTestErrorText 把渠道测试的原始错误文本转换为中文说明。
// 未匹配到已知模式时原样返回，避免丢失信息。
func LocalizeChannelTestErrorText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	for _, item := range channelTestErrorExplanations {
		matched := true
		for _, kw := range item.keywords {
			if !strings.Contains(lower, kw) {
				matched = false
				break
			}
		}
		if matched {
			return fmt.Sprintf("%s（原始错误：%s）", item.message, trimmed)
		}
	}
	return trimmed
}

// localizeChannelTestError 是 LocalizeChannelTestErrorText 的 error 版本封装。
func localizeChannelTestError(err error) string {
	if err == nil {
		return ""
	}
	return LocalizeChannelTestErrorText(err.Error())
}

const (
	// nonJSONUpstreamSniffBytes 用于识别拦截页特征时读取的字节数（越多越容易命中特征）。
	nonJSONUpstreamSniffBytes = 1024
	// nonJSONUpstreamPreviewRunes 最终展示给使用者的响应片段长度（避免提示过长）。
	nonJSONUpstreamPreviewRunes = 160
)

// truncateRunes 按字符数截断并在超长时追加省略号，避免截断多字节字符。
func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}

// detectNonJSONUpstreamResponse 在解析响应前先判断上游是否返回了网页等非 API 内容。
//
// 典型场景：上游被 WAF/人机验证拦截，或 BaseURL 指向了网页，此时会返回
// Content-Type 为 text/html 的页面，且状态码可能仍是 200，
// 后续按 JSON 解析就会得到 `invalid character '<'` 这类难以理解的报错。
// 这里提前拦截并给出明确的中文说明与响应片段。
func detectNonJSONUpstreamResponse(resp *http.Response) error {
	if resp == nil {
		return nil
	}
	rawContentType := resp.Header.Get("Content-Type")
	if rawContentType == "" {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(rawContentType)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(strings.Split(rawContentType, ";")[0]))
	}
	mediaType = strings.ToLower(mediaType)

	// 仅对明确的网页类响应做拦截，避免误伤 JSON / SSE / 二进制等正常返回。
	if mediaType != "text/html" && mediaType != "application/xhtml+xml" {
		return nil
	}

	sniffed := ""
	if resp.Body != nil {
		if data, readErr := io.ReadAll(io.LimitReader(resp.Body, nonJSONUpstreamSniffBytes)); readErr == nil {
			sniffed = strings.TrimSpace(string(data))
		}
		// 该响应必定无法作为 API 结果使用，直接关闭，避免连接泄漏。
		_ = resp.Body.Close()
	}
	// 特征识别用较长的内容，展示给使用者的片段做截断
	preview := truncateRunes(strings.Join(strings.Fields(sniffed), " "), nonJSONUpstreamPreviewRunes)

	message := fmt.Sprintf(
		"上游返回的是网页内容（Content-Type: %s，HTTP 状态码 %d）而不是 JSON 接口数据，请求没有真正到达 API。"+
			"常见原因：BaseURL 配置错误（多写或少写 /v1）、上游启用了人机验证或 WAF 拦截（返回验证页）、"+
			"被反向代理挡下返回了错误页、或服务器出口 IP 被上游限制。",
		rawContentType, resp.StatusCode,
	)
	if sniffed != "" {
		if hint := describeKnownInterceptPage(sniffed); hint != "" {
			message += hint
		}
		message += fmt.Sprintf("响应片段：%s", preview)
	}
	return fmt.Errorf("%s", message)
}

// describeKnownInterceptPage 识别常见的拦截页特征，给出更精准的中文提示。
func describeKnownInterceptPage(preview string) string {
	lower := strings.ToLower(preview)
	switch {
	case strings.Contains(lower, "aliyun_waf") || strings.Contains(lower, "aliyuncaptcha"):
		return "检测到阿里云 WAF 人机验证页：上游已对本服务器的请求要求滑块验证，程序无法自动通过。" +
			"请联系上游将服务器出口 IP 加入白名单或关闭该验证。"
	case strings.Contains(lower, "cf-browser-verification") || strings.Contains(lower, "cloudflare"):
		return "检测到 Cloudflare 人机验证/拦截页：上游要求浏览器验证，程序无法自动通过。" +
			"请联系上游放行本服务器出口 IP。"
	case strings.Contains(lower, "captcha") || strings.Contains(lower, "verification"):
		return "检测到人机验证页：上游要求完成验证，程序无法自动通过。"
	case strings.Contains(lower, "<title>404") || strings.Contains(lower, "not found"):
		return "检测到 404 页面：接口路径不存在，请检查 BaseURL 与端点类型配置。"
	case strings.Contains(lower, "502 bad gateway") || strings.Contains(lower, "504 gateway"):
		return "检测到网关错误页：上游或其反向代理不可用。"
	case strings.Contains(lower, "nginx") || strings.Contains(lower, "openresty"):
		return "检测到反向代理默认页：请求被反向代理处理而未转发到 API 服务。"
	}
	return ""
}
