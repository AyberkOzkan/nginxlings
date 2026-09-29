#!/usr/bin/env bash
# Shared helpers for exercises/<id>/check.sh scripts.
#
# Source it at the top of a check script:
#
#   LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../lib" && pwd)"
#   # shellcheck source=../lib/check-lib.sh
#   . "${LIB_DIR}/check-lib.sh"
#
# Contract for every check script:
# - exit 0 only when the exercise passes;
# - the last output line is a "[PASS] ..." or "[FAIL] ..." summary, because
#   the TUI shows that line as the status;
# - everything the check starts (processes, containers, images, networks,
#   temporary files) is removed on every exit path, including Ctrl+C and the
#   TUI's SIGTERM on quit or timeout.

set -Eeuo pipefail

# ---------------------------------------------------------------------------
# Result lines
# ---------------------------------------------------------------------------

# fail prints the final [FAIL] summary and exits with status 1.
fail() {
  printf '[FAIL] %s\n' "$*"
  exit 1
}

# pass prints the final [PASS] summary. The script should exit 0 after it.
pass() {
  printf '[PASS] %s\n' "$*"
}

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

_check_exit_hooks=()

# on_exit registers a shell command that runs when the script exits for any
# reason. Hooks run in reverse registration order, and a failing hook does
# not stop the others.
on_exit() {
  _check_exit_hooks+=("$*")
}

_check_run_exit_hooks() {
  local code=$?
  local i
  set +e
  trap - ERR
  for ((i = ${#_check_exit_hooks[@]} - 1; i >= 0; i--)); do
    eval "${_check_exit_hooks[i]}" >/dev/null 2>&1
  done
  exit "${code}"
}

# _check_on_error guarantees a [FAIL] line when a command fails under set -e
# without the script printing its own reason. Subshells (for example
# command substitutions) exit quietly so the message is printed once, by
# the main script.
_check_on_error() {
  local code=$1 line=$2 command=$3
  if [[ "${BASH_SUBSHELL}" -gt 0 ]]; then
    exit "${code}"
  fi
  printf '[FAIL] Unexpected error (exit %s) at check.sh line %s: %s\n' "${code}" "${line}" "${command}"
  exit "${code}"
}

trap '_check_run_exit_hooks' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap '_check_on_error "$?" "${LINENO}" "${BASH_COMMAND}"' ERR

# temp_file VAR creates a temporary file, stores its path in VAR, and removes
# it on exit. It sets a variable instead of printing the path because a
# command substitution would register the cleanup in a subshell.
temp_file() {
  local _check_tmp
  _check_tmp="$(mktemp)"
  on_exit "rm -f -- $(printf '%q' "${_check_tmp}")"
  printf -v "$1" '%s' "${_check_tmp}"
}

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------

# require_cmd CMD [WHY] fails with an actionable message when CMD is missing.
require_cmd() {
  local cmd=$1 why=${2:-}
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    if [[ -n "${why}" ]]; then
      fail "Missing prerequisite: '${cmd}' is not installed (${why})."
    fi
    fail "Missing prerequisite: '${cmd}' is not installed."
  fi
}

# require_docker checks the docker command and that the daemon is reachable.
require_docker() {
  local output
  require_cmd "${DOCKER_BIN:-docker}" "this exercise runs NGINX in a container"
  if ! output="$("${DOCKER_BIN:-docker}" info 2>&1 >/dev/null)"; then
    printf '%s\n' "${output}" | tail -n 3
    fail "Cannot reach the Docker daemon. Start Docker and make sure your user may use it (for example, membership in the docker group)."
  fi
}

# port_owner PORT prints the process listening on PORT when ss can see it.
port_owner() {
  local port=$1
  command -v ss >/dev/null 2>&1 || return 0
  ss -Hltnp "sport = :${port}" 2>/dev/null |
    sed -n 's/.*users:(("\([^"]*\)",pid=\([0-9]*\).*/\1 (pid \2)/p' |
    head -n 1
}

# port_in_use PORT succeeds when something already listens on TCP PORT.
port_in_use() {
  local port=$1
  if command -v ss >/dev/null 2>&1; then
    [[ -n "$(ss -Hltn "sport = :${port}" 2>/dev/null)" ]]
    return
  fi
  (exec 3<>"/dev/tcp/127.0.0.1/${port}") 2>/dev/null
}

# require_free_port PORT [PURPOSE] fails when PORT is already taken, naming
# the owner so the learner knows what to stop.
require_free_port() {
  local port=$1 purpose=${2:-this check} owner
  if port_in_use "${port}"; then
    owner="$(port_owner "${port}")"
    if [[ -n "${owner}" ]]; then
      fail "Port ${port} (needed by ${purpose}) is already in use by ${owner}. Stop that process and run the check again."
    fi
    fail "Port ${port} (needed by ${purpose}) is already in use. Stop the process listening on it and run the check again."
  fi
}

# ---------------------------------------------------------------------------
# Readiness
# ---------------------------------------------------------------------------

# wait_http URL [SECONDS] waits until URL answers with any HTTP status.
# Extra curl options (for example -k or -H) may follow the timeout.
wait_http() {
  local url=$1 seconds=${2:-15} deadline code
  shift $(($# < 2 ? $# : 2))
  deadline=$((SECONDS + seconds))
  while ((SECONDS < deadline)); do
    code="$(curl -s -o /dev/null -m 2 -w '%{http_code}' "$@" "${url}" || true)"
    if [[ -n "${code}" && "${code}" != "000" ]]; then
      return 0
    fi
    sleep 0.2
  done
  return 1
}

# wait_tcp HOST PORT [SECONDS] waits until a TCP connection succeeds.
wait_tcp() {
  local host=$1 port=$2 seconds=${3:-15} deadline
  deadline=$((SECONDS + seconds))
  while ((SECONDS < deadline)); do
    if (exec 3<>"/dev/tcp/${host}/${port}") 2>/dev/null; then
      return 0
    fi
    sleep 0.2
  done
  return 1
}

# ---------------------------------------------------------------------------
# Host NGINX
# ---------------------------------------------------------------------------

NGINX_BIN="${NGINX_BIN:-nginx}"
NGINX_PID=""
NGINX_STDERR=""

# fail_nginx_rejected FILE prints nginx -t/-T output (FILE, or - for stdin) and
# fails with NGINX's first [emerg] message, so the TUI status line shows where
# the error is, e.g. '"listen" directive is not allowed here in nginx.conf:31'.
fail_nginx_rejected() {
  local output first
  output="$(cat "$1")"
  printf '%s\n' "${output}"
  first="$(sed -nE '/\[emerg\]/{s/^.*\[emerg\] ([0-9]+#[0-9]+: )?//;p;q;}' <<<"${output}")"
  fail "NGINX rejected the configuration: ${first:-see the errors above.}"
}

# nginx_test CONF PREFIX runs nginx -t and fails with NGINX's first error.
nginx_test() {
  local conf=$1 prefix=$2 output
  if ! output="$("${NGINX_BIN}" -t -c "${conf}" -p "${prefix}" 2>&1)"; then
    fail_nginx_rejected - <<<"${output}"
  fi
}

# start_host_nginx CONF PREFIX starts NGINX in the foreground as a child of
# this script, so the check stops exactly the process it started instead of
# trusting a PID file that may point at an unrelated process.
start_host_nginx() {
  local conf=$1 prefix=$2
  temp_file NGINX_STDERR
  "${NGINX_BIN}" -c "${conf}" -p "${prefix}" -g 'daemon off;' 2>"${NGINX_STDERR}" &
  NGINX_PID=$!
  on_exit "stop_host_nginx"
}

# stop_host_nginx stops the NGINX started by start_host_nginx, if any.
stop_host_nginx() {
  local i
  [[ -n "${NGINX_PID}" ]] || return 0
  kill -TERM "${NGINX_PID}" 2>/dev/null || true
  for ((i = 0; i < 50; i++)); do
    kill -0 "${NGINX_PID}" 2>/dev/null || break
    sleep 0.1
  done
  kill -KILL "${NGINX_PID}" 2>/dev/null || true
  wait "${NGINX_PID}" 2>/dev/null || true
  NGINX_PID=""
}

# wait_host_nginx URL [SECONDS] waits for the started NGINX to answer and
# reports its stderr when it exits instead (for example, a failed bind).
wait_host_nginx() {
  local url=$1 seconds=${2:-10} deadline code
  deadline=$((SECONDS + seconds))
  while ((SECONDS < deadline)); do
    if ! kill -0 "${NGINX_PID}" 2>/dev/null; then
      cat "${NGINX_STDERR}" 2>/dev/null || true
      fail "NGINX exited during startup; see the errors above."
    fi
    code="$(curl -s -o /dev/null -m 2 -w '%{http_code}' "${url}" || true)"
    if [[ -n "${code}" && "${code}" != "000" ]]; then
      return 0
    fi
    sleep 0.2
  done
  cat "${NGINX_STDERR}" 2>/dev/null || true
  fail "NGINX did not answer on ${url} within ${seconds}s."
}

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

DOCKER_BIN="${DOCKER_BIN:-docker}"

# docker_build IMAGE CONTEXT builds an image labelled with DOCKER_LABEL and
# prints the end of the build log when the build fails.
docker_build() {
  local image=$1 context=$2 log
  temp_file log
  if ! "${DOCKER_BIN}" build --label "${DOCKER_LABEL}" -t "${image}" "${context}" >"${log}" 2>&1; then
    tail -n 40 "${log}"
    fail "docker build failed for ${image}; see the build output above."
  fi
}

# try_container_http CONTAINER URL [SECONDS] waits until URL answers while
# CONTAINER keeps running, and fails with the container logs if it exits.
# Extra curl options may follow the timeout. On timeout it prints the logs
# and returns 1, so the caller can say what the learner is probably missing.
try_container_http() {
  local container=$1 url=$2 seconds=${3:-20} deadline code
  shift $(($# < 3 ? $# : 3))
  deadline=$((SECONDS + seconds))
  while ((SECONDS < deadline)); do
    if [[ "$("${DOCKER_BIN}" inspect -f '{{.State.Running}}' "${container}" 2>/dev/null || true)" != "true" ]]; then
      echo "Container ${container} is not running. Logs:"
      "${DOCKER_BIN}" logs "${container}" 2>&1 | tail -n 40 || true
      fail "The container exited during startup; see the logs above."
    fi
    code="$(curl -s -o /dev/null -m 2 -w '%{http_code}' "$@" "${url}" || true)"
    if [[ -n "${code}" && "${code}" != "000" ]]; then
      return 0
    fi
    sleep 0.2
  done
  echo "Container logs:"
  "${DOCKER_BIN}" logs "${container}" 2>&1 | tail -n 40 || true
  return 1
}

# wait_container_http CONTAINER URL [SECONDS] is try_container_http with a
# generic failure on timeout.
wait_container_http() {
  try_container_http "$@" || fail "${2} did not answer within ${3:-20}s."
}
