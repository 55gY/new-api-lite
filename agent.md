# 本次调整记录

## 变更主题

开放全局 API 限流相关配置，并将其默认值改为关闭。

## 实际变更

后端新增 `GlobalApiRateLimitEnable`、`GlobalApiRateLimitNum`、`GlobalApiRateLimitDuration` 三个系统选项，支持从数据库读取并在保存后动态更新内存配置。`GLOBAL_API_RATE_LIMIT_ENABLE` 在未显式设置环境变量时默认值由 `true` 改为 `false`。前端速率限制设置页面新增全局 API 限流开关、每周期请求次数、周期秒数和保存按钮。

本次未改变模型请求限流、全局 Web 限流和关键接口限流，也未引入签到、计费、额度、倍率、音视频或异步任务功能。

## 验证

- `gofmt -w common/init.go model/option.go`
- `go test ./middleware ./model ./controller`：通过。
- `git diff --check`：通过。
- 前端构建：当前环境未安装 Bun，且 `web/classic/node_modules` 不存在，因此未执行生产构建；不得将其记录为已通过。

## 提交说明

`功能：开放全局 API 限流系统设置`

## 同步状态

代码变更、`plan.md`、`.plan/20260906-01.md` 和本记录文件应在同一中文提交中提交，并推送到 GitHub 远端后核验远端状态。
