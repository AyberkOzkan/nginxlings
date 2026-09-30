#!/usr/bin/env bash
# Checks exercise 02: nginx.conf includes conf.d/*.conf and conf.d/default.conf serves a response.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

CONF="${EX_DIR}/nginx.conf"
DEFAULT_CONF="${EX_DIR}/conf.d/default.conf"
EXPECTED_TEXT="config-layout-ok"

echo "[1/4] Checking prerequisites..."
require_cmd "${NGINX_BIN}" "install NGINX on this machine"
require_cmd curl
[[ -f "${CONF}" ]] || fail "nginx.conf not found in ${EX_DIR}."
[[ -f "${DEFAULT_CONF}" ]] || fail "conf.d/default.conf not found."
mkdir -p "${EX_DIR}/logs"

echo "[2/4] Testing nginx.conf with nginx -t..."
nginx_test "${CONF}" "${EX_DIR}"

echo "[3/4] Inspecting the effective configuration (nginx -T)..."
temp_file DUMP
if ! "${NGINX_BIN}" -T -c "${CONF}" -p "${EX_DIR}" >"${DUMP}" 2>&1; then
  cat "${DUMP}"
  fail "nginx -T failed."
fi

# Effective configuration: no comments, one directive per line.
temp_file ACTIVE
active_config "${DUMP}" >"${ACTIVE}"
if ! grep -qE '^[[:space:]]*include[[:space:]]+([^;[:space:]]*/)?conf\.d/\*\.conf[[:space:]]*;' "${ACTIVE}"; then
  fail "nginx.conf does not load the .conf files in conf.d/ yet; the include belongs in the http block."
fi

# Only the part of the dump that comes from conf.d/default.conf.
temp_file DEFAULT_ACTIVE
awk -v file="${DEFAULT_CONF}" '
  /^# configuration file / { on = ($0 == "# configuration file " file ":"); next }
  on { print }
' "${DUMP}" | active_config >"${DEFAULT_ACTIVE}"
if ! grep -qE '^[[:space:]]*server[[:space:]]*\{' "${DEFAULT_ACTIVE}"; then
  fail "No active 'server { ... }' block found in conf.d/default.conf (is it loaded by the include, and not commented out?)."
fi
PORT="$(sed -nE 's/^[[:space:]]*listen[[:space:]]+([^;]*:)?([0-9]+)([[:space:]][^;]*)?;.*/\2/p' "${DEFAULT_ACTIVE}" | head -n 1)"
if [[ -z "${PORT}" ]]; then
  fail "No active 'listen <port>;' directive found in the server block of conf.d/default.conf."
fi

echo "[4/4] Starting NGINX on port ${PORT} and requesting /..."
require_free_port "${PORT}" "the server block in conf.d/default.conf"
start_host_nginx "${CONF}" "${EX_DIR}"
URL="http://127.0.0.1:${PORT}/"
wait_host_nginx "${URL}"
temp_file BODY
HTTP_CODE="$(curl -s -m 5 -o "${BODY}" -w '%{http_code}' "${URL}" || true)"
if [[ "${HTTP_CODE}" != "200" ]]; then
  fail "Expected HTTP 200 from ${URL}, got ${HTTP_CODE}."
fi
if ! grep -q "${EXPECTED_TEXT}" "${BODY}"; then
  fail "The response from ${URL} does not contain '${EXPECTED_TEXT}'. Return it from location / in conf.d/default.conf."
fi

pass "02 – conf.d/default.conf is included and answers '${EXPECTED_TEXT}' on port ${PORT}."
