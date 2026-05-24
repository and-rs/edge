#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"

load_env_file() {
  local env_file="$1"
  local line=""
  local key=""
  local value=""

  [[ -f "$env_file" ]] || return 0

  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*$ ]] && continue
    [[ "$line" =~ ^[[:space:]]*# ]] && continue

    line="${line#export }"
    key="${line%%=*}"
    value="${line#*=}"

    if [[ "$key" == "$line" ]]; then
      continue
    fi

    key="${key#"${key%%[![:space:]]*}"}"
    key="${key%"${key##*[![:space:]]}"}"

    if [[ -z "$key" || -n "${!key+x}" ]]; then
      continue
    fi

    if [[ "$value" =~ ^\".*\"$ || "$value" =~ ^\'.*\'$ ]]; then
      value="${value:1:-1}"
    fi

    export "$key=$value"
  done < "$env_file"
}

load_env_file "$repo_root/.env"
load_env_file "$repo_root/.env.local"

backend_addr="${1:-:8080}"
frontend_port="${2:-3000}"
backend_port="${backend_addr#:}"
backend_url="http://127.0.0.1:${backend_port}"
backend_bin=""
backend_pid=""
frontend_pid=""

cleanup() {
  local status=$?

  if [[ -n "$frontend_pid" ]]; then
    kill "$frontend_pid" 2>/dev/null || true
    wait "$frontend_pid" 2>/dev/null || true
  fi

  if [[ -n "$backend_pid" ]]; then
    kill "$backend_pid" 2>/dev/null || true
    wait "$backend_pid" 2>/dev/null || true
  fi

  if [[ -n "$backend_bin" && -f "$backend_bin" ]]; then
    rm -f "$backend_bin"
  fi

  exit "$status"
}

trap cleanup EXIT INT TERM

backend_bin="$(mktemp -t stint-backend)"
rm "$backend_bin"
(
  cd backend
  go build -o "$backend_bin" ./cmd/server
  exec env STINT_ADDR="$backend_addr" "$backend_bin"
) &
backend_pid=$!

(
  cd frontend
  exec env VITE_API_BASE_URL="$backend_url" bun x vinxi dev --port "$frontend_port"
) &
frontend_pid=$!

while kill -0 "$backend_pid" 2>/dev/null && kill -0 "$frontend_pid" 2>/dev/null; do
  sleep 1
done

set +e
if ! kill -0 "$backend_pid" 2>/dev/null; then
  wait "$backend_pid"
  status=$?
else
  wait "$frontend_pid"
  status=$?
fi
set -e

exit "$status"
