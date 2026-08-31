#!/usr/bin/env bash
# new-api-lite Docker 管理面板。
# 默认把数据保存在执行命令所在目录的 ./data；可用 NEW_API_LITE_DATA_DIR 覆盖。

set -Eeuo pipefail

APP_NAME="new-api"
BASE_IMAGE="${NEW_API_LITE_IMAGE:-ghcr.io/55gy/new-api-lite:latest}"
IMAGE="$BASE_IMAGE"
MIRROR_MODE="${NEW_API_LITE_MIRROR_MODE:-direct}"
DOCKER_MIRRORS="${NEW_API_LITE_DOCKER_MIRRORS:-gh-proxy.org/docker/ gh-proxy.com/docker/}"
DATA_DIR="${NEW_API_LITE_DATA_DIR:-"$PWD/data"}"
HOST_PORT="${NEW_API_LITE_PORT:-3000}"
TIME_ZONE="${NEW_API_LITE_TZ:-Asia/Shanghai}"
AUTO_INSTALL_DOCKER="${NEW_API_LITE_AUTO_INSTALL_DOCKER:-1}"
SYSTEM_ARCH="$(uname -m)"
OS_ID="unknown"
OS_NAME="unknown"
DOCKER_PREFIX=()

if [[ -r /etc/os-release ]]; then
  OS_ID="$(awk -F= '$1 == "ID" {gsub(/"/, "", $2); print $2; exit}' /etc/os-release)"
  OS_NAME="$(awk -F= '$1 == "PRETTY_NAME" {gsub(/"/, "", $2); print $2; exit}' /etc/os-release)"
fi

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
  ./install.sh                 打开交互式管理面板
  ./install.sh install         首次安装或启动已有容器
  ./install.sh update          拉取最新镜像并安全重建容器，保留 data
  ./install.sh start|stop|restart
  ./install.sh status|logs|check
  ./install.sh uninstall       删除容器，保留 data
  ./install.sh remove-data     删除 data（需要输入 DELETE）

环境变量：
  NEW_API_LITE_DATA_DIR  数据目录，默认当前目录/data
  NEW_API_LITE_PORT      宿主机端口，默认 3000
  NEW_API_LITE_IMAGE     完整镜像名，默认 ghcr.io/55gy/new-api-lite:latest（多架构自动选择）
  NEW_API_LITE_MIRROR_MODE  direct 仅直连；mirror 优先镜像；auto 直连失败后尝试镜像
  NEW_API_LITE_DOCKER_MIRRORS  GHCR 加速注册表前缀，空格分隔，可自行替换
  NEW_API_LITE_TZ         时区，默认 Asia/Shanghai
  NEW_API_LITE_AUTO_INSTALL_DOCKER  为 0 时禁止自动安装 Docker
EOF
}

docker_cli() {
  "${DOCKER_PREFIX[@]}" docker "$@"
}

run_as_root() {
  if (( EUID == 0 )); then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    error "当前用户不是 root，且未安装 sudo，无法安装或启动 Docker。"
    return 1
  fi
  sudo "$@"
}

configure_docker_access() {
  if docker info >/dev/null 2>&1; then
    DOCKER_PREFIX=()
    return 0
  fi
  if (( EUID != 0 )) && command -v sudo >/dev/null 2>&1 && sudo docker info >/dev/null 2>&1; then
    DOCKER_PREFIX=(sudo)
    warn "当前用户尚未获得 docker 组权限，脚本将在本次运行中通过 sudo 管理 Docker。"
    return 0
  fi
  return 1
}

start_docker_daemon() {
  if command -v systemctl >/dev/null 2>&1; then
    run_as_root systemctl enable --now docker || true
  fi
  if ! configure_docker_access && command -v service >/dev/null 2>&1; then
    run_as_root service docker start || true
  fi
}

install_docker() {
  if [[ "$AUTO_INSTALL_DOCKER" != "1" ]]; then
    error "未安装 Docker，且 NEW_API_LITE_AUTO_INSTALL_DOCKER 已禁用自动安装。"
    return 1
  fi
  info "未检测到 Docker；将为 ${OS_NAME} 自动安装 Docker。"
  if command -v apt-get >/dev/null 2>&1; then
    run_as_root apt-get update
    run_as_root env DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io
  elif command -v dnf >/dev/null 2>&1; then
    run_as_root dnf install -y moby-engine
  elif command -v yum >/dev/null 2>&1; then
    run_as_root yum install -y docker
  elif command -v apk >/dev/null 2>&1; then
    run_as_root apk add docker
  else
    error "未识别可用的软件包管理器。请先手动安装 Docker 后重新运行脚本。"
    return 1
  fi
  start_docker_daemon
}

ensure_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    install_docker || return 1
  fi
  if ! configure_docker_access; then
    info "Docker 已安装但服务不可用，正在尝试启动 Docker daemon。"
    start_docker_daemon
  fi
  if ! configure_docker_access; then
    error "Docker 服务仍不可用。请确认 docker daemon 已启动，并为当前用户授予 Docker 访问权限。"
    return 1
  fi
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    error "未检测到 Docker。请执行 ./install.sh install 以自动安装 Docker。"
    exit 1
  fi
  if ! configure_docker_access; then
    error "Docker 服务不可用。请执行 ./install.sh install 尝试启动 Docker。"
    exit 1
  fi
}

add_image_candidate() {
  local candidate="$1"
  local existing
  [[ -n "$candidate" ]] || return 0
  for existing in "${IMAGE_CANDIDATES[@]:-}"; do
    [[ "$existing" == "$candidate" ]] && return 0
  done
  IMAGE_CANDIDATES+=("$candidate")
}

build_image_candidates() {
  IMAGE_CANDIDATES=()
  case "$MIRROR_MODE" in
    direct)
      add_image_candidate "$BASE_IMAGE"
      ;;
    mirror|auto)
      if [[ "$MIRROR_MODE" == "auto" ]]; then
        add_image_candidate "$BASE_IMAGE"
      fi
      if [[ "$BASE_IMAGE" == ghcr.io/* ]]; then
        local prefix
        for prefix in $DOCKER_MIRRORS; do
          prefix="${prefix%/}/"
          add_image_candidate "${prefix}${BASE_IMAGE}"
        done
      else
        warn "镜像不是 ghcr.io 地址，跳过 GHCR 加速节点：${BASE_IMAGE}"
      fi
      add_image_candidate "$BASE_IMAGE"
      ;;
    *)
      error "NEW_API_LITE_MIRROR_MODE 只能是 direct、mirror 或 auto，当前值：${MIRROR_MODE}"
      return 1
      ;;
  esac
}

pull_image() {
  build_image_candidates || return 1
  local candidate
  for candidate in "${IMAGE_CANDIDATES[@]}"; do
    info "拉取镜像：${candidate}"
    if docker_cli pull "$candidate"; then
      IMAGE="$candidate"
      success "镜像拉取成功：${IMAGE}"
      return 0
    fi
    warn "镜像节点不可用，尝试下一个节点：${candidate}"
  done
  error "所有镜像节点均拉取失败。可设置 NEW_API_LITE_IMAGE 或 NEW_API_LITE_DOCKER_MIRRORS 更换节点。"
  return 1
}

check_architecture() {
  info "系统：${OS_NAME}；架构：${SYSTEM_ARCH}。"
  case "$SYSTEM_ARCH" in
    x86_64|amd64) info "将通过多架构 latest 镜像自动拉取当前 amd64 变体。" ;;
    aarch64|arm64) info "将通过多架构 latest 镜像自动拉取当前 ARM64 变体。" ;;
    *) warn "当前架构为 ${SYSTEM_ARCH}。Docker 会尝试从多架构 latest 镜像选择匹配变体；如不兼容，可通过 NEW_API_LITE_IMAGE 手动覆盖。" ;;
  esac
}

check() {
  check_architecture
  if command -v docker >/dev/null 2>&1; then
    if configure_docker_access; then
      success "Docker 已安装且服务可用。"
      docker_cli version --format 'Docker Server: {{.Server.Version}} ({{.Server.Arch}})' 2>/dev/null || true
    else
      warn "Docker 已安装，但当前服务不可用或当前用户无权访问。执行 install 可尝试自动启动服务。"
    fi
  else
    warn "Docker 未安装。执行 install 会按当前系统的软件包管理器自动安装。"
  fi
}

container_exists() {
  docker_cli container inspect "$APP_NAME" >/dev/null 2>&1
}

container_running() {
  [[ "$(docker_cli container inspect --format '{{.State.Running}}' "$APP_NAME" 2>/dev/null || true)" == "true" ]]
}

ensure_data_dir() {
  mkdir -p "$DATA_DIR"
  DATA_DIR="$(cd "$DATA_DIR" && pwd -P)"
}

run_container() {
  ensure_data_dir
  docker_cli run \
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
  ensure_docker || exit 1
  check_architecture
  if container_exists; then
    if container_running; then
      info "容器 ${APP_NAME} 已在运行。"
    else
      docker_cli start "$APP_NAME"
      success "已启动已有容器 ${APP_NAME}。"
    fi
    return
  fi
  pull_image || exit 1
  run_container
  success "已启动 ${APP_NAME}，访问地址：http://localhost:${HOST_PORT}"
}

update() {
  ensure_docker || exit 1
  check_architecture
  pull_image || exit 1
  if container_exists; then
    info "删除旧容器 ${APP_NAME}（保留 ${DATA_DIR} 数据目录）。"
    docker_cli rm --force "$APP_NAME"
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
    docker_cli stop "$APP_NAME"
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
  docker_cli restart "$APP_NAME"
  success "已重启容器 ${APP_NAME}。"
}

status() {
  require_docker
  printf '应用名称：%s\n镜像：%s\n数据目录：%s\n端口：%s\n' "$APP_NAME" "$IMAGE" "$DATA_DIR" "$HOST_PORT"
  if container_exists; then
    docker_cli ps --filter "name=^/${APP_NAME}$" --format '状态：{{.Status}}\n端口映射：{{.Ports}}\n镜像：{{.Image}}'
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
  docker_cli logs --follow --tail 200 "$APP_NAME"
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
    docker_cli rm --force "$APP_NAME"
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
  check) check ;;
  logs) logs ;;
  uninstall) uninstall ;;
  remove-data) remove_data ;;
  -h|--help|help) usage ;;
  *) error "未知命令：$1"; usage; exit 1 ;;
esac
