#!/usr/bin/env bash
set -euo pipefail

# Automated binary installation for Nebula Manager (non-Docker)
# This script builds the project from the current source tree and sets up systemd service files.

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
INSTALL_DIR="${INSTALL_DIR:-/opt/nebula-manager}"
ENV_FILE="${ENV_FILE:-/etc/nebula-manager.env}"
SERVICE_FILE="/etc/systemd/system/nebula-manager.service"
DEFAULT_DSN="root:123150.wangzai7@tcp(10.10.10.1:3306)/nebula_manager?charset=utf8mb4&parseTime=True&loc=Local"
DEFAULT_PORT="8080"
DEFAULT_DATA_DIR="/var/lib/nebula-manager"

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "[error] 请使用 root 权限执行此脚本 (sudo ./scripts/install_binary.sh)" >&2
    exit 1
  fi
}

check_dependency() {
  local cmd=$1
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "[error] 未找到命令 '$cmd'，请先安装后重试。" >&2
    exit 1
  fi
}

build_frontend() {
  if [ ! -d "$ROOT_DIR/frontend/dist" ]; then
    echo "[build] 前端产物缺失，执行 npm install && npm run build"
    (cd "$ROOT_DIR/frontend" && npm install && npm run build)
  else
    echo "[build] 使用已有的 frontend/dist"
  fi
}

build_backend() {
  echo "[build] 编译后端二进制"
  env CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -o "$ROOT_DIR/build/nebula_manager" "$ROOT_DIR"
}

install_files() {
  echo "[install] 拷贝文件到 $INSTALL_DIR"
  rm -rf "$INSTALL_DIR"
  mkdir -p "$INSTALL_DIR"

  cp "$ROOT_DIR/build/nebula_manager" "$INSTALL_DIR/"
  cp "$ROOT_DIR/README.md" "$INSTALL_DIR/"
  if [ -f "$ROOT_DIR/config.yaml.default" ]; then
    cp "$ROOT_DIR/config.yaml.default" "$INSTALL_DIR/"
  fi
  mkdir -p "$INSTALL_DIR/frontend"
  cp -R "$ROOT_DIR/frontend/dist" "$INSTALL_DIR/frontend/"
  mkdir -p "$DEFAULT_DATA_DIR"
}

write_env_file() {
  echo "[config] 生成环境变量文件 ($ENV_FILE)"
  local mysql_dsn server_port data_dir api_base

  read -rp "请输入 MySQL DSN [$DEFAULT_DSN]: " mysql_dsn
  mysql_dsn=${mysql_dsn:-$DEFAULT_DSN}

  read -rp "服务监听端口 [$DEFAULT_PORT]: " server_port
  server_port=${server_port:-$DEFAULT_PORT}

  read -rp "数据目录 [$DEFAULT_DATA_DIR]: " data_dir
  data_dir=${data_dir:-$DEFAULT_DATA_DIR}

  read -rp "对外访问地址 (NEBULA_API_BASE，可留空使用默认检测): " api_base

  cat >"$ENV_FILE" <<ENV
NEBULA_MYSQL_DSN="$mysql_dsn"
NEBULA_SERVER_PORT="$server_port"
NEBULA_DATA_DIR="$data_dir"
NEBULA_FRONTEND_DIR="$INSTALL_DIR/frontend/dist"
NEBULA_API_BASE="$api_base"
ENV

  chmod 600 "$ENV_FILE"
}

write_service_file() {
  echo "[config] 写入 systemd 服务单元 ($SERVICE_FILE)"
  cat >"$SERVICE_FILE" <<UNIT
[Unit]
Description=Nebula Manager 控制面板
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=-$ENV_FILE
ExecStart=$INSTALL_DIR/nebula_manager
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
UNIT
}

reload_systemd() {
  echo "[systemd] 重载并启用服务"
  systemctl daemon-reload
  systemctl enable --now nebula-manager.service
  systemctl status nebula-manager.service --no-pager || true
}

cleanup() {
  rm -f "$ROOT_DIR/build/nebula_manager"
}

print_summary() {
  cat <<SUMMARY

安装完成 ✅
- 运行目录: $INSTALL_DIR
- 环境变量文件: $ENV_FILE
- 服务管理: systemctl [status|restart|stop] nebula-manager

如需修改配置:
1. 编辑 $ENV_FILE
2. 执行 sudo systemctl restart nebula-manager.service

SUMMARY
}

main() {
  require_root
  mkdir -p "$ROOT_DIR/build"
  check_dependency go
  check_dependency npm
  build_frontend
  build_backend
  install_files
  write_env_file
  write_service_file
  reload_systemd
  cleanup
  print_summary
}

main "$@"
