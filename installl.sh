#!/usr/bin/env bash
# new-api-lite Docker 管理面板。
# 默认把数据保存在执行命令所在目录的 ./data；可用 NEW_API_LITE_DATA_DIR 覆盖。

set -Eeuo pipefail

APP_NAME="new-api"
IMAGE="${NEW_API_LITE_IMAGE:-ghcr.io/55gy/new-api-lite:latest-amd64}"
DATA_DIR="${NEW_API_LITE_DATA_DIR:-"$PWD/data"}"
HOST_PORT="${NEW_API_LITE_PORT:-3000}"
TIME_ZONE="${NEW_API_LITE_TZ:-Asia/Shanghai}"

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  COLOR_RESET=$'\033[0m'
  COLOR_RED=$'\033[31m'
  COLOR_GREEN=$'\033[32m'
  COLOR_YELLOW=$'\033[33m'
  COLOR_BLUE=$'\033[34m'
else
  COLOR_RESET=""
  COLOR_RED=""
  COLOR_GREEN=""
  COLOR_YELLOW=""
  COLOR_BLUE=""
fi

info() { printf '%b[信息]%b %s\n' "$COLOR_BLUE" "$COLOR_RESET" "$*"; }
success() { printf '%b[完成]%b %s\n' "$COLOR_GREEN" "$COLOR_RESET" "$*"; }
warn() { printf '%b[注意]%b %s\n' "$COLOR_YELLOW" "$COLOR_RESET" "$*"; }
error() { printf '%b[错误]%b %s\n' "$COLOR_RED" "$COLOR_RESET" "$*" >&2; }

usage() {
  cat <<'EOF'
用法：
  ./installl.sh                 打开交互式管理面板
  ./installl.sh install         首次安装或启动已有容器
  ./installl.sh update          拉取最新镜像并安全重建容器，保留 data
  ./installl.sh start|stop|restart
  ./installl.sh status|logs
  ./installl.sh uninstall       删除容器，保留 data
  ./installl.sh remove-data     删除 data（需要输入 DELETE）

环境变量：
  NEW_API_LITE_DATA_DIR  数据目录，默认当前目录/data
  NEW_API_LITE_PORT      宿主机端口，默认 3000
  NEW_API_LITE_IMAGE     镜像，默认 ghcr.io/55gy/new-api-lite:latest-amd64
  NEW_API_LITE_TZ         时区，默认 Asia/Shanghai
EOF
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    error "未检测到 Docker。请先安装并启动 Docker 服务。"
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    error "Docker 服务不可用。请确认当前用户有 Docker 权限且 Docker daemon 已启动。"
    exit 1
  fi
}

check_architecture() {
  case "$(uname -m)" in
    x86_64|amd64) ;;
    *) warn "当前架构为 $(uname -m)，默认镜像为 amd64。若环境不支持 amd64 模拟，请通过 NEW_API_LITE_IMAGE 指定可用镜像。" ;;
  esac
}

container_exists() {
  docker container inspect "$APP_NAME" >/dev/null 2>&1
}

container_running() {
  [[ "$(docker container inspect --format '{{.State.Running}}' "$APP_NAME" 2>/dev/null || true)" == "true" ]]
}

ensure_data_dir() {
  mkdir -p "$DATA_DIR"
  DATA_DIR="$(cd "$DATA_DIR" && pwd -P)"
}

run_container() {
  ensure_data_dir
  docker run \
    --name "$APP_NAME" \
    --detach \
    --init \
    --restart=unless-stopped \
    --publish "${HOST_PORT}:3000" \
    --env "TZ=${TIME_ZONE}" \
    --volume "${DATA_DIR}:/data" \
    "$IMAGE"
}

install() {
  require_docker
  check_architecture
  if container_exists; then
    if container_running; then
      info "容器 ${APP_NAME} 已在运行。"
    else
      docker start "$APP_NAME"
      success "已启动已有容器 ${APP_NAME}。"
    fi
    return
  fi
  info "拉取镜像：${IMAGE}"
  docker pull "$IMAGE"
  run_container
  success "已启动 ${APP_NAME}，访问地址：http://localhost:${HOST_PORT}"
}

update() {
  require_docker
  check_architecture
  info "拉取镜像：${IMAGE}"
  docker pull "$IMAGE"
  if container_exists; then
    info "删除旧容器 ${APP_NAME}（保留 ${DATA_DIR} 数据目录）。"
    docker rm --force "$APP_NAME"
  fi
  run_container
  success "镜像更新完成，已使用保留的数据目录创建新容器。"
}

start() {
  require_docker
  if ! container_exists; then
    warn "未找到容器 ${APP_NAME}，将执行安装。"
    install
    return
  fi
  if container_running; then
    info "容器 ${APP_NAME} 已在运行。"
  else
    docker start "$APP_NAME"
    success "已启动容器 ${APP_NAME}。"
  fi
}

stop() {
  require_docker
  if ! container_exists; then
    warn "未找到容器 ${APP_NAME}。"
    return
  fi
  if container_running; then
    docker stop "$APP_NAME"
    success "已停止容器 ${APP_NAME}。"
  else
    info "容器 ${APP_NAME} 已停止。"
  fi
}

restart() {
  require_docker
  if ! container_exists; then
    warn "未找到容器 ${APP_NAME}，将执行安装。"
    install
    return
  fi
  docker restart "$APP_NAME"
  success "已重启容器 ${APP_NAME}。"
}

status() {
  require_docker
  printf '应用名称：%s\n镜像：%s\n数据目录：%s\n端口：%s\n' "$APP_NAME" "$IMAGE" "$DATA_DIR" "$HOST_PORT"
  if container_exists; then
    docker ps --filter "name=^/${APP_NAME}$" --format '状态：{{.Status}}\n端口映射：{{.Ports}}\n镜像：{{.Image}}'
  else
    warn "容器 ${APP_NAME} 尚未创建。"
  fi
}

logs() {
  require_docker
  if ! container_exists; then
    error "未找到容器 ${APP_NAME}。"
    exit 1
  fi
  docker logs --follow --tail 200 "$APP_NAME"
}

confirm() {
  local prompt="$1"
  local answer
  read -r -p "${prompt} [y/N]: " answer
  [[ "$answer" == "y" || "$answer" == "Y" ]]
}

uninstall() {
  require_docker
  if ! container_exists; then
    warn "未找到容器 ${APP_NAME}。数据目录未做任何修改。"
    return
  fi
  warn "此操作仅删除容器 ${APP_NAME}，会保留数据目录：${DATA_DIR}"
  if confirm "确认继续删除容器吗？"; then
    docker rm --force "$APP_NAME"
    success "容器已删除；数据目录仍保留在 ${DATA_DIR}。"
  else
    info "已取消。"
  fi
}

remove_data() {
  require_docker
  ensure_data_dir
  if [[ "$DATA_DIR" == "/" || "$DATA_DIR" == "$HOME" || "$DATA_DIR" == "." ]]; then
    error "拒绝删除不安全的数据目录：${DATA_DIR}"
    exit 1
  fi
  warn "此操作会不可恢复地删除数据目录：${DATA_DIR}"
  warn "如容器仍在运行，请先停止或卸载容器。"
  local answer
  read -r -p "输入 DELETE 以确认永久删除数据：" answer
  if [[ "$answer" == "DELETE" ]]; then
    rm -rf -- "$DATA_DIR"
    success "数据目录已删除。"
  else
    info "已取消。"
  fi
}

pause() {
  if [[ -t 0 ]]; then
    read -r -n 1 -s -p "按任意键返回菜单..."
    printf '\n'
  fi
}

menu() {
  while true; do
    printf '\n%bnew-api-lite Docker 管理面板%b\n' "$COLOR_GREEN" "$COLOR_RESET"
    printf '镜像：%s\n数据目录：%s\n端口：%s\n\n' "$IMAGE" "$DATA_DIR" "$HOST_PORT"
    printf '1. 安装 / 启动\n'
    printf '2. 更新镜像并重建容器（保留 data）\n'
    printf '3. 查看状态\n'
    printf '4. 查看实时日志\n'
    printf '5. 停止容器\n'
    printf '6. 重启容器\n'
    printf '7. 卸载容器（保留 data）\n'
    printf '8. 永久删除 data（危险）\n'
    printf '0. 退出\n'
    read -r -p '请选择操作：' choice
    case "$choice" in
      1) install; pause ;;
      2) if confirm '确认拉取最新镜像并重建容器吗？'; then update; else info '已取消。'; fi; pause ;;
      3) status; pause ;;
      4) logs ;;
      5) stop; pause ;;
      6) restart; pause ;;
      7) uninstall; pause ;;
      8) remove_data; pause ;;
      0) exit 0 ;;
      *) warn '无效选项。'; pause ;;
    esac
  done
}

case "${1:-}" in
  "") menu ;;
  install) install ;;
  update) update ;;
  start) start ;;
  stop) stop ;;
  restart) restart ;;
  status) status ;;
  logs) logs ;;
  uninstall) uninstall ;;
  remove-data) remove_data ;;
  -h|--help|help) usage ;;
  *) error "未知命令：$1"; usage; exit 1 ;;
esac
