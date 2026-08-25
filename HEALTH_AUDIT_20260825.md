# new-api-lite 全库健康审计报告

**审计日期：** 2026-08-25  
**审计范围：** Go 后端、经典前端 `web/classic`、依赖清单、并发资源生命周期、构建与测试链路。  
**执行原则：** 先建立隔离基线，再实施不改变 API 契约、认证流程、数据库兼容性及精简版功能边界的高置信修复；不对动态引用、公开 API 或大规模既有格式债务执行批量删除。

## 结论摘要

本轮审计完成了前后端隔离构建、Go 常规测试、生产构建、`go vet`、全库竞态检测与 staticcheck。最关键的结果是：此前全量竞态检测中暴露的 **生产级流扫描器 channel close/send 竞态** 与 **测试级 Gin 全局状态竞态** 均已消除；最终隔离副本的 `go test -race ./...` 已通过。

同时，本轮删除了已确认不可达的支付回跳辅助文件、默认供应商定价映射文件、旧 TOTP 环境变量 helper 与旧 Claude 用量判断残留；修复了 Gemini embedding handler 的无效文本收集、三个 nil 日志 context、Perplexity 模型倍率整数除法恒为零等明确问题。前端补上了 Playground 卸载时的 SSE 连接关闭、Mermaid blob URL 回收，并完成本次触及文件的 ESLint 和 Prettier 清理。

| 指标 | 审计基线 | 本轮结果 |
| --- | ---: | ---: |
| Go 常规测试 | 通过 | 通过 |
| Go 生产构建（`-trimpath`） | 通过 | 通过 |
| Go vet | 通过 | 通过 |
| Go 全库竞态检测 | 失败 | **通过** |
| 前端 Bun 生产构建 | 通过 | 通过 |
| 本次触及前端文件 ESLint / Prettier | 存在 6 项 ESLint 自动修复项 | **通过** |
| staticcheck 诊断 | 约 159 项 | 134 项 |

## 已实施修复

### 流式响应并发与资源生命周期

`relay/helper/stream_scanner.go` 原先以多个 worker 向 `stopChan` 发送通知，并在主 defer 中可能于 worker 仍运行时关闭该 channel。`SafeSendBool` 通过 recover 掩盖 send-on-closed-channel panic，不能解决竞态；竞态检测已证实 worker 发送与主协程关闭存在并发访问。

现已改为以 `context.WithCancel` 进行单向停止广播，并由 scanner 独占关闭 `dataChan` 与 `scannerDone`。发生超时、客户端断连、处理器主动停止或 ping 错误时，会取消 context 并关闭上游 response body，以解除 `Scanner.Scan()` 的阻塞；所有受跟踪 worker 均在返回前 `Wait` 完成。ping 写入不再创建未受 WaitGroup 跟踪的嵌套任务。`StreamStatus` 的结束原因和错误字段也纳入其现有 mutex 保护，避免主协程记录日志与 worker 设置结束状态发生读写竞争。

| 修复点 | 处理方式 | 验证 |
| --- | --- | --- |
| 多发送方关闭 `stopChan` | 取消 `stopChan` 所有权模型，改用 context cancel | `go test -race ./relay/helper` 通过；全库 race 通过。 |
| 上游 scanner 阻塞 | 取消时关闭 `resp.Body`，worker 退出后再返回 | timeout、handler stop、DONE、EOF、慢上游 ping 等既有测试通过。 |
| 状态/计数竞争 | `StreamStatus` 读写加锁，并在汇总前等待 worker | 原 `StreamStatus_Timeout` race 不再出现。 |
| ping 子任务游离 | 在受跟踪 ping worker 内直接写入 | 全库 race 通过。 |

### 测试级全局状态隔离

Gin 的 mode 为进程全局状态。`relay/channel/api_request_test.go`、Gemini 用量测试、MiniMax adapter 测试和 AWS 测试曾在 `t.Parallel()` 后调用 `gin.SetMode(gin.TestMode)`；`stream_scanner_test.go` 也会修改全局 timeout 与 operation settings。现已取消这些测试的并行执行，以稳定、确定性地验证全局可变配置。

### 未使用代码和明确静态问题

已删除完整不可达的 `controller/return_path.go`（支付回跳残留）及 `model/pricing_default.go`（旧默认供应商/定价映射）；两者均由 staticcheck 判定为仅含未使用私有声明。还删除了未调用的 TOTP 环境变量 helper 与旧 Claude 用量兼容判定。

本轮修正了若干无效初始化/赋值：渠道查询结果、渠道创建 key 切片、模型 owner map、性能接口磁盘状态、Gemini embedding 的未使用文本收集和密码重置 JSON 解码错误处理。三个 logger 的 nil context 改为 `context.Background()`。Perplexity `llama-3-sonar-large-32k-*` 模型倍率中的 `1 / 1000` 改为 `1.0 / 1000`，从而不再因整数除法获得零倍率。

### 前端资源回收与代码质量

Playground 状态 Hook 在组件卸载时会关闭 `sseSourceRef.current` 并置空，避免导航后仍保留活动 SSE 连接。Mermaid SVG 新窗口预览创建的 blob URL 会在给加载留出时间后释放。六个此前 ESLint 报错的文件已补齐所需版权头或移除多余空行；本轮触及文件全部通过定向 ESLint 与 Prettier。

## 验证记录

所有最终验证均在隔离副本 `/tmp/new-api-lite-final-health` 中完成，避免 Bun 安装和前端构建写入主工作区锁文件或依赖目录。

| 命令 | 结果 | 说明 |
| --- | --- | --- |
| `bun install` | 通过 | 在隔离副本执行。 |
| `bun run build` | 通过 | 仅保留既有第三方大 chunk 警告。 |
| 定向 `prettier --check` | 通过 | 覆盖本轮 8 个触及前端文件。 |
| 定向 `eslint` | 通过 | 覆盖本轮 8 个触及前端文件。 |
| `go test ./...` | 通过 | 全部包。 |
| `go build -trimpath ./...` | 通过 | 前端资源已先构建以满足 embed pattern。 |
| `go vet ./...` | 通过 | 全部包。 |
| `CGO_ENABLED=1 go test -race ./...` | **通过** | 先前报告的所有 race 均未复现。 |
| `go mod tidy -diff` | 仅依赖分类差异 | 仅建议将 `golang.org/x/sync` 由直接依赖标为 indirect；未自动改动依赖声明。 |

## 未自动处理的残余项

staticcheck 仍报告 **134** 项诊断。这不是构建或测试失败；大部分属于低风险风格项，或需要兼容性/运行时引用确认后才可删除。为避免把动态配置、公开接口或历史数据库迁移逻辑误删，本轮没有用批量脚本清空这些候选。

| 残余类别 | 数量 | 处理建议 |
| --- | ---: | --- |
| `S1023` 冗余 return | 37 | 可在后续单独的机械格式化提交中处理。 |
| `U1000` 私有未使用声明 | 42 | 包含旧 quota/cache、性能 Redis、历史 adapter helper、TTS/音频残留等；应按功能域和导入影响分批删除。 |
| `SA4006` 无效赋值 | 9 | 需逐条核实失败路径是否意图保留 error，避免改变 adapter 协议行为。 |
| `SA1019` deprecated 路由 | 2 | 仍为公开日志 API；须先评估外部客户端兼容性，不能仅凭前端未引用删除。 |
| 其他 S/SA/ST 风格项 | 44 | 包括错误字符串大小写、nil slice 检查、可简化循环与类型断言；可在不改语义的专项整理中处理。 |

i18n 静态比对产生了 2,052 个未被静态 `t(...)` 调用覆盖的键，但资源含动态键、插值键和历史兼容键，不能安全批量删除。前端可达性图对常量聚合目录曾有误报，人工复核后确认多个文件仍被直接导入。前端依赖候选也包含构建配置、间接依赖或 peer dependency，未执行未经验证的卸载。

## 后续建议

后续如继续降低 staticcheck 数量，应以每个功能域一个可回滚提交的方式处理。优先顺序为：首先复核 9 个 `SA4006` adapter/转换路径，其次处理与项目精简边界冲突且在包内完全不可达的 TTS、旧 quota/cache 和 pricing helper，最后再统一处理 `S1023` 等纯风格项目。删除前应始终运行受影响包测试、全量构建和竞态检测。

真实生产环境仍建议补充长时间慢客户端、上游半关闭、网络写阻塞和浏览器导航中的 SSE 压力验证。静态检查和单元测试已覆盖资源所有权与数据竞争，但无法替代真实网络和用户会话负载。
