package dto_test

import (
	"strings"
	"testing"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/dto"

	"github.com/samber/lo"
)

// 规则5：上行 relay 请求 DTO 的可选标量必须用指针 + omitempty，
// 保证客户端显式传入的零值/false 能透传上游，而缺省时被省略。

func TestStreamOptionsRule5PreservesExplicitFalse(t *testing.T) {
	// 缺省（nil）=> 省略
	b, err := common.Marshal(dto.StreamOptions{})
	if err != nil {
		t.Fatalf("marshal nil: %v", err)
	}
	if strings.Contains(string(b), "include_usage") {
		t.Fatalf("expected include_usage omitted when nil, got %s", b)
	}
	// 显式 false => 透传
	b, err = common.Marshal(dto.StreamOptions{IncludeUsage: lo.ToPtr(false)})
	if err != nil {
		t.Fatalf("marshal false: %v", err)
	}
	if !strings.Contains(string(b), `"include_usage":false`) {
		t.Fatalf("expected include_usage:false preserved, got %s", b)
	}
	// 显式 true => 透传
	b, err = common.Marshal(dto.StreamOptions{IncludeUsage: lo.ToPtr(true)})
	if err != nil {
		t.Fatalf("marshal true: %v", err)
	}
	if !strings.Contains(string(b), `"include_usage":true`) {
		t.Fatalf("expected include_usage:true preserved, got %s", b)
	}
}

func TestClaudeToolChoiceRule5PreservesExplicitFalse(t *testing.T) {
	// 缺省（nil）=> 省略
	b, err := common.Marshal(dto.ClaudeToolChoice{Type: "auto"})
	if err != nil {
		t.Fatalf("marshal nil: %v", err)
	}
	if strings.Contains(string(b), "disable_parallel_tool_use") {
		t.Fatalf("expected disable_parallel_tool_use omitted when nil, got %s", b)
	}
	// 显式 false => 透传
	b, err = common.Marshal(dto.ClaudeToolChoice{Type: "auto", DisableParallelToolUse: lo.ToPtr(false)})
	if err != nil {
		t.Fatalf("marshal false: %v", err)
	}
	if !strings.Contains(string(b), `"disable_parallel_tool_use":false`) {
		t.Fatalf("expected disable_parallel_tool_use:false preserved, got %s", b)
	}
}
