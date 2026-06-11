---
name: i18n-translate
description: >-
  完成并维护本项目的前端 i18n 翻译。涵盖查找缺失翻译键、检测未翻译条目，并为所有受支持的语言环境
  （en、zh、fr、ja、ru、vi）添加翻译。适用于用户要求添加翻译、修复 i18n、补全缺失翻译，
  或新的 UI 文本需要国际化的场景。
---
# 前端 i18n 翻译工作流

## 概述

- 语言环境文件：`web/classic/src/i18n/locales/{en,zh,fr,ja,ru,vi}.json`
- 格式：`"translation"` 键下的扁平 JSON，键为英文源字符串。
- 基础语言环境：`en.json`（大多数键），回退语言环境：`zh`（中文）。
- 同步脚本：`bun run i18n:sync`（从 `web/classic/` 运行）。
- 所有 `t()` 调用都必须在每个语言环境文件中有对应键。

## 工作流

### 步骤 1：运行同步并阅读报告

```bash
cd web/classic && bun run i18n:sync
```

阅读 `web/classic/src/i18n/locales/_reports/_sync-report.json`，查看每个语言环境的状态（`missingCount`、`extrasCount`、`untranslatedCount`）。

### 步骤 2：查找缺失键（代码中已使用但语言环境文件中不存在）

创建并运行 `web/classic/scripts/find-missing-keys.mjs`：

```javascript
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const SRC_DIR = path.resolve('src')

const en = JSON.parse(await fs.readFile(path.join(LOCALES_DIR, 'en.json'), 'utf8'))
const enKeys = new Set(Object.keys(en.translation))

const tCallRegex = /\bt\(\s*['"`]([^'"`\n]+?)['"`]\s*[,)]/g
const tCallMultilineRegex = /\bt\(\s*['"`]([^'"`]+?)['"`]\s*\)/g

async function walkDir(dir) {
  const files = []
  const entries = await fs.readdir(dir, { withFileTypes: true })
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (['node_modules', '.git', 'locales', '_reports', '_extras'].includes(entry.name)) continue
      files.push(...(await walkDir(fullPath)))
    } else if (/\.(tsx?|jsx?)$/.test(entry.name)) {
      files.push(fullPath)
    }
  }
  return files
}

const files = await walkDir(SRC_DIR)
const missingKeys = new Map()

for (const file of files) {
  const content = await fs.readFile(file, 'utf8')
  const relPath = path.relative(SRC_DIR, file)
  for (const regex of [tCallRegex, tCallMultilineRegex]) {
    regex.lastIndex = 0
    let match
    while ((match = regex.exec(content)) !== null) {
      const key = match[1]
      if (key.startsWith('{{') || key.includes('${')) continue
      if (!enKeys.has(key)) {
        if (!missingKeys.has(key)) missingKeys.set(key, [])
        missingKeys.get(key).push(relPath)
      }
    }
  }
}

if (missingKeys.size === 0) {
  console.log('All t() keys found in en.json!')
} else {
  console.log(`Found ${missingKeys.size} missing keys:\n`)
  for (const [key, files] of [...missingKeys.entries()].sort(([a], [b]) => a.localeCompare(b))) {
    console.log(`  "${key}"`)
    for (const f of [...new Set(files)]) console.log(`    -> ${f}`)
  }
}
```

### 步骤 3：查找未翻译条目（值等于英文）

创建并运行 `web/classic/scripts/find-untranslated.mjs`：

```javascript
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const en = JSON.parse(await fs.readFile(path.join(LOCALES_DIR, 'en.json'), 'utf8'))
const enTrans = en.translation

// Brand names, URLs, technical terms — skip these
const skipPatterns = [
  /^https?:\/\//, /^smtp\./, /^socks5:/, /^name@/, /^noreply@/,
  /^org-/, /^price_/, /^whsec_/, /^edit_this$/, /^my-status$/,
  /^_copy$/, /^gpt-/, /^checkout\./, /^footer\./, /^\[?\{/,
  /^"default/, /^\/status\//, /^\/your\//, /^example\.com/,
  /^AZURE_/, /^AccessKey/, /^OAuth/, /^Client /, /^Webhook URL/,
  /^API URL$/, /^Well-Known/, /^Worker URL$/, /^Uptime Kuma/,
  /^New API/, /^Baidu V2$/, /^Zhipu V4$/, /^Quota:$/,
]

const brandNames = new Set([
  'AIGC2D','Anthropic','API2GPT','Claude','Cloudflare','Cohere','DeepSeek',
  'DoubaoVideo','FastGPT','Gemini','Jimeng','JustSong',
  'LingYiWanWu','Midjourney','MidjourneyPlus','MiniMax','Mistral',
  'MokaAI','Moonshot','NewAPI','OhMyGPT','Ollama','OpenAI','OpenAIMax',
  'OpenRouter','Perplexity','QuantumNous','Replicate','SiliconFlow',
  'Stripe','Submodel','SunoAPI','Tencent','Vertex AI','VolcEngine',
  'Xinference','Xunfei','AI Proxy','One API',
])

const locales = ['fr', 'ja', 'ru', 'zh', 'vi']

for (const locale of locales) {
  const locFile = JSON.parse(await fs.readFile(path.join(LOCALES_DIR, `${locale}.json`), 'utf8'))
  const locTrans = locFile.translation
  const untranslated = {}

  for (const [key, enVal] of Object.entries(enTrans)) {
    const locVal = locTrans[key]
    if (locVal === undefined || locVal !== enVal) continue
    if (brandNames.has(key)) continue
    if (skipPatterns.some(p => p.test(key))) continue
    if (typeof enVal === 'string' && enVal.length < 4) continue
    if (/[a-zA-Z]{3,}/.test(String(enVal))) untranslated[key] = enVal
  }

  const count = Object.keys(untranslated).length
  if (count > 0) {
    console.log(`\n=== ${locale} (${count} untranslated) ===`)
    for (const [k, v] of Object.entries(untranslated))
      console.log(`  ${JSON.stringify(k)}: ${JSON.stringify(v)}`)
  } else {
    console.log(`\n=== ${locale}: all translated ===`)
  }
}
```

### 步骤 4：添加翻译

使用以下结构创建 `web/classic/scripts/add-missing-keys.mjs`：

```javascript
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: { /* "key": "English value" */ },
  zh: { /* "key": "中文翻译" */ },
  fr: { /* "key": "Traduction française" */ },
  ja: { /* "key": "日本語翻訳" */ },
  ru: { /* "key": "Русский перевод" */ },
  vi: { /* "key": "Bản dịch tiếng Việt" */ },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.prototype.hasOwnProperty.call(json.translation, key)) {
        json.translation[key] = value
        count++
      } else if (json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalAdded += count
  }

  console.log(`\nTotal: ${totalAdded} translations applied`)
}

main().catch((err) => { console.error(err); process.exitCode = 1 })
```

用每个语言环境的实际翻译填充 `newKeys` 对象。

### 步骤 5：验证并清理

```bash
cd web/classic
node scripts/add-missing-keys.mjs   # apply translations
node scripts/find-missing-keys.mjs  # verify: should say "All t() keys found"
bun run i18n:sync                   # normalize file order
```

完成后删除临时脚本。

## 翻译指南

| 语言 | 代码 | 说明 |
|------|------|------|
| 英语 | en | 基础语言环境，key = value |
| 中文 | zh | 回退语言环境，必须完整 |
| 法语 | fr | 许多英语同源词有效（例如 “Configuration”） |
| 日语 | ja | 技术外来词使用片假名 |
| 俄语 | ru | 使用正式语体 |
| 越南语 | vi | 使用标准越南语 |

**保留为英文（不要翻译）：**

- 品牌/产品名称（OpenAI、Claude、Gemini 等）
- URLs 和电子邮件占位符
- 技术标识符（JSON keys、API paths、model names）
- 类代码字符串（`gpt-3.5-turbo`、`price_xxx` 等）

**始终翻译：**

- UI labels、按钮文本、错误消息、描述
- 时间单位（hours、minutes、months、years）
- 操作词（Move、Show、Delete 等）

## 关键规则

1. 所有脚本都从 `web/classic/` 目录运行。
2. 使用 `node scripts/xxx.mjs`（ESM 格式，带 top-level await）。
3. 写入语言环境文件时按字母顺序排序键。
4. 始终将运行 `bun run i18n:sync` 作为最后一步。
5. 完成后删除临时脚本。
6. 所有翻译中都必须保留键里的 `{{variable}}` 占位符。
