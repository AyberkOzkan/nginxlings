#!/usr/bin/env bash
set -euo pipefail

NGINX_BIN="${NGINX_BIN:-nginx}"
PREFIX="$(pwd)"
mkdir -p "${PREFIX}/logs"

fail() {
  echo "[FAIL] $*"
  exit 1
}

echo "[1/3] Testing nginx.conf with nginx -t..."

if [ ! -f "${PREFIX}/nginx.conf" ]; then
  fail "nginx.conf not found in current directory: ${PREFIX}"
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

echo "[2/3] Looking for 'include conf.d/*.conf;' in nginx -T output..."

TMP_T_CONF="$(mktemp)"
if ! "${NGINX_BIN}" -T -c "${PREFIX}/nginx.conf" -p "${PREFIX}" >"${TMP_T_CONF}" 2>&1; then
  echo "nginx -T output:"
  cat "${TMP_T_CONF}"
  rm -f "${TMP_T_CONF}"
  fail "nginx -T failed."
fi

if ! grep -qE '^[[:space:]]*include[[:space:]]+conf\.d/\*\.conf;' "${TMP_T_CONF}"; then
  echo "First lines of nginx -T output:"
  sed -n '1,80p' "${TMP_T_CONF}"
  rm -f "${TMP_T_CONF}"
  fail "The line 'include conf.d/*.conf;' was not found in the effective configuration."
fi
echo "[OK] 'include conf.d/*.conf;' found in effective configuration."

echo "[3/3] Checking for a listening server block in conf.d/default.conf..."

if [ ! -f "${PREFIX}/conf.d/default.conf" ]; then
  rm -f "${TMP_T_CONF}"
  fail "conf.d/default.conf not found."
fi

if ! grep -qE 'server[[:space:]]*\{' "${PREFIX}/conf.d/default.conf"; then
  rm -f "${TMP_T_CONF}"
  fail "No 'server { ... }' block found in conf.d/default.conf."
fi

if ! grep -qi 'listen' "${PREFIX}/conf.d/default.conf"; then
  rm -f "${TMP_T_CONF}"
  fail "No 'listen' directive found in conf.d/default.conf."
fi

rm -f "${TMP_T_CONF}"
echo "[OK] conf.d/default.conf contains a server block with a listen directive."

echo
echo "[PASS] 02 – Config Layout & Default Paths looks good."

