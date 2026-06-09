#!/usr/bin/env bash
set -Eeuo pipefail

REPO_OWNER="${REPO_OWNER:-1975435449}"
REPO_NAME="${REPO_NAME:-s5}"
INSTALL_DIR="${INSTALL_DIR:-/etc/nps}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
WORK_DIR="${WORK_DIR:-/opt/nps-socks5}"
LOG_DIR="${LOG_DIR:-/var/log/nps}"
SERVICE_NAME="${SERVICE_NAME:-nps}"

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "Please run as root: sudo bash $0 $*" >&2
    exit 1
  fi
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing command: $1" >&2
    exit 1
  }
}

detect_asset() {
  local os arch arm_ver
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    i386|i686) arch="386" ;;
    aarch64|arm64) arch="arm64" ;;
    armv7l)
      arch="arm_v7"
      ;;
    armv6l)
      arch="arm_v6"
      ;;
    armv5l)
      arch="arm_v5"
      ;;
    arm*)
      arm_ver="$(grep -m1 -o 'ARMv[0-9]' /proc/cpuinfo 2>/dev/null | tr '[:upper:]' '[:lower:]' || true)"
      case "$arm_ver" in
        armv7) arch="arm_v7" ;;
        armv6) arch="arm_v6" ;;
        armv5) arch="arm_v5" ;;
        *) arch="arm_v7" ;;
      esac
      ;;
    mips64le|mips64|mipsle|mips) ;;
    *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
  esac

  if [ "$os" != "linux" ]; then
    echo "This stable installer only supports Linux systemd servers." >&2
    exit 1
  fi
  echo "${os}_${arch}_server.tar.gz"
}

latest_download_url() {
  local asset api json url
  asset="$(detect_asset)"
  api="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
  json="$(curl -fsSL "$api")"
  url="$(printf '%s' "$json" | grep -Eo '"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]+"' | sed -E 's/.*"([^"]+)"/\1/' | grep "/${asset}$" | head -n 1 || true)"
  if [ -z "$url" ]; then
    echo "Could not find release asset: $asset" >&2
    exit 1
  fi
  echo "$url"
}

write_service() {
  cat >"/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=nps socks5 server
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${BIN_DIR}/nps -conf_path=${INSTALL_DIR} service
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
}

write_logrotate() {
  if [ -d /etc/logrotate.d ]; then
    cat >"/etc/logrotate.d/${SERVICE_NAME}" <<EOF
${LOG_DIR}/*.log {
    daily
    rotate 14
    missingok
    notifempty
    compress
    delaycompress
    copytruncate
}
EOF
  fi
}

install_server() {
  need_root "$@"
  need_cmd curl
  need_cmd tar
  need_cmd systemctl

  local url archive tmp unpacked
  url="$(latest_download_url)"
  tmp="$(mktemp -d)"
  archive="${tmp}/$(basename "$url")"
  unpacked="${tmp}/pkg"
  mkdir -p "$unpacked" "$WORK_DIR" "$INSTALL_DIR" "$BIN_DIR" "$LOG_DIR"

  echo "Downloading $url"
  curl -fL --progress-bar -o "$archive" "$url"
  tar -xzf "$archive" -C "$unpacked"

  if [ ! -f "${unpacked}/nps" ]; then
    echo "Bad package: nps binary not found" >&2
    exit 1
  fi

  if systemctl list-unit-files | grep -q "^${SERVICE_NAME}.service"; then
    systemctl stop "$SERVICE_NAME" || true
  fi

  install -m 0755 "${unpacked}/nps" "${BIN_DIR}/nps"
  cp -a "${unpacked}/web" "$INSTALL_DIR/"

  if [ ! -d "${INSTALL_DIR}/conf" ]; then
    cp -a "${unpacked}/conf" "$INSTALL_DIR/"
  else
    cp -a "${INSTALL_DIR}/conf" "${INSTALL_DIR}/conf.bak.$(date +%Y%m%d%H%M%S)"
    for file in server.key server.pem; do
      [ -f "${INSTALL_DIR}/conf/${file}" ] || cp "${unpacked}/conf/${file}" "${INSTALL_DIR}/conf/${file}"
    done
    for file in clients.json hosts.json tasks.json global.json; do
      [ -f "${INSTALL_DIR}/conf/${file}" ] || [ ! -f "${unpacked}/conf/${file}" ] || cp "${unpacked}/conf/${file}" "${INSTALL_DIR}/conf/${file}"
    done
    [ -f "${INSTALL_DIR}/conf/nps.conf" ] || cp "${unpacked}/conf/nps.conf" "${INSTALL_DIR}/conf/nps.conf"
  fi

  if grep -q '^log_path=' "${INSTALL_DIR}/conf/nps.conf"; then
    sed -i "s#^log_path=.*#log_path=${LOG_DIR}/nps.log#" "${INSTALL_DIR}/conf/nps.conf"
  else
    printf '\nlog_path=%s/nps.log\n' "$LOG_DIR" >>"${INSTALL_DIR}/conf/nps.conf"
  fi

  write_service
  write_logrotate
  systemctl daemon-reload
  systemctl enable --now "$SERVICE_NAME"

  echo "Installed and started ${SERVICE_NAME}."
  systemctl --no-pager --full status "$SERVICE_NAME" || true
  echo "Config: ${INSTALL_DIR}/conf/nps.conf"
  echo "Logs: journalctl -u ${SERVICE_NAME} -f  or  ${LOG_DIR}/nps.log"
}

uninstall_server() {
  need_root "$@"
  systemctl disable --now "$SERVICE_NAME" || true
  rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
  systemctl daemon-reload || true
  rm -f "${BIN_DIR}/nps"
  echo "Service removed. Config is kept at ${INSTALL_DIR}; remove it manually if you no longer need it."
}

case "${1:-install-server}" in
  install-server|install|server)
    install_server "$@"
    ;;
  uninstall)
    uninstall_server "$@"
    ;;
  status)
    systemctl --no-pager --full status "$SERVICE_NAME"
    ;;
  *)
    echo "Usage: $0 [install-server|uninstall|status]" >&2
    exit 1
    ;;
esac
