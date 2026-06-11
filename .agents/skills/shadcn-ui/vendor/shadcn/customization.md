# 自定义与主题

组件引用语义化 CSS 变量 tokens。修改这些变量即可改变所有组件。

## 目录

- 工作原理（CSS 变量 → Tailwind 工具类 → 组件）
- 颜色变量与 OKLCH 格式
- 暗色模式设置
- 更改主题（presets、CSS 变量）
- 添加自定义颜色（Tailwind v3 和 v4）
- 边框圆角
- 自定义组件（variants、className、wrappers）
- 检查更新

---

## 工作原理

1. CSS 变量定义在 `:root`（亮色）和 `.dark`（暗色模式）中。
2. Tailwind 将它们映射为工具类：`bg-primary`、`text-muted-foreground` 等。
3. 组件使用这些工具类——修改变量会改变所有引用它的组件。

---

## 颜色变量

每种颜色都遵循 `name` / `name-foreground` 约定。基础变量用于背景，`-foreground` 用于该背景上的文本/图标。

| 变量 | 用途 |
| -------------------------------------------- | -------------------------------- |
| `--background` / `--foreground` | 页面背景和默认文本 |
| `--card` / `--card-foreground` | Card 表面 |
| `--primary` / `--primary-foreground` | 主要按钮和操作 |
| `--secondary` / `--secondary-foreground` | 次要操作 |
| `--muted` / `--muted-foreground` | 弱化/禁用状态 |
| `--accent` / `--accent-foreground` | 悬停和强调状态 |
| `--destructive` / `--destructive-foreground` | 错误和破坏性操作 |
| `--border` | 默认边框颜色 |
| `--input` | 表单输入边框 |
| `--ring` | 焦点环颜色 |
| `--chart-1` 到 `--chart-5` | 图表/数据可视化 |
| `--sidebar-*` | Sidebar 专用颜色 |
| `--surface` / `--surface-foreground` | 次级表面 |

颜色使用 OKLCH：`--primary: oklch(0.205 0 0)`，其中值依次为亮度（0–1）、色度（0 = 灰色）和色相（0–360）。

---

## 暗色模式

通过根元素上的 `.dark` 基于 class 切换。在 Next.js 中使用 `next-themes`：

```tsx
import { ThemeProvider } from "next-themes"

<ThemeProvider attribute="class" defaultTheme="system" enableSystem>
  {children}
</ThemeProvider>
```

---

## 更改主题

```bash
# 应用来自 ui.shadcn.com 的 preset code。
npx shadcn@latest apply --preset a2r6bw

# 位置简写也可用。
npx shadcn@latest apply a2r6bw

# 切换到命名 preset 并覆盖现有组件。
npx shadcn@latest apply --preset nova

# 改为保留现有组件。
npx shadcn@latest init --preset nova --force --no-reinstall

# 使用自定义主题 URL。
npx shadcn@latest apply --preset "https://ui.shadcn.com/init?base=radix&style=nova&theme=blue&..."
```

也可以直接在 `globals.css` 中编辑 CSS 变量。

---

## 添加自定义颜色

将变量添加到 `npx shadcn@latest info` 中 `tailwindCssFile` 指向的文件（通常是 `globals.css`）。绝不要为此新建 CSS 文件。

```css
/* 1. 在全局 CSS 文件中定义。 */
:root {
  --warning: oklch(0.84 0.16 84);
  --warning-foreground: oklch(0.28 0.07 46);
}
.dark {
  --warning: oklch(0.41 0.11 46);
  --warning-foreground: oklch(0.99 0.02 95);
}
```

```css
/* 2a. 使用 Tailwind v4 注册（@theme inline）。 */
@theme inline {
  --color-warning: var(--warning);
  --color-warning-foreground: var(--warning-foreground);
}
```

当 `tailwindVersion` 为 `"v3"`（通过 `npx shadcn@latest info` 检查）时，改为在 `tailwind.config.js` 中注册：

```js
// 2b. 使用 Tailwind v3 注册（tailwind.config.js）。
module.exports = {
  theme: {
    extend: {
      colors: {
        warning: "oklch(var(--warning) / <alpha-value>)",
        "warning-foreground":
          "oklch(var(--warning-foreground) / <alpha-value>)",
      },
    },
  },
}
```

```tsx
// 3. 在组件中使用。
<div className="bg-warning text-warning-foreground">Warning</div>
```

---

## 边框圆角

`--radius` 全局控制边框圆角。组件会从它派生值（`rounded-lg` = `var(--radius)`，`rounded-md` = `calc(var(--radius) - 2px)`）。

---

## 自定义组件

另见：[rules/styling.md](./rules/styling.md)，其中包含错误/正确示例。

按以下顺序优先采用：

### 1. 内置 variants

```tsx
<Button variant="outline" size="sm">
  Click
</Button>
```

### 2. 通过 `className` 使用 Tailwind classes

```tsx
<Card className="mx-auto max-w-md">...</Card>
```

### 3. 添加新的 variant

编辑组件源码，通过 `cva` 添加 variant：

```tsx
// components/ui/button.tsx
warning: "bg-warning text-warning-foreground hover:bg-warning/90",
```

### 4. Wrapper 组件

将 shadcn/ui primitives 组合成更高层组件：

```tsx
export function ConfirmDialog({ title, description, onConfirm, children }) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>{children}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>Confirm</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
```

---

## 检查更新

```bash
npx shadcn@latest add button --diff
```

若要在更新前精确预览会发生的更改，请使用 `--dry-run` 和 `--diff`：

```bash
npx shadcn@latest add button --dry-run        # 查看所有受影响文件
npx shadcn@latest add button --diff button.tsx # 查看特定文件的 diff
```

完整智能合并工作流见 [SKILL.md 中的更新组件](./SKILL.md#updating-components)。
