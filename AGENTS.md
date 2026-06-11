# AGENTS.md — new-api 项目约定

## 概述

这是一个使用 Go 构建的 AI API 网关/代理。它通过统一 API 聚合 40 多个上游 AI 提供商（OpenAI、Claude、Gemini、Azure、AWS Bedrock 等），并提供用户管理、计费、速率限制和管理后台。

## 技术栈

- **后端**：Go 1.22+、Gin Web 框架、GORM v2 ORM
- **前端**：React 18、Vite、Semi Design
- **数据库**：SQLite、MySQL、PostgreSQL（必须同时支持三者）
- **缓存**：Redis（go-redis）+ 内存缓存
- **认证**：JWT、两步验证（2FA）、Codex 渠道 OAuth 授权
- **前端包管理器**：Bun（优先于 npm/yarn/pnpm）

## 架构

分层架构：Router -> Controller -> Service -> Model

```
router/        — HTTP 路由（API、relay、dashboard、web）
controller/    — 请求处理器
service/       — 业务逻辑
model/         — 数据模型和数据库访问（GORM）
relay/         — AI API 中继/代理，包含提供商适配器
  relay/channel/ — 提供商专用适配器（openai/、claude/、gemini/、aws/ 等）
middleware/    — 认证、速率限制、CORS、日志、分发
setting/       — 配置管理（倍率、模型、操作、系统、性能）
common/        — 共享工具（JSON、加密、Redis、环境变量、速率限制等）
dto/           — 数据传输对象（请求/响应结构体）
constant/      — 常量（API 类型、渠道类型、上下文键）
types/         — 类型定义（中继格式、文件来源、错误）
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

### 规则 2：数据库兼容性 — SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6

所有数据库代码必须同时完全兼容这三种数据库。

**使用 GORM 抽象：**
- 优先使用 GORM 方法（`Create`、`Find`、`Where`、`Updates` 等），而不是原始 SQL。
- 让 GORM 处理主键生成 — 不要直接使用 `AUTO_INCREMENT` 或 `SERIAL`。

**无法避免原始 SQL 时：**
- 列名引用方式不同：PostgreSQL 使用 `"column"`，MySQL/SQLite 使用 `` `column` ``。
- 对于 `group` 和 `key` 等保留字列，使用 `model/main.go` 中的 `commonGroupCol`、`commonKeyCol` 变量。
- 布尔值不同：PostgreSQL 使用 `true`/`false`，MySQL/SQLite 使用 `1`/`0`。请使用 `commonTrueVal`/`commonFalseVal`。
- 使用 `common.UsingPostgreSQL`、`common.UsingSQLite`、`common.UsingMySQL` 标志为数据库特定逻辑分支。

**没有跨数据库回退方案时禁止使用：**
- MySQL 专用函数（例如没有 PostgreSQL `STRING_AGG` 等价实现的 `GROUP_CONCAT`）
- PostgreSQL 专用操作符（例如 `@>`、`?`、`JSONB` 操作符）
- SQLite 中的 `ALTER COLUMN`（不支持 — 使用添加列的变通方案）
- 没有回退方案的数据库专用列类型 — 存储 JSON 时使用 `TEXT`，而不是 `JSONB`

**迁移：**
- 确保所有迁移都能在三种数据库上工作。
- 对于 SQLite，使用 `ALTER TABLE ... ADD COLUMN`，而不是 `ALTER COLUMN`（模式参考 `model/main.go`）。

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

