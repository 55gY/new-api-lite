# 表单与输入

## 目录

- 表单使用 FieldGroup + Field
- InputGroup 需要 InputGroupInput/InputGroupTextarea
- 输入框内按钮使用 InputGroup + InputGroupAddon
- 选项集（2–7 个选项）使用 ToggleGroup
- 使用 FieldSet + FieldLegend 分组相关字段
- 字段校验与禁用状态

---

## 表单使用 FieldGroup + Field

始终使用 `FieldGroup` + `Field`——绝不要使用带 `space-y-*` 的原始 `div`：

```tsx
<FieldGroup>
  <Field>
    <FieldLabel htmlFor="email">Email</FieldLabel>
    <Input id="email" type="email" />
  </Field>
  <Field>
    <FieldLabel htmlFor="password">Password</FieldLabel>
    <Input id="password" type="password" />
  </Field>
</FieldGroup>
```

设置页使用 `Field orientation="horizontal"`。视觉隐藏的标签使用 `FieldLabel className="sr-only"`。

**选择表单控件：**

- 简单文本输入 → `Input`
- 带预定义选项的下拉框 → `Select`
- 可搜索下拉框 → `Combobox`
- 原生 HTML select（无 JS）→ `native-select`
- 布尔开关 → `Switch`（用于设置）或 `Checkbox`（用于表单）
- 从少量选项中单选 → `RadioGroup`
- 在 2–5 个选项间切换 → `ToggleGroup` + `ToggleGroupItem`
- OTP/验证码 → `InputOTP`
- 多行文本 → `Textarea`

---

## InputGroup 需要 InputGroupInput/InputGroupTextarea

绝不要在 `InputGroup` 内使用原始 `Input` 或 `Textarea`。

**错误：**

```tsx
<InputGroup>
  <Input placeholder="Search..." />
</InputGroup>
```

**正确：**

```tsx
import { InputGroup, InputGroupInput } from "@/components/ui/input-group"

<InputGroup>
  <InputGroupInput placeholder="Search..." />
</InputGroup>
```

---

## 输入框内按钮使用 InputGroup + InputGroupAddon

绝不要通过自定义定位把 `Button` 直接放在 `Input` 内部或旁边。

**错误：**

```tsx
<div className="relative">
  <Input placeholder="Search..." className="pr-10" />
  <Button className="absolute right-0 top-0" size="icon">
    <SearchIcon />
  </Button>
</div>
```

**正确：**

```tsx
import { InputGroup, InputGroupInput, InputGroupAddon } from "@/components/ui/input-group"

<InputGroup>
  <InputGroupInput placeholder="Search..." />
  <InputGroupAddon>
    <Button size="icon">
      <SearchIcon data-icon="inline-start" />
    </Button>
  </InputGroupAddon>
</InputGroup>
```

---

## 选项集（2–7 个选项）使用 ToggleGroup

不要手动循环带 active state 的 `Button` 组件。

**错误：**

```tsx
const [selected, setSelected] = useState("daily")

<div className="flex gap-2">
  {["daily", "weekly", "monthly"].map((option) => (
    <Button
      key={option}
      variant={selected === option ? "default" : "outline"}
      onClick={() => setSelected(option)}
    >
      {option}
    </Button>
  ))}
</div>
```

**正确：**

```tsx
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"

<ToggleGroup spacing={2}>
  <ToggleGroupItem value="daily">Daily</ToggleGroupItem>
  <ToggleGroupItem value="weekly">Weekly</ToggleGroupItem>
  <ToggleGroupItem value="monthly">Monthly</ToggleGroupItem>
</ToggleGroup>
```

带标签的切换组选项应与 `Field` 组合使用：

```tsx
<Field orientation="horizontal">
  <FieldTitle id="theme-label">Theme</FieldTitle>
  <ToggleGroup aria-labelledby="theme-label" spacing={2}>
    <ToggleGroupItem value="light">Light</ToggleGroupItem>
    <ToggleGroupItem value="dark">Dark</ToggleGroupItem>
    <ToggleGroupItem value="system">System</ToggleGroupItem>
  </ToggleGroup>
</Field>
```

> **注意：** `defaultValue` 和 `type`/`multiple` props 在 base 与 radix 中不同。见 [base-vs-radix.md](./base-vs-radix.md#togglegroup)。

---

## 使用 FieldSet + FieldLegend 分组相关字段

对相关的复选框、单选框或开关使用 `FieldSet` + `FieldLegend`——不要使用带标题的 `div`：

```tsx
<FieldSet>
  <FieldLegend variant="label">Preferences</FieldLegend>
  <FieldDescription>Select all that apply.</FieldDescription>
  <FieldGroup className="gap-3">
    <Field orientation="horizontal">
      <Checkbox id="dark" />
      <FieldLabel htmlFor="dark" className="font-normal">Dark mode</FieldLabel>
    </Field>
  </FieldGroup>
</FieldSet>
```

---

## 字段校验与禁用状态

两类属性都需要——`data-invalid`/`data-disabled` 用于设置字段（标签、描述）的样式，而 `aria-invalid`/`disabled` 用于设置控件样式。

```tsx
// 无效状态。
<Field data-invalid>
  <FieldLabel htmlFor="email">Email</FieldLabel>
  <Input id="email" aria-invalid />
  <FieldDescription>Invalid email address.</FieldDescription>
</Field>

// 禁用状态。
<Field data-disabled>
  <FieldLabel htmlFor="email">Email</FieldLabel>
  <Input id="email" disabled />
</Field>
```

适用于所有控件：`Input`、`Textarea`、`Select`、`Checkbox`、`RadioGroupItem`、`Switch`、`Slider`、`NativeSelect`、`InputOTP`。
