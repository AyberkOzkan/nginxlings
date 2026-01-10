#!/usr/bin/env bash
set -euo pipefail

NGINX_BIN="${NGINX_BIN:-nginx}"
PREFIX="$(pwd)"
LOG_DIR="${PREFIX}/logs"
HTML_DIR="${PREFIX}/html"

mkdir -p "${LOG_DIR}"

fail() {
  echo "[FAIL] $*"
  exit 1
}

echo "[1/3] Testing nginx.conf with nginx -t..."

if [ ! -f "${PREFIX}/nginx.conf" ]; then
  fail "nginx.conf not found in ${PREFIX}"
fi

TMP_T_LOG="$(mktemp)"
if ! "${NGINX_BIN}" -t -c "${PREFIX}/nginx.conf" -p "${PREFIX}" >"${TMP_T_LOG}" 2>&1; then
  echo "nginx -t output:"
  cat "${TMP_T_LOG}"
  rm -f "${TMP_T_LOG}"
  fail "nginx -t failed."
fi
rm -f "${TMP_T_LOG}"
echo "[OK] nginx -t succeeded."

echo "[2/3] Verifying user directive in main context..."

CURRENT_USER="$(id -un)"
CURRENT_GROUP="$(id -gn)"

# Accept:
#   user CURRENT_USER;
#   user CURRENT_USER CURRENT_GROUP;
if ! grep -qE "^[[:space:]]*user[[:space:]]+${CURRENT_USER}([[:space:]]+${CURRENT_GROUP})?[[:space:]]*;" "${PREFIX}/nginx.conf"; then
  echo "Expected a user directive for the current user in nginx.conf."
  echo "For example:"
  echo "    user ${CURRENT_USER};"
  echo "or:"
  echo "    user ${CURRENT_USER} ${CURRENT_GROUP};"
  fail "user directive not found or does not match the current user."
fi
echo "[OK] user directive matches current user (${CURRENT_USER})."

echo "[3/3] Checking web root, index.html and ownership..."

if [ ! -d "${HTML_DIR}" ]; then
  fail "Expected web root directory: ${HTML_DIR}"
fi

if [ ! -f "${HTML_DIR}/index.html" ]; then
  fail "Expected index.html inside ${HTML_DIR}"
fi

# Ownership checks
DIR_OWNER="$(stat -c '%U' "${HTML_DIR}")"
FILE_OWNER="$(stat -c '%U' "${HTML_DIR}/index.html")"

if [ "${DIR_OWNER}" != "${CURRENT_USER}" ] || [ "${FILE_OWNER}" != "${CURRENT_USER}" ]; then
  echo "Directory owner: ${DIR_OWNER}"
  echo "index.html owner: ${FILE_OWNER}"
  echo "Expected owner for both: ${CURRENT_USER}"
  fail "html/ and index.html must be owned by the current user."
fi

# Root directive should point to html/ (relative or absolute)
if ! grep -qE "root[[:space:]]+html[[:space:]]*;" "${PREFIX}/nginx.conf" && \
   ! grep -qE "root[[:space:]]+${HTML_DIR}[[:space:]]*;" "${PREFIX}/nginx.conf"; then
  echo "Tip: inside your server block use something like:"
  echo "    root html;"
  fail "Could not find a root directive pointing to the html/ directory."
fi

echo
echo "[PASS] 04 – Worker User & Permissions looks good."

