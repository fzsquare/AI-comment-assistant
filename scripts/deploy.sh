#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.deploy}"
STATE_DIR="${STATE_DIR:-$ROOT_DIR/.deploy}"
LOG_DIR="$STATE_DIR/logs"
PID_DIR="$STATE_DIR/pids"
BIN_DIR="$STATE_DIR/bin"
RUNTIME_ENV="$STATE_DIR/runtime.env"

COMMAND="${1:-start}"

log() {
  printf '[deploy] %s\n' "$*"
}

die() {
  printf '[deploy] ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage: scripts/deploy.sh [command]

Commands:
  start     Install dependencies, build, and start all services (default)
  restart   Same as start
  stop      Stop services started by this script
  status    Show service status
  logs      Tail logs for all services
  install   Install Go, Node, and Python project dependencies
  build     Build backend and frontend

Useful env:
  ENV_FILE=.env.deploy        Optional deployment env file
  FRONTEND_PORT=8989          Public gateway/frontend port
  BACKEND_PORT=18989          Local-only Go backend port
  ALLOW_EMPTY_LLM_KEY=true    Allow UI/API startup without AI generation
  INIT_DB=true                Optionally create/import MySQL schema
  MIGRATE_DB=true             Run database/migrations/*.sql once (as root) for an existing DB
  LOAD_SEED=true              Import demo seed data when INIT_DB=true
  SMOKE_TEST=true             Check frontend gateway and management APIs after start
  SMOKE_TEST_AUTH=true        Force default/custom admin and merchant login checks
  SKIP_INSTALL=true           Reuse installed dependencies
EOF
}

truthy() {
  case "${1:-}" in
    1|true|TRUE|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

load_env_files() {
  mkdir -p "$LOG_DIR" "$PID_DIR" "$BIN_DIR"

  if [[ -f "$ENV_FILE" ]]; then
    log "loading $ENV_FILE"
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
  fi

  local configured_jwt_secret="${JWT_SECRET:-}"
  local configured_agent_token="${AGENT_INTERNAL_TOKEN:-}"
  if [[ -f "$RUNTIME_ENV" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$RUNTIME_ENV"
    set +a
  fi
  if [[ -n "$configured_jwt_secret" ]]; then
    JWT_SECRET="$configured_jwt_secret"
    export JWT_SECRET
  fi
  if [[ -n "$configured_agent_token" ]]; then
    AGENT_INTERNAL_TOKEN="$configured_agent_token"
    export AGENT_INTERNAL_TOKEN
  fi
}

generate_secret() {
  python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(48))
PY
}

ensure_secret() {
  local name="$1"
  local current="${!name:-}"
  local value quoted

  if [[ -n "$current" ]]; then
    return
  fi

  value="$(generate_secret)"
  export "$name=$value"
  printf -v quoted '%q' "$value"
  printf 'export %s=%s\n' "$name" "$quoted" >> "$RUNTIME_ENV"
  chmod 600 "$RUNTIME_ENV"
  log "generated $name in $RUNTIME_ENV"
}

configure_defaults() {
  FRONTEND_HOST="${FRONTEND_HOST:-0.0.0.0}"
  FRONTEND_PORT="${FRONTEND_PORT:-8989}"
  BACKEND_HOST="${BACKEND_HOST:-127.0.0.1}"
  BACKEND_PORT="${BACKEND_PORT:-18989}"
  AGENT_HOST="${AGENT_HOST:-127.0.0.1}"
  AGENT_PORT="${AGENT_PORT:-8090}"

  APP_ENV="${APP_ENV:-production}"
  APP_HOST="$BACKEND_HOST"
  APP_PORT="$BACKEND_PORT"
  AGENT_SERVICE_URL="${AGENT_SERVICE_URL:-http://$AGENT_HOST:$AGENT_PORT}"

  DB_NAME="${DB_NAME:-ppk}"
  MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
  MYSQL_PORT="${MYSQL_PORT:-3306}"
  DB_APP_USER="${DB_APP_USER:-ppk_app}"
  DB_APP_HOST="${DB_APP_HOST:-127.0.0.1}"
  DB_APP_PASSWORD="${DB_APP_PASSWORD:-}"
  MYSQL_ROOT_USER="${MYSQL_ROOT_USER:-root}"
  MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-}"

  case "${MYSQL_DSN:-}" in
    *replace-with*|*CHANGE_ME*|*change-me*)
      if [[ -n "$DB_APP_PASSWORD" ]]; then
        MYSQL_DSN=""
      fi
      ;;
  esac
  if [[ -z "${MYSQL_DSN:-}" ]]; then
    if [[ -n "$DB_APP_PASSWORD" ]]; then
      MYSQL_DSN="$DB_APP_USER:$DB_APP_PASSWORD@tcp($MYSQL_HOST:$MYSQL_PORT)/$DB_NAME?charset=utf8mb4&parseTime=True&loc=Local"
    fi
  fi

  if [[ -z "${ALLOWED_ORIGINS:-}" ]]; then
    ALLOWED_ORIGINS="http://127.0.0.1:$FRONTEND_PORT,http://localhost:$FRONTEND_PORT"
    if [[ -n "${PUBLIC_ORIGIN:-}" ]]; then
      ALLOWED_ORIGINS="$ALLOWED_ORIGINS,$PUBLIC_ORIGIN"
    fi
  fi

  # 商家上传图片：持久化到 .deploy/uploads，经网关 /uploads 反代访问。
  # 未配置规范域名时后端返回相对路径即可；配了 PUBLIC_ORIGIN 则用作图片绝对域名。
  UPLOAD_DIR="${UPLOAD_DIR:-$STATE_DIR/uploads}"
  PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-${PUBLIC_ORIGIN:-}}"

  LLM_API_KEY="${LLM_API_KEY:-}"
  ALLOW_EMPTY_LLM_KEY="${ALLOW_EMPTY_LLM_KEY:-false}"
  LLM_BASE_URL="${LLM_BASE_URL:-https://api.openai.com/v1}"
  LLM_MODEL="${LLM_MODEL:-gpt-5.4}"
  MIN_PASS_SCORE="${MIN_PASS_SCORE:-80}"
  MAX_REVISE_ROUNDS="${MAX_REVISE_ROUNDS:-2}"
  MAX_CONCURRENCY="${MAX_CONCURRENCY:-5}"
  AGENT_MIN_GRADE="${AGENT_MIN_GRADE:-B}"
  MAX_REVIEW_GENERATE_COUNT="${MAX_REVIEW_GENERATE_COUNT:-50}"
  DEFAULT_REVIEW_TARGET_COUNT="${DEFAULT_REVIEW_TARGET_COUNT:-10}"
  SMOKE_TEST="${SMOKE_TEST:-true}"
  SMOKE_TEST_AUTH="${SMOKE_TEST_AUTH:-auto}"
  SMOKE_ADMIN_ACCOUNT="${SMOKE_ADMIN_ACCOUNT:-admin}"
  SMOKE_ADMIN_PASSWORD="${SMOKE_ADMIN_PASSWORD:-123456}"
  SMOKE_MERCHANT_ACCOUNT="${SMOKE_MERCHANT_ACCOUNT:-merchant}"
  SMOKE_MERCHANT_PASSWORD="${SMOKE_MERCHANT_PASSWORD:-123456}"
}

ensure_runtime_secrets() {
  touch "$RUNTIME_ENV"
  chmod 600 "$RUNTIME_ENV"
  ensure_secret JWT_SECRET
  ensure_secret AGENT_INTERNAL_TOKEN
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "missing command: $1"
}

require_base_commands() {
  require_command python3
  require_command go
  require_command npm
}

mysql_cmd() {
  local args=(--protocol=tcp -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_ROOT_USER")
  if [[ -n "$MYSQL_ROOT_PASSWORD" ]]; then
    MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql "${args[@]}" "$@"
  else
    mysql "${args[@]}" "$@"
  fi
}

validate_mysql_identifier() {
  local label="$1"
  local value="$2"
  [[ "$value" =~ ^[A-Za-z0-9_]+$ ]] || die "$label must contain only letters, numbers, and underscore"
}

validate_mysql_host_pattern() {
  local label="$1"
  local value="$2"
  [[ "$value" =~ ^[A-Za-z0-9_.:%-]+$ ]] || die "$label must contain only letters, numbers, dot, underscore, percent, colon, and hyphen"
}

validate_config() {
  if [[ -z "${MYSQL_DSN:-}" ]]; then
    die "MYSQL_DSN is empty; set MYSQL_DSN, or set DB_APP_PASSWORD so the deploy script can build one from DB_* values"
  fi

  case "$MYSQL_DSN" in
    *replace-with*|*CHANGE_ME*|*change-me*)
      die "MYSQL_DSN still contains a placeholder; edit $ENV_FILE, leave MYSQL_DSN empty, or set DB_APP_PASSWORD for generated DSN"
      ;;
  esac
}

mysql_dsn_endpoint() {
  MYSQL_DSN="$MYSQL_DSN" python3 - <<'PY'
import os
import re
import sys

dsn = os.environ.get("MYSQL_DSN", "")
match = re.search(r"@tcp\(([^)]*)\)", dsn)
if not match:
    sys.exit(2)

address = match.group(1).strip()
if address.startswith("["):
    end = address.find("]")
    if end == -1:
        sys.exit(2)
    host = address[1:end]
    rest = address[end + 1 :]
    port = rest[1:] if rest.startswith(":") else "3306"
elif ":" in address:
    host, port = address.rsplit(":", 1)
else:
    host, port = address, "3306"

print((host or "127.0.0.1") + " " + (port or "3306"))
PY
}

mysql_dsn_info() {
  MYSQL_DSN="$MYSQL_DSN" python3 - <<'PY'
import os
import re
import sys
import urllib.parse

dsn = os.environ.get("MYSQL_DSN", "")
match = re.match(r"([^:@/]+)(?::.*)?@tcp\(([^)]*)\)/([^?]+)", dsn)
if not match:
    sys.exit(2)

user = urllib.parse.unquote(match.group(1))
address = match.group(2).strip()
database = urllib.parse.unquote(match.group(3))
if address.startswith("["):
    end = address.find("]")
    if end == -1:
        sys.exit(2)
    host = address[1:end]
    rest = address[end + 1 :]
    port = rest[1:] if rest.startswith(":") else "3306"
elif ":" in address:
    host, port = address.rsplit(":", 1)
else:
    host, port = address, "3306"

print(" ".join([user, host or "127.0.0.1", port or "3306", database]))
PY
}

validate_init_db_consistency() {
  local info dsn_user dsn_host dsn_port dsn_db

  if ! info="$(mysql_dsn_info)"; then
    die "MYSQL_DSN must use user:password@tcp(host:port)/database when INIT_DB=true"
  fi
  read -r dsn_user dsn_host dsn_port dsn_db <<< "$info"

  [[ "$dsn_user" == "$DB_APP_USER" ]] || die "MYSQL_DSN user ($dsn_user) must match DB_APP_USER ($DB_APP_USER) when INIT_DB=true"
  [[ "$dsn_db" == "$DB_NAME" ]] || die "MYSQL_DSN database ($dsn_db) must match DB_NAME ($DB_NAME) when INIT_DB=true"
  [[ "$dsn_host" == "$MYSQL_HOST" ]] || die "MYSQL_DSN host ($dsn_host) must match MYSQL_HOST ($MYSQL_HOST) when INIT_DB=true"
  [[ "$dsn_port" == "$MYSQL_PORT" ]] || die "MYSQL_DSN port ($dsn_port) must match MYSQL_PORT ($MYSQL_PORT) when INIT_DB=true"
}

check_mysql_reachable() {
  local endpoint host port

  if ! endpoint="$(mysql_dsn_endpoint)"; then
    printf '[deploy] ERROR: MYSQL_DSN must use tcp(host:port), got: %s\n' "$MYSQL_DSN" >&2
    return 1
  fi

  read -r host port <<< "$endpoint"
  if port_open "$host" "$port"; then
    log "MySQL TCP endpoint is reachable at $host:$port"
    return 0
  fi

  cat >&2 <<EOF
[deploy] ERROR: MySQL is not reachable at $host:$port from MYSQL_DSN.
[deploy]        Fix one of these before starting backend:
[deploy]        - start MySQL on this server, for example: sudo systemctl enable --now mysql
[deploy]        - install MySQL first if this is a fresh server
[deploy]        - change MYSQL_DSN in $ENV_FILE to a reachable MySQL host
[deploy]        - if the server is running but schema/user are missing, set INIT_DB=true and DB_APP_PASSWORD
EOF
  return 1
}

validate_runtime_config() {
  local failed=0

  if [[ -z "$LLM_API_KEY" ]] && ! truthy "$ALLOW_EMPTY_LLM_KEY"; then
    cat >&2 <<EOF
[deploy] ERROR: LLM_API_KEY is empty.
[deploy]        agent-service can start without it, but AI generation will return 503.
[deploy]        Set LLM_API_KEY in $ENV_FILE, or set ALLOW_EMPTY_LLM_KEY=true for UI/API-only startup.
EOF
    failed=1
  fi

  if ! check_mysql_reachable; then
    failed=1
  fi

  if ((failed != 0)); then
    die "preflight failed; fix $ENV_FILE or server dependencies, then rerun scripts/deploy.sh start"
  fi
}

ensure_migrations_table() {
  mysql_cmd "$DB_NAME" -e "CREATE TABLE IF NOT EXISTS schema_migrations (filename VARCHAR(255) PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);"
}

# 以 root 运行 database/migrations/*.sql，每个文件经 schema_migrations 去重只跑一次。
# 迁移本身用 information_schema 守卫，幂等，全新库/旧库均可安全运行。
apply_migrations() {
  local dir="$ROOT_DIR/database/migrations"
  [[ -d "$dir" ]] || return
  ensure_migrations_table
  local f base applied
  for f in "$dir"/*.sql; do
    [[ -e "$f" ]] || continue
    base="$(basename "$f")"
    applied="$(mysql_cmd "$DB_NAME" -N -B -e "SELECT COUNT(*) FROM schema_migrations WHERE filename = '$base';")"
    if [[ "$applied" == "0" ]]; then
      log "applying migration $base"
      mysql_cmd "$DB_NAME" < "$f"
      mysql_cmd "$DB_NAME" -e "INSERT INTO schema_migrations (filename) VALUES ('$base');"
    else
      log "migration $base already applied; skipping"
    fi
  done
}

# 关键顺序：schema（仅 INIT_DB）→ migrations → seed（仅 INIT_DB+LOAD_SEED）。
# migrations 夹在 schema 与 seed 之间，保证 seed 看到的是已迁移（含 uuid/type_id）的表。
prepare_database() {
  local do_init="false" do_migrate="false"
  truthy "${INIT_DB:-false}" && do_init="true"
  truthy "${MIGRATE_DB:-false}" && do_migrate="true"
  if [[ "$do_init" != "true" && "$do_migrate" != "true" ]]; then
    return
  fi
  require_command mysql
  validate_mysql_identifier DB_NAME "$DB_NAME"

  if [[ "$do_init" == "true" ]]; then
    [[ -n "$DB_APP_PASSWORD" ]] || die "INIT_DB=true requires DB_APP_PASSWORD"
    [[ "$DB_APP_PASSWORD" != *"'"* ]] || die "DB_APP_PASSWORD must not contain a single quote when INIT_DB=true"
    validate_mysql_identifier DB_APP_USER "$DB_APP_USER"
    validate_mysql_host_pattern DB_APP_HOST "$DB_APP_HOST"
    validate_init_db_consistency

    log "initializing MySQL database $DB_NAME"
    mysql_cmd -e "CREATE DATABASE IF NOT EXISTS \`$DB_NAME\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
    mysql_cmd -e "CREATE USER IF NOT EXISTS '$DB_APP_USER'@'$DB_APP_HOST' IDENTIFIED BY '$DB_APP_PASSWORD'; ALTER USER '$DB_APP_USER'@'$DB_APP_HOST' IDENTIFIED BY '$DB_APP_PASSWORD'; GRANT SELECT, INSERT, UPDATE, DELETE ON \`$DB_NAME\`.* TO '$DB_APP_USER'@'$DB_APP_HOST'; FLUSH PRIVILEGES;"
    mysql_cmd "$DB_NAME" < "$ROOT_DIR/database/schema.sql"
  fi

  # 迁移在 schema 之后、seed 之前；旧库在此补齐 uuid/type_id 等列
  apply_migrations

  if [[ "$do_init" == "true" ]]; then
    if truthy "${LOAD_SEED:-false}"; then
      mysql_cmd "$DB_NAME" < "$ROOT_DIR/database/seed.sql"
    else
      log "LOAD_SEED=false; demo admin/merchant accounts are not imported"
    fi
  fi
}

install_dependencies() {
  require_base_commands
  if truthy "${SKIP_INSTALL:-false}"; then
    log "SKIP_INSTALL=true; dependency installation skipped"
    return
  fi

  log "installing backend Go modules"
  (cd "$ROOT_DIR/backend" && go mod download)

  log "installing frontend npm dependencies"
  if [[ -f "$ROOT_DIR/frontend/package-lock.json" ]]; then
    (cd "$ROOT_DIR/frontend" && npm ci)
  else
    (cd "$ROOT_DIR/frontend" && npm install)
  fi

  log "installing agent-service Python dependencies"
  python3 -m venv "$ROOT_DIR/agent-service/.venv"
  "$ROOT_DIR/agent-service/.venv/bin/python" -m pip install -r "$ROOT_DIR/agent-service/requirements.txt"
}

build_project() {
  require_base_commands
  log "building frontend with same-origin /api"
  (cd "$ROOT_DIR/frontend" && VITE_API_BASE_URL=/api npm run build)

  log "building backend"
  (cd "$ROOT_DIR/backend" && go build -o "$BIN_DIR/ppk-server" ./cmd/server)
}

pid_file_for() {
  printf '%s/%s.pid' "$PID_DIR" "$1"
}

stop_one() {
  local name="$1"
  local pid_file pid
  pid_file="$(pid_file_for "$name")"
  [[ -f "$pid_file" ]] || return 0
  pid="$(cat "$pid_file")"
  if [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1; then
    log "stopping $name (pid $pid)"
    kill "$pid" >/dev/null 2>&1 || true
    for _ in {1..30}; do
      kill -0 "$pid" >/dev/null 2>&1 || break
      sleep 0.2
    done
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill -9 "$pid" >/dev/null 2>&1 || true
    fi
  fi
  rm -f "$pid_file"
}

stop_services() {
  stop_one gateway
  stop_one backend
  stop_one agent-service
}

port_open() {
  python3 - "$1" "$2" <<'PY'
import socket
import sys

host = sys.argv[1]
port = int(sys.argv[2])
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.settimeout(0.5)
    sys.exit(0 if sock.connect_ex((host, port)) == 0 else 1)
PY
}

wait_for_port() {
  local name="$1"
  local host="$2"
  local port="$3"
  local pid_file="$4"
  local pid

  for _ in {1..60}; do
    if port_open "$host" "$port"; then
      log "$name is listening on $host:$port"
      return
    fi
    if [[ -f "$pid_file" ]]; then
      pid="$(cat "$pid_file")"
      if [[ -n "$pid" ]] && ! kill -0 "$pid" >/dev/null 2>&1; then
        tail -n 80 "$LOG_DIR/$name.log" >&2 || true
        die "$name exited before opening $host:$port"
      fi
    fi
    sleep 0.5
  done
  tail -n 80 "$LOG_DIR/$name.log" >&2 || true
  die "$name did not open $host:$port"
}

run_smoke_tests() {
  if ! truthy "$SMOKE_TEST"; then
    log "SMOKE_TEST=false; deployment smoke tests skipped"
    return
  fi

  local base_url args
  base_url="http://127.0.0.1:$FRONTEND_PORT"
  args=(
    --base-url "$base_url"
    --admin-account "$SMOKE_ADMIN_ACCOUNT"
    --admin-password "$SMOKE_ADMIN_PASSWORD"
    --merchant-account "$SMOKE_MERCHANT_ACCOUNT"
    --merchant-password "$SMOKE_MERCHANT_PASSWORD"
  )

  case "$SMOKE_TEST_AUTH" in
    true|TRUE|1|yes|YES|on|ON)
      ;;
    false|FALSE|0|no|NO|off|OFF)
      args+=(--skip-authenticated)
      ;;
    auto|"")
      if ! truthy "${LOAD_SEED:-false}"; then
        args+=(--skip-authenticated)
        log "authenticated smoke tests skipped; set LOAD_SEED=true or SMOKE_TEST_AUTH=true to verify default/custom logins"
      fi
      ;;
    *)
      die "SMOKE_TEST_AUTH must be auto, true, or false"
      ;;
  esac

  log "running smoke tests against $base_url"
  python3 "$ROOT_DIR/scripts/check_frontend_flows.py" "${args[@]}"
}

ensure_port_free() {
  local name="$1"
  local host="$2"
  local port="$3"
  if port_open "$host" "$port"; then
    die "$name port already in use: $host:$port"
  fi
}

start_services() {
  [[ -x "$BIN_DIR/ppk-server" ]] || die "backend binary missing; run scripts/deploy.sh build"
  [[ -f "$ROOT_DIR/frontend/dist/index.html" ]] || die "frontend dist missing; run scripts/deploy.sh build"

  stop_services
  ensure_port_free agent-service "$AGENT_HOST" "$AGENT_PORT"
  ensure_port_free backend "$BACKEND_HOST" "$BACKEND_PORT"
  ensure_port_free gateway "127.0.0.1" "$FRONTEND_PORT"

  if [[ -z "$LLM_API_KEY" ]]; then
    log "warning: LLM_API_KEY is empty; agent-service starts, but generation will return 503 until configured"
  fi

  log "starting agent-service on $AGENT_HOST:$AGENT_PORT"
  (
    cd "$ROOT_DIR/agent-service"
    env \
      LLM_API_KEY="$LLM_API_KEY" \
      LLM_BASE_URL="$LLM_BASE_URL" \
      LLM_MODEL="$LLM_MODEL" \
      MIN_PASS_SCORE="$MIN_PASS_SCORE" \
      MAX_REVISE_ROUNDS="$MAX_REVISE_ROUNDS" \
      MAX_CONCURRENCY="$MAX_CONCURRENCY" \
      AGENT_HOST="$AGENT_HOST" \
      AGENT_PORT="$AGENT_PORT" \
      AGENT_INTERNAL_TOKEN="$AGENT_INTERNAL_TOKEN" \
      "$ROOT_DIR/agent-service/.venv/bin/python" -m app.main
  ) >> "$LOG_DIR/agent-service.log" 2>&1 &
  echo $! > "$(pid_file_for agent-service)"
  wait_for_port agent-service "$AGENT_HOST" "$AGENT_PORT" "$(pid_file_for agent-service)"

  log "starting backend on $BACKEND_HOST:$BACKEND_PORT"
  (
    cd "$ROOT_DIR"
    env \
      APP_ENV="$APP_ENV" \
      APP_HOST="$APP_HOST" \
      APP_PORT="$APP_PORT" \
      MYSQL_DSN="$MYSQL_DSN" \
      JWT_SECRET="$JWT_SECRET" \
      ALLOWED_ORIGINS="$ALLOWED_ORIGINS" \
      UPLOAD_DIR="$UPLOAD_DIR" \
      PUBLIC_BASE_URL="$PUBLIC_BASE_URL" \
      AGENT_SERVICE_URL="$AGENT_SERVICE_URL" \
      AGENT_INTERNAL_TOKEN="$AGENT_INTERNAL_TOKEN" \
      AGENT_MIN_GRADE="$AGENT_MIN_GRADE" \
      MAX_REVIEW_GENERATE_COUNT="$MAX_REVIEW_GENERATE_COUNT" \
      DEFAULT_REVIEW_TARGET_COUNT="$DEFAULT_REVIEW_TARGET_COUNT" \
      "$BIN_DIR/ppk-server"
  ) >> "$LOG_DIR/backend.log" 2>&1 &
  echo $! > "$(pid_file_for backend)"
  wait_for_port backend "$BACKEND_HOST" "$BACKEND_PORT" "$(pid_file_for backend)"

  log "starting public gateway on $FRONTEND_HOST:$FRONTEND_PORT"
  (
    cd "$ROOT_DIR"
    python3 "$ROOT_DIR/scripts/serve_gateway.py" \
      --host "$FRONTEND_HOST" \
      --port "$FRONTEND_PORT" \
      --dist "$ROOT_DIR/frontend/dist" \
      --backend "http://$BACKEND_HOST:$BACKEND_PORT"
  ) >> "$LOG_DIR/gateway.log" 2>&1 &
  echo $! > "$(pid_file_for gateway)"
  wait_for_port gateway "127.0.0.1" "$FRONTEND_PORT" "$(pid_file_for gateway)"

  run_smoke_tests

  log "deployment is up: http://127.0.0.1:$FRONTEND_PORT"
  log "public port: $FRONTEND_PORT; local backend: $BACKEND_HOST:$BACKEND_PORT; local agent: $AGENT_HOST:$AGENT_PORT"
}

status_services() {
  local name pid_file pid state
  for name in gateway backend agent-service; do
    pid_file="$(pid_file_for "$name")"
    state="stopped"
    if [[ -f "$pid_file" ]]; then
      pid="$(cat "$pid_file")"
      if [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1; then
        state="running (pid $pid)"
      fi
    fi
    printf '%-14s %s\n' "$name" "$state"
  done
}

tail_logs() {
  local files=()
  [[ -f "$LOG_DIR/gateway.log" ]] && files+=("$LOG_DIR/gateway.log")
  [[ -f "$LOG_DIR/backend.log" ]] && files+=("$LOG_DIR/backend.log")
  [[ -f "$LOG_DIR/agent-service.log" ]] && files+=("$LOG_DIR/agent-service.log")
  ((${#files[@]} > 0)) || die "no log files found in $LOG_DIR"
  tail -n 120 -f "${files[@]}"
}

case "$COMMAND" in
  -h|--help|help)
    usage
    exit 0
    ;;
esac

load_env_files

case "$COMMAND" in
  start|restart)
    require_base_commands
    configure_defaults
    validate_config
    validate_runtime_config
    ensure_runtime_secrets
    install_dependencies
    prepare_database
    build_project
    start_services
    ;;
  install)
    require_base_commands
    configure_defaults
    install_dependencies
    ;;
  build)
    require_base_commands
    configure_defaults
    install_dependencies
    build_project
    ;;
  stop)
    stop_services
    ;;
  status)
    status_services
    ;;
  logs)
    tail_logs
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage
    die "unknown command: $COMMAND"
    ;;
esac
