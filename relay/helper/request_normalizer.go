package helper

import (
	"github.com/55gY/new-api-lite/dto"
	"github.com/55gY/new-api-lite/relay/common"
)

// NormalizeRequestForModelMapping 在模型映射后对请求参数进行归一化处理
// 解决客户端按"虚拟模型"能力构造请求,但上游真实模型能力不同导致的参数校验失败问题
func NormalizeRequestForModelMapping(info *common.RelayInfo, request dto.Request) {
	if !info.IsModelMapped {
		return
	}

	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		normalizeOpenAIRequest(req)
	case *dto.ClaudeRequest:
		normalizeClaudeRequest(req)
	}
}

// normalizeOpenAIRequest 归一化 OpenAI 格式请求
func normalizeOpenAIRequest(req *dto.GeneralOpenAIRequest) {
	// 1. Clamp MaxTokens 到安全上限
	// 大多数模型支持的最大 max_tokens 不超过 65536
	// DeepSeek-V4-Flash 上限是 65536,阿里云 DashScope 上限是 16384
	// 使用 65536 作为通用上限,避免上游拒绝请求
	const maxTokensLimit = 65536

	if req.MaxTokens != nil && *req.MaxTokens > maxTokensLimit {
		clamped := uint(maxTokensLimit)
		req.MaxTokens = &clamped
	}

	if req.MaxCompletionTokens != nil && *req.MaxCompletionTokens > maxTokensLimit {
		clamped := uint(maxTokensLimit)
		req.MaxCompletionTokens = &clamped
	}

	// 2. 清洗 tools schema 中的 null 值
	// 某些上游提供商(如 DeepSeek)对 function schema 的校验比 OpenAI 更严格
	// 如果客户端传了 "required": null 或某个数组字段是 null 而不是 [],会报错
	// Tools 字段是 []ToolCallRequest,需要遍历并检查每个 tool
	for i := range req.Tools {
		// ToolCallRequest 的 Type 字段是 string,检查是否为 function 类型
		if req.Tools[i].Type == "function" {
			cleanToolFunctionParameters(&req.Tools[i].Function)
		}
	}
}

// normalizeClaudeRequest 归一化 Claude 格式请求
func normalizeClaudeRequest(req *dto.ClaudeRequest) {
	// Claude 的 max_tokens 限制通常在 8192-4096 之间
	// 使用 8192 作为安全上限
	const maxTokensLimit = 8192

	if req.MaxTokens != nil && *req.MaxTokens > maxTokensLimit {
		clamped := uint(maxTokensLimit)
		req.MaxTokens = &clamped
	}

	// 清洗 tools schema
	// Tools 字段是 any 类型，需要先转换为 []any 再遍历
	if req.Tools != nil {
		if toolsSlice, ok := req.Tools.([]any); ok {
			for _, toolAny := range toolsSlice {
				if toolMap, ok := toolAny.(map[string]any); ok {
					// 检查是否为 Tool 对象 (有 name 和 input_schema 字段)
					if _, hasName := toolMap["name"]; hasName {
						// 尝试获取 InputSchema
						if inputSchema, ok := toolMap["input_schema"]; ok && inputSchema != nil {
							if schemaMap, ok := inputSchema.(map[string]any); ok {
								cleanNullValues(schemaMap)
							}
						}
					}
				}
			}
		}
	}
}

// cleanToolFunctionParameters 清洗 OpenAI 格式的 tool function parameters
// 递归删除值为 null 的字段,把 null 数组规范为空数组
func cleanToolFunctionParameters(fn *dto.FunctionRequest) {
	if fn.Parameters == nil {
		return
	}

	// Parameters 是 any 类型,需要处理为 map[string]any
	params, ok := fn.Parameters.(map[string]any)
	if !ok {
		return
	}

	cleanNullValues(params)
}

// cleanClaudeToolParameters 清洗 Claude 格式的 tool parameters
func cleanClaudeToolParameters(tool *dto.Tool) {
	if tool.InputSchema == nil {
		return
	}

	// InputSchema 已经是 map[string]interface{} 类型，直接使用
	schema := tool.InputSchema

	cleanNullValues(schema)
}

// cleanNullValues 递归清理 map 中的 null 值
// - 删除值为 null 的字段
// - 把 null 数组规范为空数组
// - 把 null 对象规范为空对象
func cleanNullValues(m map[string]any) {
	for key, value := range m {
		if value == nil {
			// 删除 null 值
			delete(m, key)
			continue
		}

		switch v := value.(type) {
		case map[string]any:
			// 递归处理嵌套对象
			cleanNullValues(v)
		case []any:
			// 处理数组,递归清理数组中的对象
			for i, item := range v {
				if item == nil {
					// 删除数组中的 null 元素
					v = append(v[:i], v[i+1:]...)
					i-- // 调整索引
				} else if nestedMap, ok := item.(map[string]any); ok {
					cleanNullValues(nestedMap)
				}
			}
			m[key] = v
		}
	}
}
