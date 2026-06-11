---
name: shadcn-ui
description: >-
  为助手提供项目感知的 shadcn/ui 上下文：components.json、组合模式、CLI、
  registries、主题和 MCP。用于处理 web/default UI、shadcn 组件或 presets。
  概览与 https://ui.shadcn.com/docs/skills.md 保持一致；完整上游 skill 文本
  已内置在 vendor/shadcn/ 下。
---

<!-- Canonical overview: https://ui.shadcn.com/docs/skills.md -->

# Skills（shadcn/ui）

Skills 为 AI 助手提供关于 shadcn/ui 的项目感知上下文。使用后，助手会知道如何基于你的项目使用正确的 API 和模式来查找、安装、组合和自定义组件。

例如，你可以这样询问：

- _"Add a login form with email and password fields."_
- _"Create a settings page with a form for updating profile information."_
- _"Build a dashboard with a sidebar, stats cards, and a data table."_
- _"Switch to --preset [CODE]"_
- _"Can you add a hero from @tailark?"_

该 skill 会读取你项目中的 `components.json`，并提供 framework、aliases、已安装组件、图标库和 base library，以便首次尝试就能生成正确代码。

---

## 安装（生态系统 vs 本仓库）

来自 [Skills — shadcn/ui](https://ui.shadcn.com/docs/skills.md) 的官方安装方式：

```bash
npx skills add shadcn/ui
```

这会在 `skills` CLI 可用的位置安装该 skill。**本仓库**在 `.agents/skills/shadcn-ui/` 下保留相同意图（此处为概览 + [`vendor/shadcn/`](./vendor/shadcn/) 中的**内置**上游文档），并从前端应用根目录运行 shadcn CLI：

```bash
cd web/default && bunx shadcn@latest info --json
```

可在 [skills.sh](https://skills.sh) 了解更多关于 skills 的信息。

---

## 包含内容（以及位置）

### 项目上下文

运行 **`shadcn info --json`**（此处为：`cd web/default && bunx shadcn@latest info --json`）以获取 framework、Tailwind 版本、aliases、base（`radix` | `base`）、图标库、已安装组件和解析后的路径。

### CLI 命令

完整命令参考（内置）：[`vendor/shadcn/cli.md`](./vendor/shadcn/cli.md)。

### 主题和自定义

内置文档：[`vendor/shadcn/customization.md`](./vendor/shadcn/customization.md)。在线文档：[Theming](https://ui.shadcn.com/docs/theming)。

### Registry 编写

未在 vendor 树中作为单独文件重复保存；请参阅 [Registry](https://ui.shadcn.com/docs/registry) 以及 [`vendor/shadcn/cli.md`](./vendor/shadcn/cli.md) 中的 `build`。

### MCP server

内置文档：[`vendor/shadcn/mcp.md`](./vendor/shadcn/mcp.md)。在线文档：[MCP Server](https://ui.shadcn.com/docs/mcp)。

---

## 工作方式

1. **项目检测** — 当 `components.json` 存在时适用（此处为：`web/default/components.json`）。
2. **上下文注入** — 将 `shadcn info --json` 作为 imports 和 APIs 的事实依据。
3. **模式强制** — 遵循 [`vendor/shadcn/SKILL.md`](./vendor/shadcn/SKILL.md) 和 [`vendor/shadcn/rules/`](./vendor/shadcn/rules/) 中的规则。
4. **组件发现** — `shadcn docs`、`shadcn search`、MCP 或 registries — 参见内置的 SKILL + MCP 文档。

---

## 了解更多（web）

- [CLI](https://ui.shadcn.com/docs/cli) — 补充 [`vendor/shadcn/cli.md`](./vendor/shadcn/cli.md)
- [Theming](https://ui.shadcn.com/docs/theming)
- [Registry](https://ui.shadcn.com/docs/registry)
- [skills.sh](https://skills.sh)

---

## 内置上游包（深层规则）

来自 [shadcn-ui/ui `skills/shadcn`](https://github.com/shadcn-ui/ui/tree/main/skills/shadcn) 的快照；修订说明见 [`vendor/shadcn/UPSTREAM.txt`](./vendor/shadcn/UPSTREAM.txt)。

| Doc | Path |
| --- | --- |
| 完整官方 skill 正文 | [`vendor/shadcn/SKILL.md`](./vendor/shadcn/SKILL.md) |
| CLI 参考 | [`vendor/shadcn/cli.md`](./vendor/shadcn/cli.md) |
| 主题 / 自定义 | [`vendor/shadcn/customization.md`](./vendor/shadcn/customization.md) |
| MCP | [`vendor/shadcn/mcp.md`](./vendor/shadcn/mcp.md) |
| 表单 | [`vendor/shadcn/rules/forms.md`](./vendor/shadcn/rules/forms.md) |
| 组合 | [`vendor/shadcn/rules/composition.md`](./vendor/shadcn/rules/composition.md) |
| 图标 | [`vendor/shadcn/rules/icons.md`](./vendor/shadcn/rules/icons.md) |
| 样式 | [`vendor/shadcn/rules/styling.md`](./vendor/shadcn/rules/styling.md) |
| Base vs Radix | [`vendor/shadcn/rules/base-vs-radix.md`](./vendor/shadcn/rules/base-vs-radix.md) |
**`vendor/shadcn/SKILL.md`** 以获取完整上游工作流、模式和 CLI 快速参考。验证具体 markup 时使用 **`vendor/shadcn/rules/*.md`**。
