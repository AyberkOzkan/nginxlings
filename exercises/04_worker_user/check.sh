#!/usr/bin/env bash
# Checks exercise 04: user directive, html/ web root and ownership.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

CONF="${EX_DIR}/nginx.conf"
HTML_DIR="${EX_DIR}/html"

echo "[1/3] Testing nginx.conf with nginx -t..."
require_cmd "${NGINX_BIN}" "install NGINX on this machine"
[[ -f "${CONF}" ]] || fail "nginx.conf not found in ${EX_DIR}."
mkdir -p "${EX_DIR}/logs"
nginx_test "${CONF}" "${EX_DIR}"

echo "[2/3] Verifying the user and root directives in the effective configuration..."
temp_file DUMP
if ! "${NGINX_BIN}" -T -c "${CONF}" -p "${EX_DIR}" >"${DUMP}" 2>&1; then
  cat "${DUMP}"
  fail "nginx -T failed."
fi
temp_file ACTIVE
grep -v '^[[:space:]]*#' "${DUMP}" >"${ACTIVE}" || true

CURRENT_USER="$(id -un)"
CURRENT_GROUP="$(id -gn)"

# Accept "user CURRENT_USER;" and "user CURRENT_USER CURRENT_GROUP;".
if ! grep -qE "^[[:space:]]*user[[:space:]]+${CURRENT_USER}([[:space:]]+${CURRENT_GROUP})?[[:space:]]*;" "${ACTIVE}"; then
  echo "Expected an active user directive for the current user in nginx.conf, for example:"
  echo "    user ${CURRENT_USER};"
  echo "or:"
  echo "    user ${CURRENT_USER} ${CURRENT_GROUP};"
  fail "user directive not found (or commented out) or does not match the current user (${CURRENT_USER})."
fi

# root html; (relative to the prefix) or the absolute path of html/.
if ! grep -qE "^[[:space:]]*root[[:space:]]+(html|${HTML_DIR})/?[[:space:]]*;" "${ACTIVE}"; then
  echo "Tip: inside your server block use something like:"
  echo "    root html;"
  fail "No active root directive pointing to the html/ directory."
fi

echo "[3/3] Checking web root, index.html and ownership..."
[[ -d "${HTML_DIR}" ]] || fail "Expected web root directory: ${HTML_DIR}"
[[ -f "${HTML_DIR}/index.html" ]] || fail "Expected index.html inside ${HTML_DIR}"

DIR_OWNER="$(stat -c '%U' "${HTML_DIR}")"
FILE_OWNER="$(stat -c '%U' "${HTML_DIR}/index.html")"
if [[ "${DIR_OWNER}" != "${CURRENT_USER}" || "${FILE_OWNER}" != "${CURRENT_USER}" ]]; then
  echo "Directory owner: ${DIR_OWNER}"
  echo "index.html owner: ${FILE_OWNER}"
  echo "Expected owner for both: ${CURRENT_USER}"
  fail "html/ and index.html must be owned by the current user."
fi

pass "04 – Worker User & Permissions looks good."
