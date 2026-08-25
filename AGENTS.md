# AGENTS.md — new-api-lite 项目约定

## 概述

本项目（new-api-lite）基于 [new-api-1.0.0-rc.10](https://github.com/QuantumNous/new-api) 进行精简调整。

new-api-lite 是一个使用 Go 构建的 AI API 网关/代理。它通过统一 API 聚合 40 多个上游 AI 提供商（OpenAI、Claude、Gemini、Azure、AWS Bedrock 等），并提供用户管理、速率限制和管理后台。

**与原项目（new-api）的核心区别**：本项目已移除所有付费/计费相关功能，包括但不限于：
- 用户额度（quota）、token 额度、余额不足、扣费/返还逻辑
- 渠道倍率、模型倍率、分组倍率、倍率计价
- 预填组管理（prefill group）
- 音频 API（transcriptions/translations/speech）及音频处理依赖
- 视频/音乐 API（Suno/Kling/Jimeng/Midjourney/异步任务）及任务轮询/计费
- MySQL/PostgreSQL 数据库支持（仅保留 SQLite）

本项目保留的核心能力：models、chat completions、responses、embeddings、必要的 image relay、用户认证、API token、渠道管理、请求日志、tokens 消耗统计。

## 技术栈

- **后端**：Go 1.22+、Gin Web 框架、GORM v2 ORM
- **前端**：React 18、Vite、Semi Design
- **数据库**：SQLite（仅支持 SQLite，已移除 MySQL/PostgreSQL 支持）
- **缓存**：Redis（go-redis）+ 内存缓存
- **认证**：JWT、两步验证（2FA）、Codex 渠道 OAuth 授权
- **前端包管理器**：Bun（优先于 npm/yarn/pnpm）

## 架构

分层架构：Router -> Controller -> Service -> Model

```
router/        — HTTP 路由（API、relay、dashboard、web）
controller/    — 请求处理器
service/       — 业务逻辑（无计费/扣费/额度逻辑，仅保留 tokens 消耗统计）
model/         — 数据模型和数据库访问（GORM，仅 SQLite）
relay/         — AI API 中继/代理，包含提供商适配器
  relay/channel/ — 提供商专用适配器（openai/、claude/、gemini/、aws/ 等，无 audio/video/task adaptor）
middleware/    — 认证、速率限制、CORS、日志、分发
setting/       — 配置管理（模型、操作、系统、性能；已移除倍率配置）
common/        — 共享工具（JSON、加密、Redis、环境变量、速率限制等）
dto/           — 数据传输对象（请求/响应结构体，无 audio/video/task DTO）
constant/      — 常量（API 类型、渠道类型、上下文键，无 task/audio 常量）
types/         — 类型定义（中继格式、文件来源、错误，无 audio/video/task 类型）
i18n/          — 后端国际化（go-i18n，en/zh）
pkg/           — 内部包（cachex、ionet）
web/             — 前端主题容器
 web/classic/   — 经典前端（React 18、Vite、Semi Design）
  web/classic/src/i18n/ — 前端国际化（i18next，zh/en/fr/ru/ja/vi）
```

## 国际化（i18n）

### 后端（`i18n/`）
- 库：`nicksnyder/go-i18n/v2`
- 语言：en、zh

### 前端（`web/classic/src/i18n/`）
- 库：`i18next` + `react-i18next` + `i18next-browser-languagedetector`
- 语言：en（基础语言）、zh（回退语言）、fr、ru、ja、vi
- 翻译文件：`web/classic/src/i18n/locales/{lang}.json` — 扁平 JSON，键为中文源字符串
- 用法：使用 `useTranslation()` hook，在组件中调用 `t('English key')`

## 规则

### 规则 1：JSON 包 — 使用 `common/json.go`

所有 JSON marshal/unmarshal 操作必须使用 `common/json.go` 中的封装函数：

- `common.Marshal(v any) ([]byte, error)`
- `common.Unmarshal(data []byte, v any) error`
- `common.UnmarshalJsonStr(data string, v any) error`
- `common.DecodeJson(reader io.Reader, v any) error`
- `common.GetJsonType(data json.RawMessage) string`

不要在业务代码中直接导入或调用 `encoding/json`。这些封装用于保持一致性，并便于未来扩展（例如替换为更快的 JSON 库）。

注意：仍可将 `encoding/json` 中的 `json.RawMessage`、`json.Number` 等类型定义作为类型引用，但实际的 marshal/unmarshal 调用必须通过 `common.*` 完成。

### 规则 2：数据库 — 仅 SQLite

本项目已移除 MySQL/PostgreSQL 支持，仅使用 SQLite。

**使用 GORM 抽象：**
- 优先使用 GORM 方法（`Create`、`Find`、`Where`、`Updates` 等），而不是原始 SQL。
- 让 GORM 处理主键生成 — 不要直接使用 `AUTO_INCREMENT` 或 `SERIAL`。

**SQLite 注意事项：**
- 列名引用使用 `` `column` `` 格式（SQLite/MySQL 风格）。
- 对于 `group` 和 `key` 等保留字列，使用 `model/main.go` 中的 `commonGroupCol`、`commonKeyCol` 变量。
- 布尔值使用 `1`/`0`。
- SQLite 不支持 `ALTER COLUMN`，迁移时使用 `ALTER TABLE ... ADD COLUMN`（模式参考 `model/main.go`）。
- 存储 JSON 时使用 `TEXT`，而不是 `JSONB`。

**禁止使用：**
- MySQL/PostgreSQL 专用函数或操作符
- `gorm.io/driver/mysql`、`gorm.io/driver/postgres`、`github.com/go-sql-driver/mysql`、`github.com/jackc/pgx` 等数据库驱动
- `common.UsingMySQL`、`common.UsingPostgreSQL` 等多数据库分支标志

### 规则 3：前端 — 优先使用 Bun

前端（`web/classic/` 目录）优先使用 `bun` 作为包管理器和脚本运行器：
- `bun install` 用于安装依赖
- `bun run dev` 用于开发服务器
- `bun run build` 用于生产构建

前后端分离且后端打包内置前端产物时，必须先编译前端，再编译后端，避免后端编译时使用旧的前端构建数据。

### 规则 4：新渠道的 StreamOptions 支持

实现新渠道时：
- 确认提供商是否支持 `StreamOptions`。
- 如果支持，将该渠道添加到 `streamSupportedChannels`。


### 规则 5：上游中继请求 DTO — 保留显式零值

对于从客户端 JSON 解析后又重新 marshal 给上游提供商的请求结构体（尤其是 relay/convert 路径）：

- 可选标量字段必须使用带 `omitempty` 的指针类型（例如 `*int`、`*uint`、`*float64`、`*bool`），不能使用非指针标量。
- 语义必须是：
  - 客户端 JSON 中字段缺失 => `nil` => marshal 时省略；
  - 字段显式设置为零值/false => 非 `nil` 指针 => 仍必须发送给上游。
- 避免对可选请求参数使用带 `omitempty` 的非指针标量，因为零值（`0`、`0.0`、`false`）会在 marshal 时被静默丢弃。

### 规则 6：子代理优先

子代理可用时，优先启动一个或多个子代理进行只读代码检索和上下文探索，尽量避免使用本机内存和 GPU 做本地检索。

### 规则 7：禁止引入付费/计费/额度/倍率功能

本项目已从原版 new-api 移除所有付费和计费相关功能，**禁止重新引入**以下内容：

- **用户额度**：用户 quota 增减、余额不足、额度转移、邀请奖励等
- **token 额度**：token quota 扣减、额度校验、RemainQuota/UnlimitedQuota 业务逻辑
- **渠道倍率/模型倍率/分组倍率**：倍率计价、价格配置、倍率热更新
- **扣费/返还**：PreConsumedQuota、PostConsumeQuota、refund、insufficient quota 等
- **预填组管理**：prefill_group API、模型预填组配置
- **计费表达式**：billingexpr 包、动态计价、quota 数学运算（本项目 quota 固定为 0）

**允许保留的兼容字段**：数据库中历史 quota/ratio 字段可保留为兼容空值（返回 0 或默认值），但不得恢复其业务逻辑。tokens 消耗统计（prompt/completion/total tokens）和 request count 统计继续保留，仅用于监控和分析，不用于计费。

### 规则 8：禁止引入音频/视频/任务等已移除功能

本项目已移除音频、视频、音乐和异步任务相关 API，**禁止重新引入**以下内容：

- **音频 API**：`/v1/audio/transcriptions`、`/v1/audio/translations`、`/v1/audio/speech` 及对应 handler/adaptor/DTO
- **视频/音乐 API**：Suno、Kling、Jimeng、Midjourney、Sora 等多媒体生成接口
- **异步任务系统**：task submit/fetch、task polling、task billing、task log
- **视频代理**：VideoProxy、Gemini video proxy 等
- **音频处理依赖**：`common/audio.go`、音频时长解析等

**允许保留的后台任务**：`StartCodexCredentialAutoRefreshTask()` 和 `StartChannelUpstreamModelUpdateTask()` 等内部维护任务不在禁止范围内。

### 规则 9：项目调整计划管理

每次列出项目调整计划时，必须遵循以下计划管理流程：

**1. 主计划文件（`plan.md`）**
- 每次列出调整计划时，必须更新 `plan.md`，记录主计划摘要
- `plan.md` 必须包含详细计划索引目录，指向 `.plan/` 目录下的详细计划文件
- 索引格式示例：
  ```
  ## 详细计划索引
  - [.plan/20260720-01.md](.plan/20260720-01.md) — 精简任务计划（已完成）
  - [.plan/20260720-02.md](.plan/20260720-02.md) — XXX 调整计划（进行中）
  ```

**2. 详细计划文件（`.plan/` 目录）**
- 详细调整计划按格式写入 `.plan/` 目录，文件名格式为 `YYYYMMDD-NN.md`（如 `.plan/20260720-01.md`、`.plan/20260720-02.md`）
- 详细计划模板参考现有 `plan.md` 文件格式，应包含：
  - 执行规则（checkbox 更新、Phase 完成后汇报等）
  - Current State / Target State
  - Affected Files
  - 分 Phase 的 Execution Plan（每步带 `[ ]`/`[x]` checkbox）
  - 每个 Phase 的 Verify 步骤
  - Rollback Plan / Risks（如适用）
  - Phase 完成记录

**3. Phase 完成汇报流程**
- 每个 Phase 完成时必须：
  1. 汇报当前调整计划的进度
  2. 更新状态到详细计划文件（将已完成步骤的 `[ ]` 更新为 `[x]`）
  3. 读取 `plan.md` 确认整体状态
  4. 若详细计划未完成，进入索引的详细计划文件，提示下一个 Phase 的内容
  5. 若详细计划已完成，在 `plan.md` 中更新索引状态为"已完成"

