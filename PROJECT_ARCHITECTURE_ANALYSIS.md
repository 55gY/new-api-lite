# new-api-lite：目录结构与核心代码架构分析

**分析范围。** 本报告基于工作区当前仓库快照（`main`，提交 `1933148`）的源码、构建描述与目录枚举形成，重点覆盖服务端启动流程、HTTP 路由、鉴权与通道调度、协议中继、数据模型以及前端资源交付。它描述的是现有实现的结构与调用关系，而非部署配置审计或安全渗透测试。[1]

> **结论。** `new-api-lite` 是一个以 Go/Gin 为运行时核心的单体式 AI API 网关与管理控制台。它把 OpenAI、Claude、Gemini 等多种客户端协议统一接入，将请求按令牌权限、模型、分组和通道能力进行调度，并通过“协议处理器 + 上游适配器”层把请求转换后转发至不同提供商；同时内置用户、令牌、额度、日志、通道和系统配置等运营能力。[2] [3]

| 维度 | 架构判断 | 主要证据 |
|---|---|---|
| 部署单元 | 单个 Go 服务进程，HTTP 服务器由 Gin 承载 | `main.go` 创建 `gin.Engine` 并监听 `PORT`。[2] |
| 产品形态 | API 中继网关 + Web 管理控制台 | 路由同时装配 API、控制台、中继与静态前端。[3] |
| 主要语言与框架 | Go、Gin；前端为 React/Vite | 服务端依赖与前端构建清单分别位于 `go.mod`、`web/classic/package.json`。[4] |
| 核心业务抽象 | 用户/令牌、通道、模型能力、额度与日志 | GORM 实体定义分散于 `model/`。[5] |
| 扩展核心 | 统一 `Adaptor` 接口 + 按 `APIType` 的适配器工厂 | `relay/channel/adapter.go` 与 `relay/relay_adaptor.go`。[6] [7] |

## 1. 目录结构：按职责而非严格分层组织

仓库的目录结构总体遵循“分层基础设施 + 业务领域模块 + 协议适配器”的组织方式，但并非严格的 Clean Architecture。`controller`、`middleware`、`service`、`model` 与 `relay` 通过直接导入协作；Gin 的 `Context` 在各层间携带请求状态，因此请求路径清晰但运行时耦合较高。仓库中约有 550 个 Go 源文件，其中 `relay/` 是体量最大的业务子系统，反映出多上游协议兼容是项目的复杂性中心。

| 目录/文件 | 职责 | 与核心链路的关系 |
|---|---|---|
| `main.go` | 进程启动、资源初始化、后台任务与 Gin 装配 | 运行时根节点。[2] |
| `router/` | 将 API、控制台、模型中继和 Web 静态资源装配到 Gin | HTTP 入口层。[3] |
| `middleware/` | 身份认证、限流、请求 ID、国际化、日志、性能检查、通道分发 | 在控制器之前建立安全与执行上下文。[8] [9] |
| `controller/` | 面向 HTTP 的业务编排；包括管理 API、用户/令牌/通道操作和统一中继入口 | 请求从路由进入具体用例。[10] |
| `service/` | 可复用业务服务，如通道选择、重试、额度结算、异常策略、HTTP 客户端 | 承担跨控制器的领域规则。[11] |
| `model/` | GORM 数据实体、数据库初始化、缓存、通道能力索引与持久化逻辑 | 用户、令牌、渠道、日志和配置的事实来源。[5] |
| `dto/` | 外部 API 请求/响应 DTO | 各协议处理器与适配器之间的数据契约。 |
| `relay/` | 协议中继编排、OpenAI/Claude/Gemini 等请求处理、转换辅助与上游适配 | 产品的转发与兼容性核心。[6] [7] |
| `relay/channel/<provider>/` | 各上游提供商的实现，例如 OpenAI、Claude、Gemini、AWS、Ali 等 | 将统一抽象落地为上游 HTTP/流式响应处理。[7] |
| `common/`、`constant/`、`types/` | 配置、缓存、序列化、通用工具、常量和统一错误/上下文类型 | 横切基础设施与领域值对象。 |
| `setting/` | 模型、比率、账单、控制台、性能与系统配置的领域化配置模块 | 影响调度、计费和运营策略。 |
| `i18n/` | 服务端语言包与懒加载逻辑 | 供中间件和 API 错误信息使用。[2] |
| `pkg/` | 较独立的基础包，如缓存、网络与性能指标 | 为主业务提供可复用能力。 |
| `web/classic/` | React/Vite 管理前端源码 | 构建后以嵌入或静态资源形式由 Go 服务交付。[4] |
| `docs/`、`docker-compose*.yml`、`Dockerfile*`、`makefile` | 使用文档、容器化与构建入口 | 开发和部署配套，而不是请求处理主链路。 |

## 2. 运行时启动：先建立状态，再暴露 HTTP 服务

启动入口采用明确的“初始化资源—启动后台同步—装配 HTTP”顺序。`InitResources()` 先读取 `.env`、初始化全局环境和日志，再初始化 HTTP 客户端、令牌编码器、主数据库、选项映射、日志数据库、Redis、性能指标与国际化；只有这些基础能力准备完成后，主函数才继续建立缓存与服务端路由。[2]

随后，进程会根据配置启用通道缓存与周期同步、选项同步、看板额度数据更新、自动测渠道、Codex 凭据刷新以及上游模型更新等后台任务。这个设计表明数据库是持久化来源，而进程内/Redis 缓存承担读路径加速；服务不是“无状态纯代理”，而是带有持续运营任务的网关。[2]

| 启动阶段 | 关键动作 | 设计含义 |
|---|---|---|
| 环境与基础设施 | `.env`、全局环境、日志、HTTP 客户端、令牌编码器 | 统一运行时依赖和可观测性。 |
| 持久化与配置 | 主库、日志库、选项映射、Redis | 先恢复业务状态，再接受流量。 |
| 缓存与后台任务 | 通道缓存同步、配置同步、配额看板、渠道检测和凭据刷新 | 通过异步任务维护调度与运营数据的时效性。 |
| HTTP 装配 | Recovery、请求 ID、国际化、日志、Cookie Session、路由与前端资源 | 以 Gin 中间件链承接全部外部流量。 |

## 3. 路由拓扑：控制面、数据面与前端共用一个进程

`router.SetRouter` 依次装配 API、Dashboard 和 Relay 路由，再决定是本地交付前端静态资源，还是将非 API 路径重定向到 `FRONTEND_BASE_URL`。因此项目在同一进程中同时提供控制面（用户、令牌、渠道、配置等管理接口）、数据面（模型转发 API）与 UI 资源；部署时可以选择前后端同域或将前端托管到独立地址。[3]

```text
客户端 / 浏览器
   │
   ├── 管理与业务 API ──> API/Dashboard 路由组 ──> 认证与业务控制器
   │
   ├── 模型 API
   │     /v1/chat/completions、/v1/messages、/v1beta/models/*
   │        └── Relay 路由组 ──> TokenAuth ──> 限流/性能检查 ──> Distribute ──> controller.Relay
   │
   └── Web 页面 ──> 静态文件或 SPA fallback / 外部前端重定向
```

| 路由区域 | 典型行为 | 核心边界 |
|---|---|---|
| Relay 路由 | 接受 OpenAI、Claude、Gemini、Embedding、Rerank、图片与实时 WebSocket 请求 | 统一鉴权、模型级限流和通道分发后进入 `controller.Relay`。[12] |
| API/Dashboard 路由 | 为控制台和管理能力提供接口 | 与令牌/用户角色认证和运营数据模型相连。[3] |
| Web 路由 | Gzip、Web 限流、缓存、嵌入式静态资源与 SPA 回退 | 与 API 路径隔离，未知 API 路径返回网关风格的 404。[13] |

## 4. 核心请求链路：从客户端协议到上游协议

中继请求的设计重点是把“协议、鉴权、模型、计费和供应商”解耦为连续步骤。入口路由会为模型 API 叠加 `TokenAuth`、性能检查、模型限流和 `Distribute`；其中 `Distribute` 从 JSON、表单、Gemini URL 或 WebSocket 查询参数中提取模型名，检查令牌模型白名单，优先使用可能存在的通道亲和性，然后从可用通道中选取目标，并把渠道密钥、模型映射、上游地址和特定渠道参数写入 Gin Context。[9] [12]

`controller.Relay` 负责协议级总编排：验证请求体并生成 `RelayInfo`，执行可选敏感词检查和输入 token 预估，随后按“原模型重试阶段—映射模型重试阶段”循环选择通道。发生可重试的通道故障时，控制器会记录已用通道、按策略切换渠道；针对特定上游错误还会触发模型级禁用或渠道自动禁用。这个控制器是**可靠性策略的中心**，而不是简单的 HTTP 转发器。[10]

```text
HTTP / WebSocket 请求
  → 全局中间件（Recovery、Request ID、i18n、日志、Session）
  → Relay 路由中间件（TokenAuth、限流、性能检查）
  → Distribute（模型解析、权限检查、通道初选、Context 注入）
  → controller.Relay（验证、RelayInfo、敏感检查、估算、重试）
  → 协议处理器（Text / Claude / Gemini / Image / Embedding / Rerank / Responses）
  → Adaptor 工厂与提供商适配器（请求转换、HTTP 转发、流式/普通响应转换）
  → 用量计算、额度扣减、消费/错误日志、性能样本
  → 统一格式的客户端响应
```

| 步骤 | 主要代码职责 | 关键机制 |
|---|---|---|
| 认证 | `middleware.TokenAuth` | 接受 `Authorization`，并兼容 Anthropic 的 `x-api-key`、Gemini 查询参数/请求头以及实时 WebSocket 的子协议携带方式；验证后写入用户、令牌、额度与模型限制上下文。[8] |
| 分发 | `middleware.Distribute` | 解析模型名，执行业务令牌模型限制、亲和通道优先和可用通道选择。[9] |
| 编排与重试 | `controller.Relay` | 统一处理请求校验、敏感词、token 估算、原模型/映射模型两阶段重试、错误归一化与自动禁用。[10] |
| 协议处理 | `relay/*.go` | 按请求类型调用 Text、Claude、Gemini、Images、Embedding、Rerank、Responses 或 WebSocket 处理器。[10] |
| 上游适配 | `relay/channel` | 根据 `APIType` 创建适配器，将统一 DTO 转换为各供应商所需请求，并解析普通或 SSE 响应。[6] [7] |
| 结算与观测 | `service.PostTextConsumeQuota`、日志与性能采样 | 在成功响应后依据 usage 扣减额度；失败路径会留下带渠道、模型、请求 ID 的错误记录。[10] [14] |

## 5. 通道调度：能力索引、优先级权重与故障转移

`Channel` 表示一个可调用的上游配置；`Ability` 以“分组、模型、通道”为主键记录通道是否能提供某模型，以及优先级、权重、探测状态和响应时间。启动时初始化通道缓存，并可以按固定频率同步缓存。对于同一模型的可用通道，调度逻辑先依照**优先级**选择一个层级，再在该层级内按**权重随机**选择具体通道；当当前优先级层重试耗尽时，下一次会转向其他优先级层。[5] [11] [15]

该方案把“管理层配置通道能力”与“请求期快速选路”分开：管理操作主要影响数据库与缓存，而转发热路径尽量从内存索引读取。与此同时，分发中间件和中继控制器都能选择通道：前者负责首选通道及早期上下文初始化，后者负责请求期间的重试重选。这样可降低首包延迟并支持故障切换，但也要求二者对 Context 和缓存一致性保持严格约定。[9] [10]

| 调度能力 | 实现方式 | 效果 |
|---|---|---|
| 模型可用性 | `Ability(group, model, channel_id)` | 将“渠道支持哪些模型”从渠道基本信息中拆出。 |
| 优先级 | 重试索引映射到降序优先级层 | 先使用优先级更高的通道。 |
| 负载分摊 | 同优先级内按通道权重随机 | 控制各通道的流量占比。 |
| 通道亲和性 | 优先读取已绑定渠道；成功后记录亲和关系 | 增强会话/重复请求的一致性。 |
| 多密钥渠道 | `GetNextEnabledKey` 选择可用密钥并写入 Context | 单渠道可承载密钥池。 |
| 故障处置 | 可重试错误切换渠道；模型或渠道可自动禁用 | 降低单上游故障的用户影响。 |

## 6. 协议适配：稳定的抽象，集中式的注册

适配器接口规定了完整上游交互生命周期：初始化中继信息、确定请求 URL、设置请求头、分别转换 OpenAI/Claude/Gemini/图片/Embedding/Rerank/Responses 请求、发起请求、处理响应以及返回模型/渠道元信息。`TextHelper` 展示了典型路径：它进行模型映射和请求参数归一化，处理流式 usage、直通请求体、系统提示词注入、参数覆写和禁用字段，然后调用适配器发送请求并解析响应，最后进入额度扣减流程。[6] [14]

提供商选择通过 `GetAdaptor(apiType)` 中的显式 `switch` 完成。其优点是类型关系直接、无运行时注册隐患，且实现定位非常容易；代价是新增供应商需同时修改常量、渠道配置、工厂分支及适配器实现，属于**集中式扩展点**。对于目前以多供应商兼容为主要复杂度的项目，这一取舍是务实的，但工厂文件和通用接口的演进应被视为高变更风险区域。[7]

| 抽象层 | 责任 | 示例 |
|---|---|---|
| 客户端协议处理器 | 决定怎样读取和返回 OpenAI/Claude/Gemini 等客户端协议 | `TextHelper`、`ClaudeHelper`、`GeminiHelper`。 |
| 通用中继信息 | 将用户、令牌、模型、通道、流式状态、价格和重试状态集中到 `RelayInfo` | 控制器与适配器之间的运行时契约。 |
| 提供商适配器 | 实现 URL、头、请求 DTO、HTTP I/O、SSE/usage 解析 | `openai.Adaptor`、`claude.Adaptor`、`gemini.Adaptor` 等。 |
| 配置覆盖层 | 模型映射、禁用字段、参数覆盖、Header 覆盖、系统提示词 | 让同一上游能服务不同客户策略。[9] [14] |

## 7. 数据与配置模型：网关运营状态的核心

数据模型以 GORM 实体为中心。`User` 与 `Token` 负责身份、授权和额度边界；`Channel` 保存上游连接与运行策略；`Ability` 表达“某渠道在某分组中可提供某模型”的调度索引；`Log` 记录消费、错误、请求 ID、上游请求 ID、模型、通道和 token 使用情况；`Option` 提供键值形式的系统配置。配置目录还将模型、比率、账单、系统、性能等规则拆分为子模块，使价格与运营策略不必硬编码在协议处理器内。[5]

这组模型意味着项目的核心不是单纯反向代理，而是一个有账号体系、额度治理、渠道资源池和运营审计能力的多租户网关。尤其是 `Log` 保存渠道、用户、token、时间、用量和请求关联字段，使故障排查可以从客户端请求 ID 追溯到具体的上游通道。[5] [10]

| 实体 | 主要责任 | 影响路径 |
|---|---|---|
| `User` | 用户身份、角色和状态 | 管理认证与 API token 的归属校验。 |
| `Token` | API key、余额/不限额、模型限制、IP 限制与渠道指定权限 | `TokenAuth` 后写入请求上下文。[8] |
| `Channel` | 上游地址、密钥、类型、模型映射、参数/头覆盖、优先级与权重 | 分发、重试与适配器配置来源。[9] |
| `Ability` | 渠道—模型—分组的可用性、权重和探测结果 | 缓存调度的基础索引。[5] [15] |
| `Log` | 消费/错误/系统事件和链路关联 | 额度审计、问题定位与管理端看板。 |
| `Option` | 系统级可持久化配置 | 启动后加载并周期同步。[2] [5] |

## 8. 架构特征、优点与演进关注点

该架构的最大优点是业务闭环完整：客户端兼容层、渠道调度、故障切换、额度计费、日志和管理前端共用同一领域模型；而适配器接口让新增或维护上游协议时能集中在 `relay/channel/<provider>` 中处理。启动期缓存和运行期后台同步也符合 API 网关高频读、配置低频写的工作负载特征。[2] [6] [15]

同时，代码的复杂度集中在 `relay/` 和围绕 Gin Context 的跨层协作。`controller.Relay` 同时处理校验、内容治理、计量、重试、错误处理和日志触发；`middleware.Distribute` 与其共同参与通道选择；`GetAdaptor` 则是静态的集中注册点。后续重构应优先保持这些边界的行为不变，并以可测接口收拢 Context 键和重试状态，而不宜先拆分为大量独立服务，因为数据库、缓存和运营规则当前仍高度共享。

| 关注点 | 当前体现 | 建议的演进方向 |
|---|---|---|
| 中继控制器复杂度 | `controller.Relay` 负责多项横切决策 | 将“选择—执行—失败分类—记账”抽成可单测的服务编排对象，但保留现有 HTTP 契约。 |
| Context 隐式耦合 | 认证、分发和适配器通过大量 Context 键传递状态 | 为上下文构建/读取提供强类型封装，并集中维护键与必填字段。 |
| 适配器注册 | 工厂 `switch` 需随供应商扩展而增长 | 保持显式注册的可读性，同时可采用注册表减少工厂修改面。 |
| 缓存一致性 | 数据库、内存缓存、Redis 与后台同步并存 | 对渠道变更、自动禁用和缓存刷新建立明确的失效与可观测规则。 |
| 协议回归测试 | 协议类型、流式语义与模型映射组合很多 | 用固定请求夹具覆盖每个提供商适配器的转换、错误响应与 SSE 解析。 |

## 9. 建议的源码阅读顺序

若目标是快速修改功能，建议先从启动和路由建立全局心智模型，再沿一个典型的 `/v1/chat/completions` 请求进入认证、分发、中继控制器、文本处理器与 OpenAI 适配器。若目标是新增上游，先阅读适配器接口、目标协议的相近实现和工厂映射；若目标是修改调度或计费，则优先阅读 `model/channel_cache.go`、`service/channel_select.go`、`middleware/distributor.go` 和各 `setting` 子模块。

| 目标 | 推荐入口 | 后续追踪 |
|---|---|---|
| 理解总体运行 | `main.go` → `router/main.go` | `router/relay-router.go` → 中间件。 |
| 修改 API 转发 | `controller/relay.go` | `relay/compatible_handler.go` → 相应 provider 适配器。 |
| 新增供应商 | `relay/channel/adapter.go` | `relay/relay_adaptor.go` → `constant`/渠道配置。 |
| 修改通道调度 | `middleware/distributor.go` | `service/channel_select.go` → `model/channel_cache.go`。 |
| 修改鉴权/额度 | `middleware/auth.go` | `model/token.go`、`model/user.go`、消费/日志服务。 |

## References

[1] [仓库快照提交 `1933148`](https://github.com/55gY/new-api-lite/commit/1933148)  
[2] [应用启动与资源初始化：`main.go`](https://github.com/55gY/new-api-lite/blob/main/main.go)  
[3] [顶层路由装配：`router/main.go`](https://github.com/55gY/new-api-lite/blob/main/router/main.go)  
[4] [Go 模块依赖：`go.mod`](https://github.com/55gY/new-api-lite/blob/main/go.mod)；[Classic 前端构建清单：`web/classic/package.json`](https://github.com/55gY/new-api-lite/blob/main/web/classic/package.json)  
[5] [数据模型目录：`model/`](https://github.com/55gY/new-api-lite/tree/main/model)  
[6] [通道适配器接口：`relay/channel/adapter.go`](https://github.com/55gY/new-api-lite/blob/main/relay/channel/adapter.go)  
[7] [提供商适配器工厂：`relay/relay_adaptor.go`](https://github.com/55gY/new-api-lite/blob/main/relay/relay_adaptor.go)  
[8] [认证与令牌上下文：`middleware/auth.go`](https://github.com/55gY/new-api-lite/blob/main/middleware/auth.go)  
[9] [模型解析与通道分发：`middleware/distributor.go`](https://github.com/55gY/new-api-lite/blob/main/middleware/distributor.go)  
[10] [统一中继控制器：`controller/relay.go`](https://github.com/55gY/new-api-lite/blob/main/controller/relay.go)  
[11] [通道选择服务：`service/channel_select.go`](https://github.com/55gY/new-api-lite/blob/main/service/channel_select.go)  
[12] [中继路由：`router/relay-router.go`](https://github.com/55gY/new-api-lite/blob/main/router/relay-router.go)  
[13] [前端静态资源路由：`router/web-router.go`](https://github.com/55gY/new-api-lite/blob/main/router/web-router.go)  
[14] [通用文本中继处理：`relay/compatible_handler.go`](https://github.com/55gY/new-api-lite/blob/main/relay/compatible_handler.go)  
[15] [通道缓存与调度：`model/channel_cache.go`](https://github.com/55gY/new-api-lite/blob/main/model/channel_cache.go)
