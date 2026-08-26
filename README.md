# new-api-lite

`new-api-lite` 是基于 new-api 精简维护的 AI API 网关。项目仅支持 SQLite，并已移除充值、计费、额度、倍率、音频、视频及异步任务等功能；仍保留模型代理、用户认证、API Token、渠道管理、日志和 tokens 使用统计。

## Docker 快速启动

以下命令会在**当前目录**创建持久化数据目录，并启动 `amd64` 镜像。`--pull=always` 会在启动时拉取标签对应的最新镜像；`--restart=unless-stopped` 会在 Docker 服务重启后自动恢复，但管理员主动停止容器后不会擅自再次启动。

```bash
mkdir -p ./data

docker run \
  --pull=always \
  --name new-api \
  --detach \
  --init \
  --restart=unless-stopped \
  --publish 3000:3000 \
  --env TZ=Asia/Shanghai \
  --volume "$(pwd)/data:/data" \
  ghcr.io/55gy/new-api-lite:latest-amd64
```

启动完成后可访问 `http://localhost:3000`。如服务器通过防火墙或云安全组对外提供服务，还需要自行放通 TCP `3000` 端口。

| 常用操作 | 命令 |
| --- | --- |
| 查看容器状态 | `docker ps --filter name=^/new-api$` |
| 实时查看日志 | `docker logs --follow --tail 200 new-api` |
| 停止服务 | `docker stop new-api` |
| 再次启动 | `docker start new-api` |
| 更新至镜像标签的最新版本 | `docker pull ghcr.io/55gy/new-api-lite:latest-amd64 && docker rm -f new-api`，然后重新执行上方启动命令 |

> 数据库和运行配置保存在宿主机的 `./data` 中。请在**计划执行命令的目录**运行上述命令；使用 `"$(pwd)/data"` 而不是相对路径，可避免 Docker 解析工作目录差异导致的数据卷挂载错误。

## 卸载

下面的命令仅删除本项目容器，**保留 `./data` 数据目录**，因此之后重新执行启动命令即可继续使用既有配置和 SQLite 数据。

```bash
docker rm --force new-api
```

如需同时删除本项目已下载的镜像，可额外执行：

```bash
docker image rm ghcr.io/55gy/new-api-lite:latest-amd64
```

如需彻底删除全部本地数据，请先确认当前目录中的 `./data` 确实是本项目的数据目录，再手动执行下列不可恢复操作：

```bash
rm -rf ./data
```

> 不建议在本项目的卸载命令中使用 `docker container prune` 或 `docker image prune -a`。这两条命令会清理主机上**其他项目**的已停止容器或未被引用镜像，可能造成无关服务的数据和部署资产丢失。

## 备份与还原配置

Root 管理员可在「系统设置 → 配置备份」中选择当前有效配置类别导出 JSON，并在需要时选择类别还原。账户、API Token、2FA、日志、tokens 使用统计、运行时性能数据和 SQLite 数据库文件均不会被导出或还原。敏感凭据与渠道配置默认排除，并需要明确确认；渠道还原会整体替换现有渠道及其能力配置。

更多项目约束和开发规则请参阅 [AGENTS.md](AGENTS.md)。
