#!/usr/bin/env bash
# Checks exercise 03: nginx.conf has a valid context hierarchy and serves its response on :8080.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

CONF="${EX_DIR}/nginx.conf"
PORT=8080
URL="http://127.0.0.1:${PORT}/"
EXPECTED_TEXT="context-hierarchy-ok"

echo "[1/3] Checking prerequisites..."
require_cmd "${NGINX_BIN}" "install NGINX on this machine"
require_cmd curl
[[ -f "${CONF}" ]] || fail "nginx.conf not found in ${EX_DIR}."
require_free_port "${PORT}" "the NGINX server in this exercise"
mkdir -p "${EX_DIR}/logs"

echo "[2/3] Testing nginx.conf with nginx -t (it enforces the contexts)..."
temp_file T_OUTPUT
if ! "${NGINX_BIN}" -t -c "${CONF}" -p "${EX_DIR}" >"${T_OUTPUT}" 2>&1; then
  fail_nginx_rejected "${T_OUTPUT}"
fi

echo "[3/3] Starting NGINX on port ${PORT} and requesting /..."
start_host_nginx "${CONF}" "${EX_DIR}"
wait_host_nginx "${URL}"
temp_file BODY
HTTP_CODE="$(curl -s -m 5 -o "${BODY}" -w '%{http_code}' "${URL}" || true)"
if [[ "${HTTP_CODE}" != "200" ]]; then
  fail "Expected HTTP 200 from ${URL}, got ${HTTP_CODE}. Is the server listening on ${PORT} with a location /?"
fi
if ! grep -q "${EXPECTED_TEXT}" "${BODY}"; then
  fail "The response from ${URL} does not contain '${EXPECTED_TEXT}'. Keep the return in location /."
fi

pass "03 – Context Hierarchy: http > server > location answers on port ${PORT}."
