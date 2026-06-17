package service

import (
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx           *gin.Context
	TokenGroup    string
	ModelName     string // 请求模型名称（未映射前）
	ActualRetry   *int   // 实际模型重试次数（独立）
	MappedRetry   *int   // 映射模型重试次数（独立）
	IsMappedPhase bool   // 当前是否在映射模型阶段
	resetNextTry  bool
}

func (p *RetryParam) GetActualRetry() int {
	if p.ActualRetry == nil {
		return 0
	}
	return *p.ActualRetry
}

func (p *RetryParam) GetMappedRetry() int {
	if p.MappedRetry == nil {
		return 0
	}
	return *p.MappedRetry
}

func (p *RetryParam) SetActualRetry(retry int) {
	p.ActualRetry = &retry
}

func (p *RetryParam) SetMappedRetry(retry int) {
	p.MappedRetry = &retry
}

func (p *RetryParam) IncreaseActualRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.ActualRetry == nil {
		p.ActualRetry = new(int)
	}
	*p.ActualRetry++
}

func (p *RetryParam) IncreaseMappedRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.MappedRetry == nil {
		p.MappedRetry = new(int)
	}
	*p.MappedRetry++
}

func (p *RetryParam) SwitchToMappedPhase() {
	p.IsMappedPhase = true
	p.MappedRetry = new(int)
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
//
//   - priorityRetry = Retry - startRetryIndex, represents the priority level within current group.
//     priorityRetry = Retry - startRetryIndex，表示当前分组内的优先级级别。
//
//   - When GetRandomSatisfiedChannel returns nil (priorities exhausted), moves to next group.
//     当 GetRandomSatisfiedChannel 返回 nil（优先级用完）时，切换到下一个分组。
//
// Example flow (2 groups, each with 2 priorities, RetryTimes=3):
// 示例流程（2个分组，每个有2个优先级，RetryTimes=3）：
//
//	Retry=0: GroupA, priority0 (startRetryIndex=0, priorityRetry=0)
//	         分组A, 优先级0
//
//	Retry=1: GroupA, priority1 (startRetryIndex=0, priorityRetry=1)
//	         分组A, 优先级1
//
//	Retry=2: GroupA exhausted → GroupB, priority0 (startRetryIndex=2, priorityRetry=0)
//	         分组A用完 → 分组B, 优先级0
//
//	Retry=3: GroupB, priority1 (startRetryIndex=2, priorityRetry=1)
//	         分组B, 优先级1
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	retry := param.GetActualRetry()
	if param.IsMappedPhase {
		retry = param.GetMappedRetry()
	}
	channel, err := model.GetRandomSatisfiedChannel("", param.ModelName, retry, param.IsMappedPhase)
	return channel, "", err
}
