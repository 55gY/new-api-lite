package service

import "github.com/55gY/new-api-lite/setting/operation_setting"

// ToolCallUsage captures all tool call counts from a single request.
type ToolCallUsage struct {
	ModelName              string
	WebSearchCalls         int
	WebSearchToolName      string // "web_search_preview", "web_search", etc.
	FileSearchCalls        int
	ImageGenerationCall    bool
	ImageGenerationQuality string
	ImageGenerationSize    string
}

// ToolCallItem represents a single tool usage line.
type ToolCallItem struct {
	Name       string  `json:"name"`
	CallCount  int     `json:"call_count"`
	PricePer1K float64 `json:"price_per_1k"`
	TotalPrice float64 `json:"total_price"`
	Quota      int     `json:"quota"`
}

// ToolCallResult holds the aggregated tool call usage for a request.
type ToolCallResult struct {
	TotalQuota int            `json:"total_quota"`
	Items      []ToolCallItem `json:"items,omitempty"`
}

// ComputeToolCallQuota keeps tool call usage metadata while local billing is disabled.
func ComputeToolCallQuota(usage ToolCallUsage, groupRatio float64) ToolCallResult {
	var items []ToolCallItem

	addItem := func(toolName string, count int) {
		if count <= 0 {
			return
		}
		pricePer1K := operation_setting.GetToolPriceForModel(toolName, usage.ModelName)
		items = append(items, ToolCallItem{
			Name:       toolName,
			CallCount:  count,
			PricePer1K: pricePer1K,
			TotalPrice: 0,
			Quota:      0,
		})
	}

	if usage.WebSearchCalls > 0 && usage.WebSearchToolName != "" {
		addItem(usage.WebSearchToolName, usage.WebSearchCalls)
	}

	if usage.FileSearchCalls > 0 {
		addItem("file_search", usage.FileSearchCalls)
	}

	if usage.ImageGenerationCall {
		price := operation_setting.GetGPTImage1PriceOnceCall(usage.ImageGenerationQuality, usage.ImageGenerationSize)
		items = append(items, ToolCallItem{
			Name:       "image_generation",
			CallCount:  1,
			PricePer1K: price,
			TotalPrice: 0,
			Quota:      0,
		})
	}

	return ToolCallResult{
		TotalQuota: 0,
		Items:      items,
	}
}
