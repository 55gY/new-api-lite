---
name: classic-to-default-sync
description: "检查给定提交中的 web/classic 修改，并将所有功能/修复同步到 web/default。适用于用户提供提交 ID，并希望审计 web/default 是否已具备与 web/classic 相同的功能、移植缺失功能、改进次优实现、修复错误以及移除冗余代码的场景。触发短语包括：\"/classic-to-default-sync <hash>\"、\"classic-to-default-sync <hash>\"、\"sync classic to default\"、\"port from classic\"、\"compare classic commit\"、\"classic 和 default 对比\"、\"把这次 classic 的修改同步到 default\"、\"查看这次提交 classic 中的修改并同步\"，或任何同时提供提交哈希并表达 classic/default 对比意图的请求。"
---

# Classic-to-Default 同步

给定一个 **commit ID**，审计所有 `web/classic` 修改，并确保 `web/default` 以尽可能最佳的实现达到功能对等。

## 输入

用户必须提供一个 `<commit-id>`。

## 工作流

### 步骤 1 — 提取 classic diff

```bash
git show <commit-id> -- web/classic
```

阅读 `web/classic` 中每个被修改的文件。识别**逻辑变更**（新功能、UI/UX 改进、错误修复、配置调整、移除死代码等），而不只是逐行 diff。

### 步骤 2 — 映射到 default 对应文件

对于步骤 1 中发现的每项逻辑变更，在 `web/default/src/` 中定位等价文件。按需使用 Glob/Grep/SemanticSearch。注意：

- `web/classic` 使用 **React 18 + Vite + Semi Design**
- `web/default` 使用 **React 19 + Rsbuild + Base UI + Tailwind CSS**
- 组件名称、文件路径和 API 形态可能不同；应按**功能**匹配，而不是按文件名匹配。

### 步骤 3 — 分流每项变更

将每项逻辑变更分类为以下之一：

| 状态 | 含义 |
|--------|---------|
| ✅ 已存在且最优 | 无需操作 |
| ⚠️ 已存在但不够理想 | 改进：逻辑、布局、样式或代码质量 |
| ❌ 缺失 | 使用 default 技术栈从零实现 |

### 步骤 4 — 实现

对于每个 **⚠️** 或 **❌** 项：

1. 编辑前先**阅读 `web/default` 中的目标文件**（项目约定要求）。
2. 使用 `web/default` 约定实现：
   - React 19 模式（hooks、Suspense 等）
   - 适用时使用 Base UI primitives
   - 使用 Tailwind CSS 编写样式（禁止内联样式或 Semi Design imports）
   - 所有用户可见字符串使用 `useTranslation()` + `t('English key')`
   - TypeScript — 显式类型，不使用 `any`
   - 不保留死代码，不添加冗余注释
3. 如果触及 relay 相关 TS 类型，遵循 **Rule 6**（可选 relay DTO 使用指针类型）。
4. 编辑后，在修改文件上运行 `ReadLints`，并修复任何新引入的 lint 错误。

### 步骤 5 — i18n

如果添加了任何新的用户可见字符串，运行 i18n 同步：

```bash
cd web/default && bun run i18n:sync
```

然后按照 **i18n-translate** skill，为所有受支持的语言环境（en、zh、fr、ja、ru、vi）添加缺失翻译。

### 步骤 6 — 报告

用简洁表格汇总工作：

| # | 变更（来自 classic 提交） | 状态 | 已采取操作 |
|---|------------------------------|--------|--------------|
| 1 | … | ✅ / ⚠️ / ❌ | 无 / 已改进 / 已实现 |

如果每一项都是 ✅ 且无需操作，直接回复：**"已完成 — web/default 已具备此次提交的所有功能，且实现质量良好，无需修改。"**

## 质量标准

- 没有未使用的 imports、变量或组件
- 不遗留被注释掉的代码
- 命名与周围 `web/default` 代码保持一致
- 所有交互元素均具备可访问性（键盘导航，以及在 Radix 未自动提供时添加 ARIA labels）
- 无回归：不得破坏 `web/default` 中的现有行为
