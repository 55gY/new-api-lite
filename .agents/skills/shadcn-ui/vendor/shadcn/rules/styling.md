# 样式与自定义

主题、CSS 变量和添加自定义颜色请参见 [customization.md](../customization.md)。

## 目录

- 语义化颜色
- 优先使用内置 variants
- className 仅用于布局
- 不使用 space-x-* / space-y-*
- 宽高相等时优先使用 size-*
- 优先使用 truncate 简写
- 不手动覆盖 dark: 颜色
- 使用 cn() 处理条件 classes
- 不在 overlay 组件上手动设置 z-index

---

## 语义化颜色

**错误：**

```tsx
<div className="bg-blue-500 text-white">
  <p className="text-gray-600">Secondary text</p>
</div>
```

**正确：**

```tsx
<div className="bg-primary text-primary-foreground">
  <p className="text-muted-foreground">Secondary text</p>
</div>
```

---

## 状态/状态指示器不要使用原始颜色值

对于正向、负向或状态指示器，使用 Badge variants、`text-destructive` 这类语义化 tokens，或定义自定义 CSS 变量——不要直接使用原始 Tailwind 颜色。

**错误：**

```tsx
<span className="text-emerald-600">+20.1%</span>
<span className="text-green-500">Active</span>
<span className="text-red-600">-3.2%</span>
```

**正确：**

```tsx
<Badge variant="secondary">+20.1%</Badge>
<Badge>Active</Badge>
<span className="text-destructive">-3.2%</span>
```

如果需要语义化 token 中不存在的成功/正向颜色，请使用 Badge variant，或询问用户是否要向主题添加自定义 CSS 变量（见 [customization.md](../customization.md)）。

---

## 优先使用内置 variants

**错误：**

```tsx
<Button className="border border-input bg-transparent hover:bg-accent">
  Click me
</Button>
```

**正确：**

```tsx
<Button variant="outline">Click me</Button>
```

---

## className 仅用于布局

使用 `className` 做布局（例如 `max-w-md`、`mx-auto`、`mt-4`），**不要**用它覆盖组件颜色或排版。要修改颜色，请使用语义化 tokens、内置 variants 或 CSS 变量。

**错误：**

```tsx
<Card className="bg-blue-100 text-blue-900 font-bold">
  <CardContent>Dashboard</CardContent>
</Card>
```

**正确：**

```tsx
<Card className="max-w-md mx-auto">
  <CardContent>Dashboard</CardContent>
</Card>
```

自定义组件外观时，按以下顺序优先选择：
1. **内置 variants** — `variant="outline"`、`variant="destructive"` 等。
2. **语义化颜色 tokens** — `bg-primary`、`text-muted-foreground`。
3. **CSS 变量** — 在全局 CSS 文件中定义自定义颜色（见 [customization.md](../customization.md)）。

---

## 不使用 space-x-* / space-y-*

改用 `gap-*`。`space-y-4` → `flex flex-col gap-4`。`space-x-2` → `flex gap-2`。

```tsx
<div className="flex flex-col gap-4">
  <Input />
  <Input />
  <Button>Submit</Button>
</div>
```

---

## 宽高相等时优先使用 size-*

使用 `size-10`，不要使用 `w-10 h-10`。适用于图标、头像、骨架屏等。

---

## 优先使用 truncate 简写

使用 `truncate`，不要使用 `overflow-hidden text-ellipsis whitespace-nowrap`。

---

## 不手动覆盖 dark: 颜色

使用语义化 tokens——它们会通过 CSS 变量处理亮/暗模式。使用 `bg-background text-foreground`，不要使用 `bg-white dark:bg-gray-950`。

---

## 使用 cn() 处理条件 classes

使用项目中的 `cn()` 工具处理条件或合并后的 class names。不要在 className 字符串中手写三元表达式。

**错误：**

```tsx
<div className={`flex items-center ${isActive ? "bg-primary text-primary-foreground" : "bg-muted"}`}>
```

**正确：**

```tsx
import { cn } from "@/lib/utils"

<div className={cn("flex items-center", isActive ? "bg-primary text-primary-foreground" : "bg-muted")}>
```

---

## 不在 overlay 组件上手动设置 z-index

`Dialog`、`Sheet`、`Drawer`、`AlertDialog`、`DropdownMenu`、`Popover`、`Tooltip`、`HoverCard` 会自行处理层叠。绝不要添加 `z-50` 或 `z-[999]`。
