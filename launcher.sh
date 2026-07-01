#!/usr/bin/env bash
# QoLauncher — interactive deployment wrapper
# Usage: ./launcher.sh [--run-all|--stop]

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APPS_DIR="${ROOT_DIR}/apps"
LAUNCHER_DIR="${ROOT_DIR}/.launcher"
ENV_FILE="${ROOT_DIR}/.env"
COMPOSE_FILE="${ROOT_DIR}/docker-compose.yml"
STATE_FILE="${LAUNCHER_DIR}/state"
LOGS_DIR="${ROOT_DIR}/logs"
IMAGE_NAME="qolauncher:latest"

# ─── Colors ───────────────────────────────────────────────────────────────────
if [[ -t 1 ]]; then
  BOLD='\033[1m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  CYAN='\033[0;36m'
  RED='\033[0;31m'
  NC='\033[0m'
else
  BOLD='' GREEN='' YELLOW='' CYAN='' RED='' NC=''
fi

info()  { echo -e "${CYAN}[info]${NC} $*"; }
ok()    { echo -e "${GREEN}[ok]${NC} $*"; }
warn()  { echo -e "${YELLOW}[warn]${NC} $*"; }
err()   { echo -e "${RED}[error]${NC} $*" >&2; }

# ─── Helpers ──────────────────────────────────────────────────────────────────
need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { err "'$1' tidak ditemukan. Install dulu."; exit 1; }
}

docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_FILE" "$@"
  else
    docker-compose -f "$COMPOSE_FILE" "$@"
  fi
}

prompt() {
  local msg="$1" default="${2:-}"
  if [[ -n "$default" ]]; then
    read -r -p "${msg} [${default}]: " ans
    echo "${ans:-$default}"
  else
    read -r -p "${msg}: " ans
    echo "$ans"
  fi
}

prompt_secret() {
  local msg="$1" default="${2:-}"
  local ans
  if [[ -n "$default" ]]; then
    read -r -s -p "${msg} [****]: " ans
    echo ""
    echo "${ans:-$default}"
  else
    read -r -s -p "${msg}: " ans
    echo ""
    echo "$ans"
  fi
}

prompt_yes_no() {
  local msg="$1" default="${2:-y}"
  local hint="y/n"
  [[ "$default" == "y" ]] && hint="Y/n" || hint="y/N"
  local ans
  read -r -p "${msg} [${hint}]: " ans
  ans="${ans:-$default}"
  [[ "$ans" =~ ^[Yy] ]]
}

mkdir_safe() {
  mkdir -p "$@"
}

read_env_value() {
  local file="$1" key="$2" default="${3:-}"
  [[ -f "$file" ]] || { echo "$default"; return; }
  local line val
  line="$(grep -E "^${key}=" "$file" 2>/dev/null | tail -1 || true)"
  [[ -z "$line" ]] && { echo "$default"; return; }
  val="${line#*=}"
  val="${val#\"}"
  val="${val%\"}"
  echo "${val:-$default}"
}

service_name() {
  local id="$1"
  local slug
  slug="$(echo "$id" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9-' '-' | sed -e 's/-\+$//' -e 's/^-\+//')"
  [[ -n "$slug" ]] || slug="app"
  echo "qolauncher-${slug}"
}

# Resolve service key in docker-compose.yml (handles legacy names with trailing dash)
compose_service_for_app() {
  local app_id="$1"
  local expected legacy svc
  expected="$(service_name "$app_id")"
  legacy="${expected}-"

  [[ -f "$COMPOSE_FILE" ]] || { echo "$expected"; return; }

  while IFS= read -r svc; do
    [[ -z "$svc" ]] && continue
    if [[ "$svc" == "$expected" || "$svc" == "$legacy" ]]; then
      echo "$svc"
      return
    fi
  done < <(docker_compose config --services 2>/dev/null || true)

  svc="$(grep -B30 "logs/${app_id}" "$COMPOSE_FILE" 2>/dev/null | grep -E '^  qolauncher-' | tail -1 | sed -E 's/^  ([^:]+):.*/\1/')"
  if [[ -n "$svc" ]]; then
    echo "$svc"
    return
  fi
  echo "$expected"
}

# ─── Container status (single source of truth) ─────────────────────────────────
# Active = running | restarting | paused (sama di semua menu)

container_state() {
  local svc="$1"
  local cid state
  cid="$(docker_compose ps -aq "$svc" 2>/dev/null | head -1 || true)"
  [[ -n "$cid" ]] || return 1
  state="$(docker inspect -f '{{.State.Status}}' "$cid" 2>/dev/null || true)"
  echo "$state"
}

container_is_active() {
  local svc="$1"
  local state
  state="$(container_state "$svc" 2>/dev/null || true)"
  [[ "$state" == "running" || "$state" == "restarting" || "$state" == "paused" ]]
}

container_status_for_app() {
  local idx="$1"
  if [[ ! -f "$COMPOSE_FILE" ]]; then
    echo "not deployed"
    return
  fi
  local svc state cid found=0 s
  svc="$(compose_service_for_app "${APP_IDS[$idx]}")"
  while IFS= read -r s; do
    [[ "$s" == "$svc" ]] && { found=1; break; }
  done < <(docker_compose config --services 2>/dev/null || true)
  if [[ $found -eq 0 ]]; then
    echo "not deployed"
    return
  fi
  cid="$(docker_compose ps -aq "$svc" 2>/dev/null | head -1 || true)"
  if [[ -z "$cid" ]]; then
    echo "stopped"
    return
  fi
  state="$(docker inspect -f '{{.State.Status}}' "$cid" 2>/dev/null || true)"
  echo "${state:-stopped}"
}

format_state_label() {
  local state="$1"
  case "$state" in
    running)    echo -e "${GREEN}running${NC}" ;;
    restarting) echo -e "${YELLOW}restarting${NC}" ;;
    paused)     echo -e "${YELLOW}paused${NC}" ;;
    exited)     echo -e "${RED}exited${NC}" ;;
    dead)       echo -e "${RED}dead${NC}" ;;
    stopped|created) echo -e "${YELLOW}${state}${NC}" ;;
    "not deployed") echo -e "${YELLOW}not deployed${NC}" ;;
    *)          echo "$state" ;;
  esac
}

normalize_relpath() {
  local p="$1"
  while [[ "$p" == *//* ]]; do
    p="${p//\/\//\/}"
  done
  echo "$p"
}

app_is_active() {
  local idx="$1"
  [[ -f "$COMPOSE_FILE" ]] || return 1
  local svc
  svc="$(compose_service_for_app "${APP_IDS[$idx]}")"
  container_is_active "$svc"
}

detect_active_apps() {
  detect_apps
  ACTIVE_INDICES=()
  local i
  for i in "${!APP_IDS[@]}"; do
    if app_is_active "$i"; then
      ACTIVE_INDICES+=("$i")
    fi
  done
}

active_app_names() {
  detect_active_apps
  local names="" idx
  for idx in "${ACTIVE_INDICES[@]}"; do
    names+="${APP_IDS[$idx]},"
  done
  echo "${names%,}"
}

active_app_summary() {
  detect_active_apps
  local parts="" idx state
  for idx in "${ACTIVE_INDICES[@]}"; do
    state="$(container_status_for_app "$idx")"
    parts+="${APP_IDS[$idx]} (${state}),"
  done
  echo "${parts%,}"
}

skip_app_file() {
  local base="$1"
  [[ "$base" == ".env" || "$base" == ".env.example" ]] && return 0
  [[ "$base" == ".gitkeep" || "$base" == "README.md" ]] && return 0
  [[ "$base" == "main.go" || "$base" == *.go ]] && return 0
  return 1
}

save_state_multi() {
  mkdir_safe "$LAUNCHER_DIR"
  {
    echo "DEPLOYED_COUNT=${#DEPLOY_INDICES[@]}"
    local ids="" idx
    for idx in "${DEPLOY_INDICES[@]}"; do
      ids+="${APP_IDS[$idx]},"
    done
    ids="${ids%,}"
    echo "DEPLOYED_IDS=${ids}"
  } >"$STATE_FILE"
}

load_state() {
  if [[ -f "$STATE_FILE" ]]; then
    # shellcheck source=/dev/null
    source "$STATE_FILE"
  fi
}

# ─── App detection ───────────────────────────────────────────────────────────
declare -a APP_IDS=()
declare -a APP_DIRS=()
declare -a APP_BINARIES=()
declare -a APP_ENV_FILES=()
declare -a DEPLOY_INDICES=()   # semua app di compose
declare -a SELECTED_INDICES=() # app yang dipilih user untuk aksi ini
declare -a ACTIVE_INDICES=()  # app dengan container aktif (running/restarting/paused)

detect_apps() {
  APP_IDS=()
  APP_DIRS=()
  APP_BINARIES=()
  APP_ENV_FILES=()

  mkdir_safe "$APPS_DIR"

  local found_subdir=0
  for dir in "$APPS_DIR"/*/; do
    [[ -d "$dir" ]] || continue
    found_subdir=1
    local app_id
    app_id="$(basename "$dir")"
    [[ "$app_id" == "*" ]] && continue

    local binary="" env_file=""
    dir="${dir%/}"
    [[ -f "$dir/.env" ]] && env_file="$dir/.env"
    for f in "$dir"/*; do
      [[ -e "$f" ]] || continue
      local base
      base="$(basename "$f")"
      skip_app_file "$base" && continue
      if [[ -f "$f" && ! -d "$f" ]]; then
        binary="$base"
        break
      fi
    done

    [[ -z "$binary" ]] && continue
    APP_IDS+=("$app_id")
    APP_DIRS+=("$dir")
    APP_BINARIES+=("$binary")
    APP_ENV_FILES+=("${env_file:-}")
  done

  if [[ $found_subdir -eq 0 ]]; then
    local flat_binary="" flat_env=""
    [[ -f "$APPS_DIR/.env" ]] && flat_env="$APPS_DIR/.env"
    for f in "$APPS_DIR"/*; do
      [[ -e "$f" ]] || continue
      local base
      base="$(basename "$f")"
      skip_app_file "$base" && continue
      if [[ -f "$f" && ! -d "$f" ]]; then
        flat_binary="$base"
        break
      fi
    done
    if [[ -n "$flat_binary" ]]; then
      APP_IDS+=("default")
      APP_DIRS+=("$APPS_DIR")
      APP_BINARIES+=("$flat_binary")
      APP_ENV_FILES+=("${flat_env:-}")
    fi
  fi
}

get_effective_ports() {
  local idx="$1"
  local -n _app_port="$2"
  local -n _log_port="$3"

  local env_f="${APP_ENV_FILES[$idx]:-}"
  _app_port="$(read_env_value "$ENV_FILE" APP_PORT "8080")"
  _log_port="$(read_env_value "$ENV_FILE" LOG_PORT "8081")"

  if [[ -n "$env_f" && -f "$env_f" ]]; then
    if grep -q '^APP_PORT=' "$env_f" 2>/dev/null; then
      _app_port="$(read_env_value "$env_f" APP_PORT "$_app_port")"
    fi
    if grep -q '^LOG_PORT=' "$env_f" 2>/dev/null; then
      _log_port="$(read_env_value "$env_f" LOG_PORT "$_log_port")"
    fi
  fi
}

validate_deploy_ports() {
  declare -A seen_app seen_log
  local idx app_port log_port
  for idx in "${DEPLOY_INDICES[@]}"; do
    get_effective_ports "$idx" app_port log_port
    if [[ "$app_port" != "0" ]]; then
      if [[ -n "${seen_app[$app_port]:-}" ]]; then
        err "APP_PORT ${app_port} bentrok antara '${seen_app[$app_port]}' dan '${APP_IDS[$idx]}'"
        err "Atur port unik di apps/<app-id>/.env"
        return 1
      fi
      seen_app[$app_port]="${APP_IDS[$idx]}"
    fi
    if [[ -n "${seen_log[$log_port]:-}" ]]; then
      err "LOG_PORT ${log_port} bentrok antara '${seen_log[$log_port]}' dan '${APP_IDS[$idx]}'"
      err "Atur port unik di apps/<app-id>/.env"
      return 1
    fi
    seen_log[$log_port]="${APP_IDS[$idx]}"
  done
  return 0
}

show_detected_apps() {
  if [[ ${#APP_IDS[@]} -eq 0 ]]; then
    warn "Tidak ada app di ${APPS_DIR}/"
    echo ""
    echo "  Struktur yang didukung:"
    echo "    apps/myapp/server          ← binary"
    echo "    apps/myapp/.env            ← optional env per app"
    echo ""
    echo "  Atau flat:"
    echo "    apps/server"
    echo "    apps/.env"
    return 1
  fi

  echo ""
  echo -e "${BOLD}App terdeteksi di apps/:${NC}"
  local i app_port log_port
  for i in "${!APP_IDS[@]}"; do
    get_effective_ports "$i" app_port log_port
    local env_status=".env tidak ada (pakai .env global)"
    [[ -n "${APP_ENV_FILES[$i]}" ]] && env_status=".env: ${APP_ENV_FILES[$i]}"
    local port_info="app :${app_port}  viewer :${log_port}"
    [[ "$app_port" == "0" ]] && port_info="(no HTTP)  viewer :${log_port}"
    local cstatus cstatus_label
    cstatus="$(container_status_for_app "$i")"
    cstatus_label="$(format_state_label "$cstatus")"
    echo "  [$((i + 1))] ${APP_IDS[$i]}"
    echo -e "       status : ${cstatus_label}"
    echo "       binary : ${APP_DIRS[$i]}/${APP_BINARIES[$i]}"
    echo "       env    : ${env_status}"
    echo "       port   : ${port_info}"
  done
  echo ""
  return 0
}

dedupe_indices() {
  declare -A seen=()
  local -a unique=()
  local idx
  for idx in "${SELECTED_INDICES[@]}"; do
    [[ -n "${seen[$idx]+x}" ]] && continue
    seen[$idx]=1
    unique+=("$idx")
  done
  SELECTED_INDICES=("${unique[@]}")
}

indices_to_services() {
  local -n _out=$1
  _out=()
  local idx
  for idx in "${SELECTED_INDICES[@]}"; do
    _out+=("$(compose_service_for_app "${APP_IDS[$idx]}")")
  done
}

selected_app_names() {
  local names="" idx
  for idx in "${SELECTED_INDICES[@]}"; do
    names+="${APP_IDS[$idx]},"
  done
  echo "${names%,}"
}

app_is_running() { app_is_active "$1"; }
detect_running_apps() { detect_active_apps; }
running_app_names() { active_app_names; }

show_active_apps() {
  detect_active_apps
  if [[ ${#APP_IDS[@]} -eq 0 ]]; then
    warn "Tidak ada app di ${APPS_DIR}/"
    return 1
  fi
  if [[ ${#ACTIVE_INDICES[@]} -eq 0 ]]; then
    warn "Tidak ada container aktif (running/restarting/paused)."
    echo "  Jalankan Run dari menu untuk start app."
    return 1
  fi

  echo ""
  echo -e "${BOLD}Container aktif:${NC}"
  local slot idx app_port log_port cstatus cstatus_label
  for slot in "${!ACTIVE_INDICES[@]}"; do
    idx="${ACTIVE_INDICES[$slot]}"
    get_effective_ports "$idx" app_port log_port
    local port_info="app :${app_port}  viewer :${log_port}"
    [[ "$app_port" == "0" ]] && port_info="(no HTTP)  viewer :${log_port}"
    cstatus="$(container_status_for_app "$idx")"
    cstatus_label="$(format_state_label "$cstatus")"
    echo -e "  [$((slot + 1))] ${APP_IDS[$idx]} — ${cstatus_label}"
    echo "       port   : ${port_info}"
  done
  echo ""
  return 0
}

show_running_apps() { show_active_apps; }

select_active_apps_interactive() {
  local action="${1:-}"
  show_active_apps || return 1

  if [[ ${#ACTIVE_INDICES[@]} -eq 1 ]]; then
    SELECTED_INDICES=("${ACTIVE_INDICES[0]}")
    info "Satu container aktif — otomatis: ${APP_IDS[${ACTIVE_INDICES[0]}]}"
    return 0
  fi

  echo "  Ketik: all | satu nomor (2) | beberapa (1,3)"
  local input
  input="$(prompt "Pilih app${action:+ untuk ${action}}" "all")"
  input="$(echo "$input" | tr '[:upper:]' '[:lower:]' | xargs)"

  SELECTED_INDICES=()
  if [[ "$input" == "all" || "$input" == "*" ]]; then
    SELECTED_INDICES=("${ACTIVE_INDICES[@]}")
    return 0
  fi

  local part slot
  IFS=',' read -ra parts <<<"$input"
  for part in "${parts[@]}"; do
    part="$(echo "$part" | xargs)"
    if ! [[ "$part" =~ ^[0-9]+$ ]] || (( part < 1 || part > ${#ACTIVE_INDICES[@]} )); then
      err "Nomor tidak valid: ${part} (hanya container aktif ditampilkan)"
      return 1
    fi
    slot=$((part - 1))
    SELECTED_INDICES+=("${ACTIVE_INDICES[$slot]}")
  done

  if [[ ${#SELECTED_INDICES[@]} -eq 0 ]]; then
    err "Tidak ada app dipilih"
    return 1
  fi

  dedupe_indices
  return 0
}

select_running_apps_interactive() { select_active_apps_interactive "$@"; }

select_apps_interactive() {
  local action="${1:-}"
  detect_apps
  show_detected_apps || return 1

  if [[ ${#APP_IDS[@]} -eq 1 ]]; then
    SELECTED_INDICES=(0)
    info "Satu app — otomatis dipilih: ${APP_IDS[0]}"
    return 0
  fi

  if [[ "${RUN_ALL:-0}" == "1" || "${SELECT_ALL:-0}" == "1" ]]; then
    SELECTED_INDICES=()
    local i
    for i in "${!APP_IDS[@]}"; do
      SELECTED_INDICES+=("$i")
    done
    return 0
  fi

  echo ""
  echo "  Ketik: all | satu nomor (2) | beberapa (1,3)"
  local input
  input="$(prompt "Pilih app${action:+ untuk ${action}}" "all")"
  input="$(echo "$input" | tr '[:upper:]' '[:lower:]' | xargs)"

  SELECTED_INDICES=()
  if [[ "$input" == "all" || "$input" == "*" ]]; then
    local i
    for i in "${!APP_IDS[@]}"; do
      SELECTED_INDICES+=("$i")
    done
    return 0
  fi

  local part
  IFS=',' read -ra parts <<<"$input"
  for part in "${parts[@]}"; do
    part="$(echo "$part" | xargs)"
    if ! [[ "$part" =~ ^[0-9]+$ ]] || (( part < 1 || part > ${#APP_IDS[@]} )); then
      err "Nomor tidak valid: ${part}"
      return 1
    fi
    SELECTED_INDICES+=("$((part - 1))")
  done

  if [[ ${#SELECTED_INDICES[@]} -eq 0 ]]; then
    err "Tidak ada app dipilih"
    return 1
  fi

  dedupe_indices
  return 0
}

confirm_selection() {
  local action="$1"
  if [[ "${RUN_ALL:-0}" == "1" ]]; then
    return 0
  fi

  echo ""
  echo -e "${BOLD}Konfirmasi ${action}:${NC}"
  local idx app_port log_port cstatus cstatus_label
  for idx in "${SELECTED_INDICES[@]}"; do
    get_effective_ports "$idx" app_port log_port
    cstatus="$(container_status_for_app "$idx")"
    cstatus_label="$(format_state_label "$cstatus")"
    echo -e "  • ${APP_IDS[$idx]} — ${cstatus_label}"
    if [[ "$app_port" == "0" ]]; then
      echo "    viewer : http://localhost:${log_port}/logs"
    else
      echo "    app    : http://localhost:${app_port}"
      echo "    viewer : http://localhost:${log_port}/logs"
    fi
  done
  echo ""

  if ! prompt_yes_no "Lanjutkan?" "y"; then
    warn "Dibatalkan."
    return 1
  fi
  return 0
}

# ─── First-run wizard ─────────────────────────────────────────────────────────
write_default_env() {
  local log_password="$1"
  cat >"$ENV_FILE" <<EOF
# QoLauncher — generated by launcher.sh
# Per-app overrides: apps/<app-id>/.env

APP_ARGS=
APP_PORT=8080
APP_WORKDIR=/app

APP_RESTART_POLICY=on-failure
APP_RESTART_DELAY=3s
APP_MAX_RESTART=0
APP_RESTART_WINDOW=60s
APP_RESTART_BURST=10
APP_SHUTDOWN_TIMEOUT=30s

LOG_DIR=/var/log/qolauncher
LOG_RETENTION_DAYS=30
LOG_LEVEL=info
TZ=UTC

LOG_PORT=8081
LOG_USERNAME=admin
LOG_PASSWORD=${log_password}
VIEWER_ENABLED=true

HEALTHCHECK_ENABLED=true
HEALTHCHECK_TYPE=http
HEALTHCHECK_URL=http://127.0.0.1:8080/health
HEALTHCHECK_INTERVAL=30s
HEALTHCHECK_TIMEOUT=5s
HEALTHCHECK_FAILURES=3
EOF
  ok "Dibuat: ${ENV_FILE}"
}

get_effective_restart_policy() {
  local idx="$1"
  local env_f="${APP_ENV_FILES[$idx]:-}"
  local policy
  policy="$(read_env_value "$ENV_FILE" APP_RESTART_POLICY "on-failure")"
  if [[ -n "$env_f" && -f "$env_f" ]] && grep -q '^APP_RESTART_POLICY=' "$env_f" 2>/dev/null; then
    policy="$(read_env_value "$env_f" APP_RESTART_POLICY "$policy")"
  fi
  echo "$policy"
}

docker_restart_for_policy() {
  local policy="$1"
  case "$policy" in
    never)  echo "no" ;;
    always) echo "always" ;;
    *)      echo "unless-stopped" ;;
  esac
}

write_compose_multi() {
  local idx app_id app_dir binary env_f svc vol_binary vol_logs
  local app_port log_port rel_env docker_restart restart_policy

  {
    echo "# Generated by launcher.sh — do not edit manual (gunakan ./launcher.sh → Setup)"
    echo "services:"
  } >"$COMPOSE_FILE"

  for idx in "${DEPLOY_INDICES[@]}"; do
    app_id="${APP_IDS[$idx]}"
    app_dir="${APP_DIRS[$idx]}"
    binary="${APP_BINARIES[$idx]}"
    env_f="${APP_ENV_FILES[$idx]:-}"
    svc="$(service_name "$app_id")"

    get_effective_ports "$idx" app_port log_port
    restart_policy="$(get_effective_restart_policy "$idx")"
    docker_restart="$(docker_restart_for_policy "$restart_policy")"
    mkdir_safe "${LOGS_DIR}/${app_id}"

    if [[ "${app_dir}/${binary}" == "${ROOT_DIR}/"* ]]; then
      vol_binary="./${app_dir#${ROOT_DIR}/}/${binary}"
    else
      vol_binary="${app_dir}/${binary}"
    fi
    vol_binary="$(normalize_relpath "$vol_binary")"
    vol_logs="./logs/${app_id}"

    {
      echo "  ${svc}:"
      echo "    build:"
      echo "      context: ."
      echo "      args:"
      echo "        VERSION: \${VERSION:-0.1.0-dev}"
      echo "    image: ${IMAGE_NAME}"
      echo "    container_name: ${svc}"
      echo "    restart: ${docker_restart}"
      echo "    env_file:"
      echo "      - .env"
      if [[ -n "$env_f" && -f "$env_f" ]]; then
        if [[ "$env_f" == "${ROOT_DIR}/"* ]]; then
          rel_env="./${env_f#${ROOT_DIR}/}"
        else
          rel_env="$env_f"
        fi
        rel_env="$(normalize_relpath "$rel_env")"
        echo "      - ${rel_env}"
      fi
      echo "    environment:"
      echo "      APP_BINARY: /app/${binary}"
      if [[ "$app_port" != "0" ]]; then
        echo "      APP_PORT: \"${app_port}\""
      fi
      echo "      LOG_PORT: \"${log_port}\""
      echo "    volumes:"
      echo "      - ${vol_binary}:/app/${binary}:ro"
      echo "      - ${vol_logs}:/var/log/qolauncher"
      echo "    ports:"
      if [[ "$app_port" != "0" ]]; then
        echo "      - \"${app_port}:${app_port}\""
      fi
      echo "      - \"${log_port}:${log_port}\""
      echo "    stop_grace_period: 35s"
      echo ""
    } >>"$COMPOSE_FILE"
  done

  ok "Dibuat: ${COMPOSE_FILE} (${#DEPLOY_INDICES[@]} service)"
}

first_run_wizard() {
  echo ""
  echo -e "${BOLD}╔══════════════════════════════════════╗${NC}"
  echo -e "${BOLD}║   QoLauncher — Setup Awal            ║${NC}"
  echo -e "${BOLD}╚══════════════════════════════════════╝${NC}"
  echo ""
  info "Belum ada konfigurasi. Kita buat .env global (shared) untuk launcher."
  echo ""

  mkdir_safe "$APPS_DIR" "$LAUNCHER_DIR" "$LOGS_DIR"

  local log_user log_pass app_port log_port restart_policy
  log_user="$(prompt "LOG_USERNAME (viewer, shared)" "admin")"
  log_pass="$(prompt_secret "LOG_PASSWORD (viewer, shared)" "admin")"
  app_port="$(prompt "APP_PORT default (jika app tanpa .env sendiri)" "8080")"
  log_port="$(prompt "LOG_PORT default (jika app tanpa .env sendiri)" "8081")"
  restart_policy="$(prompt "APP_RESTART_POLICY default" "on-failure")"

  write_default_env "$log_pass"

  sed -i "s/^LOG_USERNAME=.*/LOG_USERNAME=${log_user}/" "$ENV_FILE"
  sed -i "s/^APP_PORT=.*/APP_PORT=${app_port}/" "$ENV_FILE"
  sed -i "s/^LOG_PORT=.*/LOG_PORT=${log_port}/" "$ENV_FILE"
  sed -i "s/^APP_RESTART_POLICY=.*/APP_RESTART_POLICY=${restart_policy}/" "$ENV_FILE"

  detect_apps
  if [[ ${#APP_IDS[@]} -gt 0 ]]; then
    info "Tip: multi-app butuh port unik per apps/<id>/.env (lihat apps/README.md)"
  else
    warn "Belum ada binary di apps/. Copy binary lalu pilih Run dari menu."
  fi

  ok "Setup selesai. Pilih Run dari menu untuk menjalankan."
}

ensure_global_env() {
  if [[ -f "$ENV_FILE" ]]; then
    return 0
  fi
  if [[ -f "${ROOT_DIR}/.env.example" && "${RUN_ALL:-0}" == "1" ]]; then
    cp "${ROOT_DIR}/.env.example" "$ENV_FILE"
    ok "Disalin .env.example → .env"
    return 0
  fi
  first_run_wizard
}

validate_global_env() {
  [[ -f "$ENV_FILE" ]] || return 0
  local user pass viewer
  user="$(read_env_value "$ENV_FILE" LOG_USERNAME "")"
  pass="$(read_env_value "$ENV_FILE" LOG_PASSWORD "")"
  viewer="$(read_env_value "$ENV_FILE" VIEWER_ENABLED "true")"
  viewer="$(echo "$viewer" | tr '[:upper:]' '[:lower:]')"
  if [[ "$viewer" == "true" && ( -z "$user" || -z "$pass" ) ]]; then
    err "LOG_USERNAME / LOG_PASSWORD wajib di ${ENV_FILE} (VIEWER_ENABLED=true)"
    err "Jalankan Setup (menu 7) atau perbaiki file — password harus satu baris: LOG_PASSWORD=secret"
    return 1
  fi
  return 0
}

ensure_compose_all() {
  detect_apps
  if [[ ${#APP_IDS[@]} -eq 0 ]]; then
    err "Tidak ada app di apps/"
    return 1
  fi
  DEPLOY_INDICES=()
  local i
  for i in "${!APP_IDS[@]}"; do
    DEPLOY_INDICES+=("$i")
  done
  if ! validate_deploy_ports; then
    return 1
  fi
  write_compose_multi
  save_state_multi
  return 0
}

prepare_deploy() {
  ensure_global_env
  if ! validate_global_env; then return 1; fi

  if ! select_apps_interactive "run"; then
    return 1
  fi

  if ! ensure_compose_all; then
    return 1
  fi

  if ! confirm_selection "run"; then
    return 1
  fi

  return 0
}

# ─── Docker actions ───────────────────────────────────────────────────────────
ensure_image() {
  info "Membangun image Docker (jika belum ada)..."
  docker_compose build --quiet 2>/dev/null || docker_compose build
}

cmd_run() {
  if ! prepare_deploy; then return 1; fi
  ensure_image

  local services=()
  indices_to_services services

  info "Menjalankan ${#SELECTED_INDICES[@]} container di background..."
  docker_compose up -d --remove-orphans "${services[@]}" || {
    err "Gagal start container — cek port bentrok (docker compose ps -a)"
    return 1
  }

  echo ""
  ok "Container berjalan di background."
  local idx app_port log_port
  for idx in "${SELECTED_INDICES[@]}"; do
    get_effective_ports "$idx" app_port log_port
    echo "  ${APP_IDS[$idx]}:"
    echo "    logs   : ${LOGS_DIR}/${APP_IDS[$idx]}/"
    if [[ "$app_port" != "0" ]]; then
      echo "    App    : http://localhost:${app_port}"
    fi
    echo "    Viewer : http://localhost:${log_port}/logs"
  done
  echo ""
  info "Launcher bisa ditutup — container tetap jalan."
}

cmd_stop() {
  if [[ ! -f "$COMPOSE_FILE" ]]; then
    warn "docker-compose.yml belum ada. Nothing to stop."
    return 0
  fi

  if [[ "${STOP_ALL:-0}" == "1" ]]; then
    detect_active_apps
    if [[ ${#ACTIVE_INDICES[@]} -eq 0 ]]; then
      info "Tidak ada container aktif."
      docker_compose down --remove-orphans 2>/dev/null || true
      ok "Selesai."
      return 0
    fi
    local services=()
    SELECTED_INDICES=("${ACTIVE_INDICES[@]}")
    indices_to_services services
    info "Menghentikan container aktif: $(active_app_names)"
    docker_compose stop "${services[@]}"
    ok "Container dihentikan."
    return 0
  fi

  if ! select_active_apps_interactive "stop"; then return 1; fi
  if ! confirm_selection "stop"; then return 1; fi

  local services=()
  indices_to_services services
  info "Menghentikan: $(selected_app_names)"
  docker_compose stop "${services[@]}"
  ok "Container dihentikan: $(selected_app_names)"
}

cmd_restart() {
  if [[ ! -f "$COMPOSE_FILE" ]]; then
    err "Jalankan Run dulu atau setup awal."
    return 1
  fi

  if ! select_active_apps_interactive "restart"; then return 1; fi
  if ! confirm_selection "restart"; then return 1; fi

  local services=()
  indices_to_services services
  info "Restart: $(selected_app_names)"
  docker_compose restart "${services[@]}"
  ok "Container di-restart: $(selected_app_names)"
}

cmd_status() {
  if [[ ! -f "$COMPOSE_FILE" ]]; then
    warn "Belum ada deployment."
    detect_apps
    [[ ${#APP_IDS[@]} -gt 0 ]] && show_detected_apps || true
    return 0
  fi

  detect_apps
  echo ""
  echo -e "${BOLD}Status container:${NC}"
  local i cstatus cstatus_label app_port log_port
  for i in "${!APP_IDS[@]}"; do
    cstatus="$(container_status_for_app "$i")"
    cstatus_label="$(format_state_label "$cstatus")"
    get_effective_ports "$i" app_port log_port
    echo -e "  ${APP_IDS[$i]} — ${cstatus_label}"
    if [[ "$cstatus" == "running" || "$cstatus" == "restarting" || "$cstatus" == "paused" ]]; then
      if [[ "$app_port" != "0" ]]; then
        echo "       app    : http://localhost:${app_port}"
      fi
      echo "       viewer : http://localhost:${log_port}/logs"
    fi
  done

  load_state
  [[ -n "${DEPLOYED_IDS:-}" ]] && echo "" && echo "  Compose  : ${DEPLOYED_IDS}"
  echo ""
  docker_compose ps -a
}

cmd_logs() {
  if [[ ! -f "$COMPOSE_FILE" ]]; then
    err "Belum ada deployment."
    return 1
  fi

  if ! select_active_apps_interactive "logs"; then return 1; fi

  local services=()
  indices_to_services services
  info "Log container: $(selected_app_names) (Ctrl+C untuk keluar)..."
  docker_compose logs -f --tail=100 "${services[@]}"
}

cmd_configure() {
  if [[ -f "$ENV_FILE" ]]; then
    warn "File .env sudah ada — wizard akan menimpa nilai dasar."
    if ! prompt_yes_no "Lanjutkan re-configure?" "n"; then
      return 0
    fi
  fi
  first_run_wizard
}

cmd_list_apps() {
  detect_apps
  show_detected_apps || true
}

# ─── Main menu ────────────────────────────────────────────────────────────────
show_banner() {
  echo ""
  echo -e "${BOLD}╔══════════════════════════════════════╗${NC}"
  echo -e "${BOLD}║            QoLauncher                ║${NC}"
  echo -e "${BOLD}╚══════════════════════════════════════╝${NC}"
  local summary
  summary="$(active_app_summary)"
  if [[ -n "$summary" ]]; then
    echo -e "  Active: ${GREEN}${summary}${NC}"
  else
    echo -e "  Active: ${YELLOW}(tidak ada)${NC}"
  fi
}

show_menu() {
  echo ""
  echo "  1) Run      — deploy & jalankan (pilih satu/beberapa/semua)"
  echo "  2) Stop     — hentikan (pilih satu/beberapa/semua)"
  echo "  3) Restart  — restart (pilih satu/beberapa/semua)"
  echo "  4) Status   — lihat status container"
  echo "  5) Logs     — tail log container"
  echo "  6) Apps     — lihat app di folder apps/ (+ status container)"
  echo "  7) Setup    — buat/ulang .env global"
  echo "  0) Exit"
  echo ""
}

main_menu() {
  if [[ ! -f "$ENV_FILE" ]]; then
    first_run_wizard
  fi

  while true; do
    show_banner
    show_menu
    local choice
    choice="$(prompt "Pilih menu" "0")"
    case "$choice" in
      1) cmd_run ;;
      2) cmd_stop ;;
      3) cmd_restart ;;
      4) cmd_status ;;
      5) cmd_logs ;;
      6) cmd_list_apps ;;
      7) cmd_configure ;;
      0) info "Bye."; exit 0 ;;
      *) err "Pilihan tidak valid" ;;
    esac || true
  done
}

main() {
  need_cmd docker
  mkdir_safe "$APPS_DIR" "$LAUNCHER_DIR" "$LOGS_DIR"

  case "${1:-}" in
    --run-all)
      RUN_ALL=1
      ensure_global_env
      cmd_run
      ;;
    --stop)
      STOP_ALL=1
      cmd_stop
      ;;
    --help|-h)
      echo "Usage: ./launcher.sh [--run-all|--stop|--help]"
      echo ""
      echo "  (tanpa arg)  Menu interaktif"
      echo "  --run-all    Deploy semua app di apps/ tanpa prompt (untuk CI/Makefile)"
      echo "  --stop       Hentikan semua container"
      ;;
    "")
      main_menu
      ;;
    *)
      err "Argumen tidak dikenal: $1 (gunakan --help)"
      exit 1
      ;;
  esac
}

main "$@"
