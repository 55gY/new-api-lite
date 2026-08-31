# new-api-lite

`new-api-lite` 是基于 new-api 精简维护的 AI API 网关。项目仅支持 SQLite，并已移除充值、计费、额度、倍率、音频、视频及异步任务等功能；仍保留模型代理、用户认证、API Token、渠道管理、日志和 tokens 使用统计。

## Docker 快速启动

如果主机已安装 Docker，最快的一键安装方式是直接运行 GitHub 上的管理脚本：

```bash
# GitHub 直连，一键安装
bash <(curl -fsSL https://raw.githubusercontent.com/55gY/new-api-lite/main/install.sh) install

# GitHub 无法直连时：按顺序尝试 Raw 加速节点
downloaded=0
raw_mirrors="${NEW_API_LITE_GITHUB_MIRRORS:-https://gh-proxy.com/ https://gh-proxy.org/ https://v6.gh-proxy.org/}"
for mirror in $raw_mirrors; do
  if script="$(curl -fsSL --connect-timeout 8 --max-time 30 "${mirror}https://raw.githubusercontent.com/55gY/new-api-lite/main/install.sh")" \
    && printf '%s' "$script" | NEW_API_LITE_MIRROR_MODE=mirror bash -s -- install; then
    downloaded=1
    break
  fi
done
if [ "$downloaded" -ne 1 ]; then
  echo '所有 GitHub Raw 加速节点均不可用，请检查网络或改用直连命令。' >&2
  exit 1
fi
```

直连命令会直接执行安装；脚本会自动检查系统架构和 Docker，并在需要时安装 Docker。若希望使用菜单，下载脚本后直接执行 `./install.sh`。镜像加速命令会先从多个 GitHub Raw 节点获取同一份脚本，再以 `mirror` 模式尝试多个 GHCR 镜像节点。脚本下载失败时会直接退出，不会把代理返回的错误页面交给 Bash 执行。也可以先下载脚本再运行，以便审阅或重复使用：

```bash
curl -fLso install.sh https://raw.githubusercontent.com/55gY/new-api-lite/main/install.sh
chmod +x ./install.sh
./install.sh install
```

以下命令会在**当前目录**创建持久化数据目录，并启动 `latest` 多架构镜像；Docker 会自动拉取与当前主机架构匹配的变体。`--pull=always` 会在启动时拉取标签对应的最新镜像；`--restart=unless-stopped` 会在 Docker 服务重启后自动恢复，但管理员主动停止容器后不会擅自再次启动。

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
  ghcr.io/55gy/new-api-lite:latest
```

启动完成后可访问 `http://localhost:3000`。如服务器通过防火墙或云安全组对外提供服务，还需要自行放通 TCP `3000` 端口。

## Docker 管理面板

仓库根目录提供 [`install.sh`](install.sh) 原生 Docker 管理脚本。脚本会在启动时检查 Docker 服务状态，并默认将数据保存到**执行命令所在目录**的 `./data`。首次一键安装可执行：

```bash
chmod +x ./install.sh
./install.sh install
```

如需打开交互式管理面板，直接执行 `./install.sh`。

项目的 `ghcr.io/55gy/new-api-lite:latest` 为多架构镜像，Docker 会按当前主机架构自动拉取 ARM64 或 AMD64 变体。脚本会显示当前系统发行版与 CPU 架构，并检查 Docker 客户端、守护进程和当前用户访问权限。若在 `install` 或 `update` 时未检测到 Docker，脚本会使用当前系统可用的软件包管理器自动安装并尝试启动 Docker；目前支持 `apt-get`、`dnf`、`yum` 与 `apk`。当当前用户没有 Docker 组权限时，脚本会在本次运行中使用 `sudo`，不会静默修改用户组。交互面板提供安装/启动、拉取镜像并重建容器、状态、日志、停止、重启、卸载及数据删除功能。容器更新和卸载均会明确保留 `data`；删除数据需输入 `DELETE` 二次确认。

脚本支持 `direct`、`mirror` 和 `auto` 三种模式。`direct` 只拉取原始 GHCR；`mirror` 按顺序尝试 `gh-proxy.org/docker/`、`gh-proxy.com/docker/` 等 GHCR 加速节点，最后回退到原始 GHCR；`auto` 先尝试原始 GHCR，再尝试加速节点。每个成功拉取的代理镜像引用会直接用于创建容器，因此容器运行阶段不会再次直连原始 GHCR。可通过 `NEW_API_LITE_GITHUB_MIRRORS` 和 `NEW_API_LITE_DOCKER_MIRRORS` 自定义空格分隔的节点列表。也可使用非交互命令，适合写入自己的运维脚本：

```bash
./install.sh install
./install.sh update
./install.sh check
./install.sh status
./install.sh logs
./install.sh stop
./install.sh restart
./install.sh uninstall
```

如需调整数据目录、端口、镜像或时区，可在执行时覆盖默认值。例如：

```bash
NEW_API_LITE_DATA_DIR=/srv/new-api/data \
NEW_API_LITE_PORT=3000 \
NEW_API_LITE_TZ=Asia/Shanghai \
./install.sh update

# 使用 GHCR 镜像加速节点更新，并在失败时回退直连
NEW_API_LITE_MIRROR_MODE=mirror ./install.sh update

# 自定义 Raw 脚本加速节点（空格分隔）
export NEW_API_LITE_GITHUB_MIRRORS='https://gh-proxy.com/ https://gh-proxy.org/'
for mirror in $NEW_API_LITE_GITHUB_MIRRORS; do
  if script="$(curl -fsSL --connect-timeout 8 --max-time 30 "${mirror}https://raw.githubusercontent.com/55gY/new-api-lite/main/install.sh")"; then
    printf '%s' "$script" | NEW_API_LITE_MIRROR_MODE=mirror bash -s -- install
    break
  fi
done

# 自定义 GHCR 容器加速节点（空格分隔）
NEW_API_LITE_DOCKER_MIRRORS='gh-proxy.org/docker/ gh-proxy.com/docker/' \
  NEW_API_LITE_MIRROR_MODE=mirror ./install.sh update

# 如只检查系统与 Docker 状态，不执行安装或更新
./install.sh check
```

| 常用操作 | 命令 |
| --- | --- |
| 查看容器状态 | `docker ps --filter name=^/new-api$` |
| 实时查看日志 | `docker logs --follow --tail 200 new-api` |
| 停止服务 | `docker stop new-api` |
| 再次启动 | `docker start new-api` |
| 更新至镜像标签的最新版本 | 执行下方「更新镜像」完整命令。 |

> 数据库和运行配置保存在宿主机的 `./data` 中。请在**计划执行命令的目录**运行上述命令；使用 `"$(pwd)/data"` 而不是相对路径，可避免 Docker 解析工作目录差异导致的数据卷挂载错误。

## 更新镜像

推荐使用 `./install.sh update` 完成直连更新；网络无法访问 GHCR 时使用 `NEW_API_LITE_MIRROR_MODE=mirror ./install.sh update`，脚本会轮询 GHCR 加速节点，成功后使用代理镜像引用重建容器，全部节点失败时再回退原始 GHCR。该命令同样会先检查 Docker；若 Docker 尚未安装，会先按系统的软件包管理器自动安装并尝试启动服务。若需禁止自动安装，可设置 `NEW_API_LITE_AUTO_INSTALL_DOCKER=0`。若不使用管理脚本，可在项目数据目录所在的同一目录执行下列原生 Docker 命令。该流程会先拉取镜像，随后仅在 `new-api` 容器存在时停止并删除它，最后使用原有宿主机 `./data` 数据目录创建新容器；**不会删除 SQLite 数据库或程序配置**。

```bash
set -e

image='ghcr.io/55gy/new-api-lite:latest'
data_dir="$(pwd)/data"

# 直连 GHCR；镜像模式请使用上方 install.sh update，它会为容器保留成功的代理镜像引用。
docker pull "$image"

if docker container inspect new-api >/dev/null 2>&1; then
  docker rm --force new-api
fi

mkdir -p "$data_dir"
docker run \
  --name new-api \
  --detach \
  --init \
  --restart=unless-stopped \
  --publish 3000:3000 \
  --env TZ=Asia/Shanghai \
  --volume "$data_dir:/data" \
  "$image"
```

更新完成后可用 `docker logs --follow --tail 200 new-api` 检查启动日志。若容器端口、时区或数据目录曾按你的环境做过调整，请同步修改更新命令中的对应参数，避免新容器回到默认值。

## 卸载

下面的命令仅删除本项目容器，**保留 `./data` 数据目录**，因此之后重新执行启动命令即可继续使用既有配置和 SQLite 数据。

```bash
docker rm --force new-api
```

如需同时删除本项目已下载的镜像，可额外执行：

```bash
docker image rm ghcr.io/55gy/new-api-lite:latest
```

如需彻底删除全部本地数据，请先确认当前目录中的 `./data` 确实是本项目的数据目录，再手动执行下列不可恢复操作：

```bash
rm -rf ./data
```

> 不建议在本项目的卸载命令中使用 `docker container prune` 或 `docker image prune -a`。这两条命令会清理主机上**其他项目**的已停止容器或未被引用镜像，可能造成无关服务的数据和部署资产丢失。

## 备份与还原配置

Root 管理员可在「系统设置 → 配置备份」中选择当前有效配置类别导出 JSON，并在需要时选择类别还原。账户、API Token、2FA、日志、tokens 使用统计、运行时性能数据和 SQLite 数据库文件均不会被导出或还原。敏感凭据与渠道配置默认排除，并需要明确确认；渠道还原会整体替换现有渠道及其能力配置。

更多项目约束和开发规则请参阅 [AGENTS.md](AGENTS.md)。
