#!/usr/bin/env bash
# Checks exercise 01: NGINX serves index.html from this directory on :8080.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

CONF="${EX_DIR}/nginx.conf"
PORT=8080
URL="http://127.0.0.1:${PORT}/"
EXPECTED_TEXT="Merhaba Nginxlings"

echo "[1/4] Checking prerequisites..."
require_cmd "${NGINX_BIN}" "install NGINX on this machine"
require_cmd curl
require_free_port "${PORT}" "the NGINX server in this exercise"
mkdir -p "${EX_DIR}/logs"

echo "[2/4] Testing nginx.conf with nginx -t..."
nginx_test "${CONF}" "${EX_DIR}"

echo "[3/4] Starting NGINX on port ${PORT}..."
start_host_nginx "${CONF}" "${EX_DIR}"
wait_host_nginx "${URL}"

echo "[4/4] Checking that / serves index.html from this directory..."
temp_file BODY
HTTP_CODE="$(curl -s -o "${BODY}" -w '%{http_code}' "${URL}" || true)"
if [[ "${HTTP_CODE}" != "200" ]]; then
  fail "Expected HTTP 200 from ${URL}, got ${HTTP_CODE}. Check root and index."
fi
if ! grep -q "${EXPECTED_TEXT}" "${EX_DIR}/index.html"; then
  fail "index.html does not contain '${EXPECTED_TEXT}' yet."
fi
# Comparing with the file proves the page is served from disk; a directive
# that only returns the text would not match the whole document.
if ! cmp -s "${BODY}" "${EX_DIR}/index.html"; then
  fail "The response to / is not the content of index.html. Serve the file with root and index instead of returning text."
fi

pass "01 – NGINX serves index.html from this directory on port ${PORT}."
