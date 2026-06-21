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
  INIT_DB=true                Optionally create/import MySQL schema
  LOAD_SEED=true              Import demo seed data when INIT_DB=true
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
  DB_APP_PASSWORD="${DB_APP_PASSWORD:-}"
  MYSQL_ROOT_USER="${MYSQL_ROOT_USER:-root}"
  MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-}"

  if [[ -z "${MYSQL_DSN:-}" ]]; then
    if [[ -n "$DB_APP_PASSWORD" ]]; then
      MYSQL_DSN="$DB_APP_USER:$DB_APP_PASSWORD@tcp($MYSQL_HOST:$MYSQL_PORT)/$DB_NAME?charset=utf8mb4&parseTime=True&loc=Local"
    else
      MYSQL_DSN="ppk_dev:ppk_dev_password@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"
    fi
  fi

  if [[ -z "${ALLOWED_ORIGINS:-}" ]]; then
    ALLOWED_ORIGINS="http://127.0.0.1:$FRONTEND_PORT,http://localhost:$FRONTEND_PORT"
    if [[ -n "${PUBLIC_ORIGIN:-}" ]]; then
      ALLOWED_ORIGINS="$ALLOWED_ORIGINS,$PUBLIC_ORIGIN"
    fi
  fi

  LLM_API_KEY="${LLM_API_KEY:-}"
  LLM_BASE_URL="${LLM_BASE_URL:-https://api.openai.com/v1}"
  LLM_MODEL="${LLM_MODEL:-gpt-5.4}"
  MIN_PASS_SCORE="${MIN_PASS_SCORE:-80}"
  MAX_REVISE_ROUNDS="${MAX_REVISE_ROUNDS:-2}"
  MAX_CONCURRENCY="${MAX_CONCURRENCY:-5}"
  AGENT_MIN_GRADE="${AGENT_MIN_GRADE:-B}"
  MAX_REVIEW_GENERATE_COUNT="${MAX_REVIEW_GENERATE_COUNT:-50}"
  DEFAULT_REVIEW_TARGET_COUNT="${DEFAULT_REVIEW_TARGET_COUNT:-10}"

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

validate_config() {
  case "$MYSQL_DSN" in
    *replace-with*|*CHANGE_ME*|*change-me*)
      die "MYSQL_DSN still contains a placeholder; edit $ENV_FILE or unset MYSQL_DSN to use local defaults"
      ;;
  esac
}

init_db_if_requested() {
  if ! truthy "${INIT_DB:-false}"; then
    return
  fi
  require_command mysql
  [[ -n "$DB_APP_PASSWORD" ]] || die "INIT_DB=true requires DB_APP_PASSWORD"
  [[ "$DB_APP_PASSWORD" != *"'"* ]] || die "DB_APP_PASSWORD must not contain a single quote when INIT_DB=true"
  validate_mysql_identifier DB_NAME "$DB_NAME"
  validate_mysql_identifier DB_APP_USER "$DB_APP_USER"

  log "initializing MySQL database $DB_NAME"
  mysql_cmd -e "CREATE DATABASE IF NOT EXISTS \`$DB_NAME\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
  mysql_cmd -e "CREATE USER IF NOT EXISTS '$DB_APP_USER'@'127.0.0.1' IDENTIFIED BY '$DB_APP_PASSWORD'; GRANT SELECT, INSERT, UPDATE, DELETE ON \`$DB_NAME\`.* TO '$DB_APP_USER'@'127.0.0.1'; FLUSH PRIVILEGES;"
  mysql_cmd "$DB_NAME" < "$ROOT_DIR/database/schema.sql"
  if truthy "${LOAD_SEED:-false}"; then
    mysql_cmd "$DB_NAME" < "$ROOT_DIR/database/seed.sql"
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

load_env_files

case "$COMMAND" in
  start|restart)
    configure_defaults
    validate_config
    install_dependencies
    init_db_if_requested
    build_project
    start_services
    ;;
  install)
    configure_defaults
    validate_config
    install_dependencies
    ;;
  build)
    configure_defaults
    validate_config
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
