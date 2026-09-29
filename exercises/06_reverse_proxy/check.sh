#!/usr/bin/env bash
# Checks exercise 06: NGINX on :8081 proxies to the backend on :8080.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

CONF="${EX_DIR}/nginx.conf"
BACKEND_PORT=8080
PROXY_PORT=8081
BACKEND_URL="http://127.0.0.1:${BACKEND_PORT}/"
PROXY_URL="http://127.0.0.1:${PROXY_PORT}/"
BACKEND_PID=""

stop_backend() {
  [[ -n "${BACKEND_PID}" ]] || return 0
  kill "${BACKEND_PID}" 2>/dev/null || true
  wait "${BACKEND_PID}" 2>/dev/null || true
  BACKEND_PID=""
}

echo "[1/5] Checking prerequisites and files..."
require_cmd "${NGINX_BIN}" "install NGINX on this machine"
require_cmd curl
require_cmd python3 "needed to run backend_app.py"
[[ -f "${CONF}" ]] || fail "nginx.conf not found in ${EX_DIR}."
[[ -f "${EX_DIR}/backend_app.py" ]] || fail "backend_app.py not found in ${EX_DIR}."
require_free_port "${BACKEND_PORT}" "the backend app"
require_free_port "${PROXY_PORT}" "the NGINX reverse proxy"
mkdir -p "${EX_DIR}/logs"

# A random per-run token that only the backend process knows: it cannot be
# faked by returning a fixed string from nginx.conf.
TOKEN="$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')"

echo "[2/5] Starting the backend on 127.0.0.1:${BACKEND_PORT}..."
temp_file BACKEND_LOG
on_exit "stop_backend"
NGINXLINGS_BACKEND_TOKEN="${TOKEN}" python3 "${EX_DIR}/backend_app.py" >"${BACKEND_LOG}" 2>&1 &
BACKEND_PID=$!
if ! wait_tcp 127.0.0.1 "${BACKEND_PORT}" 10 || ! kill -0 "${BACKEND_PID}" 2>/dev/null; then
  cat "${BACKEND_LOG}" || true
  fail "The backend did not start on ${BACKEND_URL}."
fi
temp_file DIRECT
curl -s -m 5 -o "${DIRECT}" "${BACKEND_URL}" || true
if ! grep -q "${TOKEN}" "${DIRECT}"; then
  cat "${BACKEND_LOG}" || true
  fail "The process answering on ${BACKEND_URL} is not this exercise's backend."
fi

echo "[3/5] Testing nginx.conf with nginx -t..."
nginx_test "${CONF}" "${EX_DIR}"

echo "[4/5] Starting the NGINX reverse proxy on 127.0.0.1:${PROXY_PORT}..."
start_host_nginx "${CONF}" "${EX_DIR}"
wait_host_nginx "${PROXY_URL}"

echo "[5/5] Verifying that the response comes from the backend..."
temp_file BODY
HTTP_CODE="$(curl -s -m 5 -o "${BODY}" -w '%{http_code}' "${PROXY_URL}" || true)"
if ! kill -0 "${BACKEND_PID}" 2>/dev/null; then
  cat "${BACKEND_LOG}" || true
  fail "The backend stopped unexpectedly."
fi
if [[ "${HTTP_CODE}" != "200" ]]; then
  fail "Expected HTTP 200 from ${PROXY_URL}, got ${HTTP_CODE}. Is proxy_pass configured correctly?"
fi
if ! grep -q "${TOKEN}" "${BODY}"; then
  echo "Response via proxy:"
  cat "${BODY}"
  fail "The response via NGINX does not come from the backend on 127.0.0.1:${BACKEND_PORT}. Forward location / to it."
fi

pass "06 – HTTP Reverse Proxy Basics looks good."
