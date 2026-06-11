# shadcn MCP 服务器

CLI 包含一个 MCP 服务器，可让 AI 助手从 registries 中搜索、浏览、查看和安装组件。

---

## 设置

```bash
shadcn mcp        # start the MCP server (stdio)
shadcn mcp init   # write config for your editor
```

编辑器配置文件：

| Editor | Config file |
|--------|------------|
| Claude Code | `.mcp.json` |
| Cursor | `.cursor/mcp.json` |
| VS Code | `.vscode/mcp.json` |
| OpenCode | `opencode.json` |
| Codex | `~/.codex/config.toml` (manual) |

---

## 工具

> **提示：** MCP 工具处理 registry 操作（search、view、install）。对于项目配置（aliases、framework、Tailwind 版本），请使用 `npx shadcn@latest info` — 没有对应的 MCP 等价工具。

### `shadcn:get_project_registries`

从 `components.json` 返回 registry 名称。如果不存在 `components.json`，则报错。

**输入：** 无

### `shadcn:list_items_in_registries`

列出一个或多个 registries 中的所有 items。

**输入：** `registries` (string[]), `limit` (number, optional), `offset` (number, optional)

### `shadcn:search_items_in_registries`

跨 registries 进行模糊搜索。

**输入：** `registries` (string[]), `query` (string), `limit` (number, optional), `offset` (number, optional)

### `shadcn:view_items_in_registries`

查看 item 详情，包括完整文件内容。

**输入：** `items` (string[]) — 例如 `["@shadcn/button", "@shadcn/card"]`

### `shadcn:get_item_examples_from_registries`

查找带源代码的用法示例和 demos。

**输入：** `registries` (string[]), `query` (string) — 例如 `"accordion-demo"`, `"button example"`

### `shadcn:get_add_command_for_items`

返回 CLI 安装命令。

**输入：** `items` (string[]) — 例如 `["@shadcn/button"]`

### `shadcn:get_audit_checklist`

返回用于验证组件（imports、deps、lint、TypeScript）的 checklist。

**输入：** 无

---

## 配置 Registries

Registries 在 `components.json` 中设置。`@shadcn` registry 始终为内置。

```json
{
  "registries": {
    "@acme": "https://acme.com/r/{name}.json",
    "@private": {
      "url": "https://private.com/r/{name}.json",
      "headers": { "Authorization": "Bearer ${MY_TOKEN}" }
    }
  }
}
```

- 名称必须以 `@` 开头。
- URLs 必须包含 `{name}`。
- `${VAR}` 引用会从环境变量解析。

社区 registry 索引：`https://ui.shadcn.com/r/registries.json`
