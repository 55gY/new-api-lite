# new-api 精简任务计划

## 执行规则

- 本文件是本次精简工作的唯一执行清单。
- 每完成一个具体步骤，必须把对应 checkbox 从 `[ ]` 更新为 `[x]`。
- 每完成一个 Phase，必须重新读取本文件，并提示下一步 Phase 的计划内容。
- 执行时跳过 `.bak` 目录及其内容。
- 当前项目直接精简，不新增 `lite` build tag，不保留 MySQL/PostgreSQL/音视频任务的编译期兼容分支。
- 删除功能前先断开路由、入口和引用，再删除孤立文件，避免难以定位的大面积编译错误。
- 每个阶段都要做残留引用检查；最终必须完成前端构建、后端构建和 smoke test。

## Refactor Plan: new-api 单一精简版改造

### Current State

- 后端同时支持 SQLite、MySQL、PostgreSQL，编译期包含 MySQL/PostgreSQL 驱动和多数据库分支。
- 后端包含 OpenAI audio relay、音频时长解析和多种音频格式依赖。
- 后端包含视频、音乐、Suno、Kling、Jimeng、Midjourney、异步任务、任务轮询、任务计费和任务日志相关逻辑。
- 后端仍保留多处计费、额度、倍率、预填组逻辑；用户目标是只保留 tokens 消耗统计。
- 前端仍可能保留任务、视频、音乐、Midjourney、额度、倍率、预填组等入口。
- Tokenizer 同时服务 token 统计、上下文估算和计费链路；本次保留 token 统计能力，但切断计费扣费用途。

### Target State

- 编译期只保留纯 Go SQLite 数据库路径。
- 删除 MySQL/PostgreSQL 驱动、配置分支、迁移差异分支和测试矩阵。
- 删除音频 API、音频处理代码、音频格式解析依赖。
- 删除视频/音乐/Suno/Kling/Jimeng/Midjourney/异步任务 API、任务记录、任务日志、任务轮询、任务计费。
- 删除用户额度、token 额度、渠道倍率、模型倍率、分组倍率、扣费、返还、余额不足、预填组管理。
- 保留核心第三方 AI API 聚合能力：models、chat completions、responses、embeddings、必要的 image relay、用户认证、API token、渠道管理、请求日志、tokens 消耗统计。
- 保留 Tokenizer 的 token count/context estimation；不再将其结果用于扣费。
- 前端只保留精简后的核心管理与调用入口。

### Affected Files

| File/Pattern | Change Type | Dependencies |
| --- | --- | --- |
| `common/database.go` | modify | SQLite-only 全局数据库类型与状态 |
| `model/main.go` | modify | 删除 MySQL/PostgreSQL driver imports、DSN 分支、MySQL charset 检查 |
| `model/db_time.go` | modify | 固定 SQLite 时间表达式 |
| `model/channel.go` | modify | 删除 MySQL/PostgreSQL SQL 分支 |
| `model/ability.go` | modify | 删除多数据库分支 |
| `model/usedata_rankings.go` | modify | 删除 MySQL 分支 |
| `controller/setup.go` | modify | 数据库类型展示固定 SQLite |
| `controller/*_test.go` | modify | 删除 MySQL/PostgreSQL 测试矩阵 |
| `router/relay-router.go` | modify | 删除 audio、Suno/task relay routes |
| `router/video-router.go` | delete | 删除视频/Kling/Jimeng routes |
| `router/main.go` | modify | 移除 `SetVideoRouter` 调用 |
| `router/api-router.go` | modify | 删除 task、prefill、quota/rate/billing 管理入口 |
| `common/audio.go` | delete | 删除音频格式解析依赖 |
| `service/audio.go` | delete/modify | 删除 audio-only helper；若有非音频引用则改为局部保留 |
| `relay/audio_handler.go` | delete | 删除 OpenAI audio relay |
| `relay/channel/openai/audio.go` | delete | 删除 audio adaptor |
| `dto/audio.go` | delete | 删除 audio DTO |
| `types/relay_format.go` | modify | 删除 `RelayFormatOpenAIAudio` |
| `relay/helper/valid_request.go` | modify | 删除 audio request validation case |
| `relay/common/request_conversion.go` | modify | 删除 audio conversion case |
| `relay/common/relay_info.go` | modify | 删除 audio relay info generator/case |
| `relay/constant/relay_mode.go` | modify | 删除 audio route mode |
| `middleware/distributor.go` | modify | 删除 audio、Suno、task 特判 |
| `constant/task.go` | delete | 删除 task constants |
| `model/task.go` | delete | 删除 task model/query/update/migration |
| `dto/task.go` | delete | 删除 task DTO |
| `controller/task.go` | delete | 删除 task log/list API |
| `relay/relay_task.go` | delete | 删除 task relay submit/fetch |
| `service/task.go` | delete | 删除 task service |
| `service/task_polling.go` | delete | 删除 task polling |
| `service/task_billing.go` | delete | 删除 task billing/refund |
| `controller/task_video.go` | delete | 删除 video task update |
| `controller/video_proxy.go` | delete | 删除 video proxy |
| `controller/video_proxy_gemini.go` | delete | 删除 Gemini/Vertex video proxy helper |
| `controller/swag_video.go` | delete | 删除 video swagger annotations |
| `dto/video.go` | delete | 删除 video DTO |
| `dto/openai_video.go` | delete | 删除 OpenAI video DTO |
| `common/init.go` | modify | 删除 task-related env 初始化 |
| `common/endpoint_type.go` | modify | 删除 OpenAI video endpoint type |
| `constant/endpoint_type.go` | modify | 删除 OpenAI video endpoint type |
| `common/quota.go` | delete/modify | 删除额度格式化/计算，保留必要兼容 shim 时需 no-op |
| `service/quota.go` | delete/modify | 删除扣费/返还/余额不足；保留 usage-only 记录路径 |
| `service/token_counter.go` | modify | 保留 token 统计，删除 audio/计费耦合 |
| `controller/token.go` | modify | 保留 API token，删除 token 额度字段/校验 |
| `model/token.go` | modify | 保留认证 token 字段，删除 token quota 字段/逻辑 |
| `controller/user.go` | modify | 删除用户额度展示/调整/扣减 |
| `model/user.go` | modify | 删除用户 quota 业务逻辑；谨慎处理历史字段迁移 |
| `controller/log.go` | modify | 保留 tokens/request usage，删除扣费语义 |
| `model/log.go` | modify | 保留 usage log，删除 quota/billing 字段依赖 |
| `setting/model-ratio.go` | delete/modify | 删除模型倍率配置入口 |
| `setting/ratio_setting.go` | delete/modify | 删除倍率配置入口 |
| `controller/prefill_group.go` | delete | 删除预填组管理 API |
| `model/prefill_group.go` | delete | 删除预填组模型/迁移 |
| `web/classic/src/**` | modify/delete | 删除前端任务/视频/音乐/Midjourney/额度/倍率/预填组入口 |
| `go.mod`, `go.sum` | modify | `go mod tidy` 清理无用依赖 |

### Execution Plan

#### Phase 0: 建立基线与保护点

- [x] Step 0.1: 检查当前工作树状态，确认已有未提交改动，避免覆盖用户工作。备注：当前目录不是 Git 仓库，无法通过 `git status` 建立提交级保护点，后续按小批量修改并同步记录。
- [x] Step 0.2: 记录当前后端构建结果与 `new-api.exe` 体积，作为精简前基线。备注：当前 `new-api.exe` 约 58.6 MB，`web/classic/dist` 约 11.17 MB。
- [x] Step 0.3: 重新搜索并记录 `.bak` 以外的数据库、音频、视频、任务、计费、倍率、预填组引用清单。备注：数据库引用集中在 `common/database.go`、`model/main.go`、`model/db_time.go`、`model/channel.go`、`model/ability.go`、`model/usedata_rankings.go`、相关 controller tests；音频引用集中在 relay/router/middleware/dto/service/common；视频/任务/计费/预填引用范围已由搜索和子代理确认。
- [x] Step 0.4: 确认前端实际目录为 `web/classic`，重新定位非 `.bak` 前端入口。备注：入口包含 `web/classic/src/App.jsx`、`components/layout/SiderBar.jsx`、`components/layout/PageLayout.jsx`、`pages/Task`、`components/table/task-logs`、tokens/users 表单、`constants/common.constant.js`、`constants/channel.constants.js`、i18n 文案等。
- [x] Verify 0: 基线信息和影响范围已确认，无需修改业务代码。

#### Phase 1: SQLite-only 数据库精简

- [x] Step 1.1: 修改 `common/database.go`，只保留 SQLite 必要常量/状态，移除 MySQL/PostgreSQL 使用路径。
- [x] Step 1.2: 修改 `model/main.go`，删除 MySQL/PostgreSQL driver imports 与 `chooseDB` 分支，只允许 SQLite/local/空 DSN。备注：SQLite-only 下 `InitLogDB` 复用主库，不再读取独立日志库 DSN。
- [x] Step 1.3: 删除 `checkMySQLChineseSupport` 调用和 MySQL-only 检查逻辑。
- [x] Step 1.4: 固定 `initCol()` 为 SQLite 引号、bool 表达式。
- [x] Step 1.5: 修改 `model/db_time.go`、`model/channel.go`、`model/ability.go`、`model/usedata_rankings.go`，删除多数据库 SQL 分支。备注：同时清理 `FixAbility()` 的非 SQLite `TRUNCATE` 死分支。
- [x] Step 1.6: 修改 `controller/setup.go`，数据库展示/配置固定 SQLite。
- [x] Step 1.7: 修改相关测试，删除 MySQL/PostgreSQL dialect 矩阵和 driver imports。备注：补齐 `model/model_owner_test.go` SQLite 内存库初始化。
- [x] Verify 1: 搜索 `UsingMySQL|UsingPostgreSQL|driver/mysql|driver/postgres|go-sql-driver|jackc/pgx`，确认主代码无残留编译引用。备注：非 `.bak` 主代码无残留；`go.mod`/`go.sum` 仍保留待 Phase 8 `go mod tidy` 清理的依赖记录；`go test ./model ./controller` 与后端 `go build` 均通过。

#### Phase 2: 移除音频 API 与音频处理依赖

- [x] Step 2.1: 从 `router/relay-router.go` 删除 `/v1/audio/transcriptions`、`/v1/audio/translations`、`/v1/audio/speech`。
- [x] Step 2.2: 删除或断开 `relay/audio_handler.go`、`relay/channel/openai/audio.go`、`dto/audio.go`。备注：同时断开 OpenAI/Cloudflare/MiniMax/Volcengine 的 STT/TTS 分支和 helper 文件。
- [x] Step 2.3: 从 `types/relay_format.go` 删除 `RelayFormatOpenAIAudio`。
- [x] Step 2.4: 从 `relay/helper/valid_request.go`、`relay/common/request_conversion.go`、`relay/common/relay_info.go`、`relay/constant/relay_mode.go` 删除 audio case。
- [x] Step 2.5: 从 `middleware/distributor.go` 删除 audio multipart/audio route 特判。
- [x] Step 2.6: 删除 `common/audio.go` 和无用 `service/audio.go` 引用。备注：`common.GetAudioDuration` 调用链已删除；`service/audio.go` 当前不承载公开音频 API 入口，后续若 Phase 5 确认无 usage 统计依赖再清理。
- [x] Step 2.7: 删除 audio consume quota 路径，保留 text token usage 统计路径。备注：`PostAudioConsumeQuota` 已删除，compatible/responses 路径统一调用 `PostTextConsumeQuota`。
- [x] Verify 2: 搜索 `AudioHelper|OpenAIAudio|RelayFormatOpenAIAudio|GetAudioDuration|audio/transcriptions|audio/translations|audio/speech|PostAudioConsumeQuota`，确认无主代码残留。备注：搜索结果仅来自 `.bak`；前端构建通过（本机无 bun，使用本地 Vite CLI 等价执行 `vite build`）；前端 dist 更新后后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过，当前 `new-api.exe` 约 56.19 MB；聚焦测试 `go test -count=1 ./relay/channel ./relay/common ./relay/constant ./controller ./middleware` 通过。

#### Phase 3: 移除视频/音乐/Midjourney/异步任务 API

- [x] Step 3.1: 从 `router/main.go` 移除 `SetVideoRouter(router)`。
- [x] Step 3.2: 删除 `router/video-router.go`。
- [x] Step 3.3: 从 `router/relay-router.go` 删除 `/suno/submit/:action`、`/suno/fetch`、`/suno/fetch/:id`。
- [x] Step 3.4: 从 `controller/relay.go` 删除 `RelayTask`、`RelayTaskFetch` 等异步任务 relay 入口。备注：同时删除残留 `RelayMidjourney`。
- [x] Step 3.5: 删除 `relay/relay_task.go`。
- [x] Step 3.6: 删除 video proxy/update 相关文件：`controller/video_proxy.go`、`controller/video_proxy_gemini.go`、`controller/task_video.go`、`controller/swag_video.go`。
- [x] Step 3.7: 删除 video/task DTO：`dto/video.go`、`dto/openai_video.go`、`dto/task.go`。备注：同时删除 `dto/openai_response.go` 中未使用的 `OpenAIVideoResponse`。
- [x] Step 3.8: 删除 `constant/task.go` 中 Suno/Midjourney/task constants。
- [x] Step 3.9: 修改 endpoint type 常量，删除 OpenAI video endpoint。备注：同时清理 Midjourney/Suno/Kling/Jimeng endpoint 注释残留、task/MJ relay format、video/task middleware adapter、Midjourney 专用鉴权兼容分支。
- [x] Verify 3: 搜索 `Suno|Midjourney|Kling|Jimeng|RelayTask|SetVideoRouter|VideoProxy|OpenAIVideo|TaskPlatform`，确认主代码无残留编译引用；注意不要误删 `service/codex_credential_refresh_task.go`。备注：`.bak` 仍有历史备份命中；主代码中 `Jimeng` 仅保留必要图片 adaptor/渠道历史兼容常量，`Kling/Jimeng/Vidu` 供应商图标规则后续随前端/渠道展示精简处理；后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过。

#### Phase 4: 移除任务日志、任务记录、任务轮询、任务计费

- [x] Step 4.1: 从 `model/main.go` AutoMigrate/初始化列表移除 task model。备注：已随 Phase 3 编译清理提前完成，同时移除 Midjourney/Task 迁移。
- [x] Step 4.2: 删除 `model/task.go`。备注：已提前完成。
- [x] Step 4.3: 删除 `controller/task.go` 并从 `router/api-router.go` 删除 `/api/task` 管理入口。备注：已提前完成。
- [x] Step 4.4: 删除 `service/task.go`、`service/task_polling.go`、`service/task_billing.go`。备注：已提前完成。
- [x] Step 4.5: 从 `common/init.go` 删除 `UPDATE_TASK`、`TASK_QUERY_LIMIT`、`TASK_TIMEOUT_MINUTES`、`TASK_PRICE_PATCH` 等 task env 初始化。备注：同时删除 `constant/env.go` 中旧 task env 变量定义。
- [x] Step 4.6: 删除 main/init 中 task polling 或 task update background goroutine。备注：旧 task polling 已无入口；保留 `StartCodexCredentialAutoRefreshTask()` 与 `StartChannelUpstreamModelUpdateTask()`。
- [x] Step 4.7: 保留普通请求日志、错误日志、系统日志、tokens usage；仅删除 task log/list/record。备注：task list controller/model/service 已删除，普通 log 与 usage 路径未在本 Phase 改动。
- [x] Verify 4: 搜索 `TaskGet|TaskCount|GetByTaskId|TaskBulkUpdate|UPDATE_TASK|TASK_QUERY_LIMIT|TASK_TIMEOUT|TASK_PRICE_PATCH|/api/task`，确认主代码无残留。备注：命中仅来自 `.bak`、保留的 `channel_upstream_update` 后台任务，或 Ali 图片内部 `updateTask` 命名；后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过。

#### Phase 5: 计费/额度/倍率精简，只保留 tokens 消耗统计

- [x] Step 5.1: 梳理 quota/billing 调用链，区分“认证/限流/usage 统计”和“扣费/余额/倍率”。备注：已确认保留 API token 认证、用户/分组/模型路由、限流、request count、prompt/completion/total tokens usage；中和扣费、余额、返还、倍率价格路径。
- [x] Step 5.2: 修改 `service/quota.go`，删除或 no-op 扣费、返还、余额不足、倍率计价逻辑。备注：WSS/text relay 后处理保留 tokens usage log 和 request count，quota 固定为 0；预扣费/返还 no-op。
- [x] Step 5.3: 修改 `common/quota.go`，删除额度格式化/额度计算，仅保留必要兼容函数时返回 usage-only 结果。备注：`GetTrustQuota()` 固定返回 0；基础 quota 常量默认中性值。
- [x] Step 5.4: 修改 `model/user.go`、`controller/user.go`，删除用户额度调整、额度展示、余额不足语义。备注：用户 quota 增减/转移/邀请奖励 no-op，注册 quota 固定 0，自身额度字段兼容返回 0，管理员 add_quota 兼容 no-op。
- [x] Step 5.5: 修改 `model/token.go`、`controller/token.go`，保留 API token 认证，删除 token 额度、额度扣减和额度展示。备注：token quota 增减仅更新访问时间；token 创建/更新强制 `RemainQuota=0`、`UnlimitedQuota=true`，usage/status 额度兼容字段返回 0。
- [x] Step 5.6: 修改 `model/channel.go`、`controller/channel.go`，删除渠道倍率计费字段/展示/更新路径，保留渠道转发配置。备注：channel used quota 写入 no-op，clone 仅保留历史字段置 0 兼容。
- [x] Step 5.7: 删除 `setting/model-ratio.go`、`setting/ratio_setting.go` 或将其调用点彻底断开。备注：未整包删除以保留 compact model suffix、group routing/特殊可用分组兼容；倍率/价格读取统一返回 0/1/free，billing config 热路径返回 false。
- [x] Step 5.8: 修改 `controller/log.go`、`model/log.go`，保留 request/tokens usage，删除 quota/billing 语义字段依赖。备注：usage/error log 写入 quota 为 0，统计保留 token/rpm/tpm；历史 log type 常量保留兼容。
- [x] Step 5.9: 修改 `service/token_counter.go`，保留 token 统计与上下文估算，切断计费扣费连接。备注：tokenizer/token_counter 保留，调用结果只进入 usage 统计/上下文估算，不再触发扣费。
- [x] Verify 5: 搜索 `quota|Quota|billing|Billing|ratio|Ratio|余额|额度|倍率|insufficient|refund|PreConsumed`，逐项确认只剩兼容字段或 tokens usage 所需引用。备注：主代码无实际 quota/user/token/channel 加减写入；剩余主要为兼容字段、历史 DB/API 名称、token usage 统计、前端待 Phase 7 清理入口、`.bak` 噪声；后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过。

#### Phase 6: 移除预填组管理

- [x] Step 6.1: 从 `router/api-router.go` 删除 `prefill_group` routes。
- [x] Step 6.2: 删除 `controller/prefill_group.go`。
- [x] Step 6.3: 删除 `model/prefill_group.go` 并从迁移/初始化中移除。
- [x] Step 6.4: 删除 service/setting/frontend 中预填组引用。备注：已移除 `EditChannelModal.jsx` 的 `/api/prefill_group?type=model` 请求和组菜单；保留 tokens 页 Fluent prefill，因为它不是预填组管理。
- [x] Verify 6: 搜索 `prefill|Prefill|预填组|prefill_group`，确认主代码无残留。备注：`prefill_group|PrefillGroup|预填组` 命中仅来自 `.bak`；主代码 `prefill|Prefill` 仅剩 tokens 页 Fluent 预填能力；前端 `vite build` 通过，前端构建后后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过。

#### Phase 7: 前端入口精简

- [x] Step 7.1: 重新搜索 `web/classic/src`，确认非 `.bak` 的 Task/Midjourney/Suno/video/music/quota/rate/prefill 入口。备注：已定位并清理 task route/table、任务菜单、状态开关、多媒体渠道展示、Suno/MJ/Sora 模型预设与 i18n 残留；`.bak` 噪声忽略。
- [x] Step 7.2: 修改 `web/classic/src/App.jsx`，删除任务、Midjourney、视频、音乐、预填组、额度、倍率相关 routes/imports。备注：已删除 `/console/task` 与 `Task` 页面 import；预填组前端入口已在 Phase 6 删除；tokens 页 Fluent prefill 保留。
- [x] Step 7.3: 修改菜单/侧边栏/导航组件，删除对应入口。备注：已清理 `SiderBar.jsx`、`PageLayout.jsx`、sidebar module settings 与 notification module 中任务日志入口。
- [x] Step 7.4: 修改 channel constants，删除 Midjourney Proxy、Midjourney Proxy Plus、Suno API、video-only provider 展示。备注：已移除 Suno、可灵、即梦、Vidu、豆包视频、Sora 的前端渠道展示和图标；Replicate/Codex 保留。
- [x] Step 7.5: 修改 common constants，删除 task action constants。备注：已删除 task action constants 与 audio endpoint constants。
- [x] Step 7.6: 修改请求封装/API client，删除 task、prefill、quota/rate/billing endpoint 调用。备注：已删除 task log hook/table/page 文件与 `/api/task` 调用；预填组请求已在 Phase 6 删除；usage/token 统计相关 quota/rate 展示按 usage-only 兼容保留。
- [x] Step 7.7: 清理 i18n locale 中任务、视频、音乐、Midjourney、额度、倍率、预填组文案，必要时使用项目 i18n 规则同步多语言。备注：实际主 locale 仅 `zh-CN.json`；已清理已删除 UI 对应的任务日志、Midjourney、Sora 示例、Suno base URL 文案。
- [x] Verify 7: 前端搜索 `Task|任务|Midjourney|Suno|Kling|Jimeng|video|music|quota|ratio|prefill|倍率|额度`，确认只剩允许的展示或历史兼容文案。备注：主代码残留检查通过；保留 generic markdown `<video>`、tokens 页 Fluent prefill、usage/token 统计和兼容 localStorage 清理；`Push-Location web\\classic; node node_modules\\vite\\bin\\vite.js build; Pop-Location` 通过；前端 dist 更新后后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过。

#### Phase 8: 依赖清理

- [x] Step 8.1: 运行 `go mod tidy` 清理 MySQL/PostgreSQL/audio/video-task-only 依赖。备注：`go mod tidy` 已完成，主 `go.mod`/`go.sum` 不再包含目标数据库和音频解析依赖。
- [x] Step 8.2: 检查 `go.mod`、`go.sum`，确认移除 `gorm.io/driver/mysql`、`gorm.io/driver/postgres`、`github.com/go-sql-driver/mysql`、`github.com/jackc/pgx/v5`、音频解析库。备注：目标依赖命中仅来自 `.bak`。
- [x] Step 8.3: 检查前端 package 依赖，确认是否存在只服务被删除页面的依赖；仅在无其他引用时删除。备注：主源码无 `react-dropzone`、`react-telegram-login` 引用，已从 `package.json` 和直接锁文件条目移除；`bun.lock` 剩余 `leva/react-dropzone` 是其他包传递依赖。
- [x] Verify 8: 依赖文件变更符合实际引用，无误删核心 relay/provider 依赖。备注：前端 Vite 构建通过（仅 chunk size warning），后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过。

#### Phase 9: 残留引用和死代码检查

- [x] Step 9.1: 搜索 `.bak` 以外所有被删符号和 route，确认无残留引用。备注：主代码残留仅剩 `.bak` 噪声或保留的后台任务命名。
- [x] Step 9.2: 使用子代理或独立检索检查所有修改文件的 undefined 变量、未导入组件、孤立文件。备注：子代理确认无核心 route/import/orphan 编译级残留。
- [x] Step 9.3: 清理未使用 imports、常量、接口、函数。备注：清理 Midjourney option/error/i18n 常量、废弃多媒体价格项、未使用 audio helper、侧边栏 task 死配置、前端 lockfile 直接依赖残留。
- [x] Step 9.4: 检查 OpenAPI/Swagger/docs 中是否还有已删除接口入口；按需清理明显错误入口。备注：清理 `constant/README.md` 与 `docs/openapi/api.json` 中任务/Midjourney 入口；`docs/openapi/relay.json` 仍含较大的生成型 audio/video 文档残留，后续如需可单独重生成或大块清理。
- [x] Verify 9: `go test` 或编译前静态检查不再报告引用错误。备注：前端 Vite 构建通过（仅 chunk size warning），后端 `CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o new-api.exe .` 通过。

#### Phase 10: 构建与功能验证

- [x] Step 10.1: 构建前端：进入 `web/classic`，优先使用 Bun/Vite 项目脚本构建。备注：本机无 Bun 时使用项目本地 Vite CLI，`vite build` 通过，仅有大 chunk warning。
- [x] Step 10.2: 构建后端：`$env:CGO_ENABLED="0"; go build -ldflags="-s -w" -trimpath -o new-api.exe .` 备注：最终构建通过。
- [x] Step 10.3: 记录精简后 `new-api.exe` 体积，并与 Phase 0 基线对比。备注：最终 `new-api.exe` 55,567,360 bytes，约 52.99 MB；Phase 0 基线约 58.6 MB，减少约 5.61 MB；`web/classic/dist` 约 11.11 MB，基线约 11.17 MB，减少约 0.06 MB。
- [x] Step 10.4: Smoke test：启动服务，验证登录/设置页/渠道列表/API token/模型列表可用。备注：`/`、`/api/status` 200；登录路由存在；模型、渠道、token 管理入口在未登录时返回 401，说明路由存在且受保护。
- [x] Step 10.5: Smoke test：验证核心 API 聚合路径 `models`、`chat completions`、`responses`、`embeddings` 至少不会因路由缺失或 panic 失败。备注：`/v1/models` 未授权返回 401；`/v1/chat/completions`、`/v1/responses`、`/v1/embeddings` 在无可用渠道/无有效请求环境时返回 503，未出现 404 或 panic。
- [x] Step 10.6: Smoke test：确认已删除 audio/video/music/task/prefill/quota/rate routes 返回 404 或不存在入口。备注：`/v1/audio/transcriptions`、`/v1/videos`、`/suno/submit/music`、`/api/task/`、`/api/prefill_group/`、`/api/rate/` 返回 404；`/api/user/quota` 返回 401（仍受历史用户路由保护但无公开功能入口）。修复 `router/web-router.go`，避免 `/suno*` 被 SPA fallback 返回 200。
- [x] Verify 10: 前端构建成功、后端构建成功、核心功能 smoke test 通过、删除功能无入口。备注：子代理复核运行时路由无 `/suno`、`/api/task`、`/api/prefill_group`、`/v1/audio`、`/v1/videos` 注册；`docs/openapi/relay.json` 仍有生成型 audio/video 文档残留，后续可单独重生成或清理。

### Rollback Plan

1. 若某个 Phase 出现不可控编译失败，优先回滚该 Phase 中最后一批文件修改，不回滚已验证通过的前置 Phase。
2. 若 SQLite-only 改造失败，恢复 `model/main.go`、`common/database.go` 和数据库相关 tests，然后重新以更小步骤删除 MySQL/PostgreSQL 分支。
3. 若音频/视频/task 删除后出现核心 relay 断裂，先恢复 route 之外的必要公共 helper，再继续拆分引用。
4. 若计费/额度精简影响认证/API token，优先恢复 token 认证字段和 middleware 所需逻辑，但保持扣费 no-op。
5. 若前端入口删除导致构建失败，先恢复最小 import/route，再按组件级别逐个删除。
6. 所有回滚都必须同步更新本文件 checkbox 状态和备注。

### Risks

- 数据库代码中部分 SQL 分支可能同时承载业务差异，删除时需确认 SQLite 表达式等价。
- `task` 命名可能用于普通后台任务，例如 Codex 凭据刷新；不能因文件名含 task 就盲删。
- 计费/额度字段可能与用户、token、日志表结构耦合；删除业务逻辑时需谨慎处理历史数据库字段，避免迁移破坏现有 SQLite 数据。
- 前端 `.bak` 搜索结果容易误导，实施时必须只处理非 `.bak` 文件。
- 删除音频/video provider 时可能影响部分 channel type 显示，需确认核心 chat/responses/embeddings provider 不受影响。
- Tokenizer 不能删除；否则会影响 token usage 统计、上下文估算和部分请求限制。

## Phase 完成记录

- [x] Phase 0 完成后：读取本文件，并提示 Phase 1 的下一步内容。
- [x] Phase 1 完成后：读取本文件，并提示 Phase 2 的下一步内容。
- [x] Phase 2 完成后：读取本文件，并提示 Phase 3 的下一步内容。
- [x] Phase 3 完成后：读取本文件，并提示 Phase 4 的下一步内容。
- [x] Phase 4 完成后：读取本文件，并提示 Phase 5 的下一步内容。
- [x] Phase 5 完成后：读取本文件，并提示 Phase 6 的下一步内容。
- [x] Phase 6 完成后：读取本文件，并提示 Phase 7 的下一步内容。
- [x] Phase 7 完成后：读取本文件，并提示 Phase 8 的下一步内容。
- [x] Phase 8 完成后：读取本文件，并提示 Phase 9 的下一步内容。
- [x] Phase 9 完成后：读取本文件，并提示 Phase 10 的下一步内容。
- [x] Phase 10 完成后：读取本文件，并汇总最终结果。
