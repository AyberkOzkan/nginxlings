#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
EXERCISE_ID="13_include_modular_config"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"

IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex13_include_modular_run}"
HOST_PORT="${HOST_PORT:-18130}"

PREFIX="$(pwd)"

TMP_T=""
CONF_T=""
BUILT_IMAGE="0"

fail() {
  echo "[FAIL] $*"
  exit 1
}

info() {
  echo "$@"
}

cleanup() {
  set +e

  [ -n "${TMP_T}" ] && rm -f "${TMP_T}" >/dev/null 2>&1 || true
  [ -n "${CONF_T}" ] && rm -f "${CONF_T}" >/dev/null 2>&1 || true

  "${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

  if [ "${BUILT_IMAGE}" = "1" ]; then
    "${DOCKER_BIN}" image rm -f "${IMAGE_NAME}" >/dev/null 2>&1 || true
  fi

}
trap cleanup EXIT

command -v "${DOCKER_BIN}" >/dev/null 2>&1 || fail "docker not found in PATH."
command -v curl >/dev/null 2>&1 || fail "curl not found in PATH."

# Minimal expected files (student edits these)
[ -f "${PREFIX}/Dockerfile" ] || fail "Dockerfile not found in ${PREFIX}"
[ -f "${PREFIX}/nginx.conf" ] || fail "nginx.conf not found in ${PREFIX}"
[ -f "${PREFIX}/conf.d/10-server.conf" ] || fail "conf.d/10-server.conf not found"
[ -f "${PREFIX}/server.d/20-modular.conf" ] || fail "server.d/20-modular.conf not found"
[ -f "${PREFIX}/server.d/30-modular-extra.conf" ] || fail "server.d/30-modular-extra.conf not found"

info "[1/7] Building Docker image..."
DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build --label "${DOCKER_LABEL}" -t "${IMAGE_NAME}" "${PREFIX}" >/dev/null 2>&1 \
  || fail "docker build failed."
BUILT_IMAGE="1"

info "[2/7] Testing nginx config inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}" || true
  fail "nginx -t failed."
fi
rm -f "${TMP_T}" >/dev/null 2>&1 || true
TMP_T=""
info "[OK] nginx -t succeeded."

info "[3/7] Starting container..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container."

sleep 1

info "[4/7] Checking endpoints..."
body_root="$(curl -fsS "http://127.0.0.1:${HOST_PORT}/" || true)"
[ "${body_root}" = "base-ok" ] || {
  echo "[DEBUG] GET / returned:"
  printf '%s\n' "${body_root}"
  fail "Expected GET / to return exactly: base-ok"
}
info "[OK] GET / -> base-ok"

body_mod="$(curl -fsS "http://127.0.0.1:${HOST_PORT}/modular" || true)"
[ "${body_mod}" = "include-ok" ] || {
  echo "[DEBUG] GET /modular returned:"
  printf '%s\n' "${body_mod}"
  fail "Expected GET /modular to return exactly: include-ok"
}
info "[OK] GET /modular -> include-ok"

body_mod2="$(curl -fsS "http://127.0.0.1:${HOST_PORT}/modular2" || true)"
[ "${body_mod2}" = "include2-ok" ] || {
  echo "[DEBUG] GET /modular2 returned:"
  printf '%s\n' "${body_mod2}"
  fail "Expected GET /modular2 to return exactly: include2-ok"
}
info "[OK] GET /modular2 -> include2-ok"

info "[5/7] Extracting effective config (nginx -T)..."
CONF_T="$(mktemp)"
if ! "${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc "nginx -T 2>&1" > "${CONF_T}"; then
  echo "nginx -T output (first 160 lines):"
  sed -n '1,160p' "${CONF_T}" || true
  fail "Could not run nginx -T inside container."
fi

info "[6/7] Validating include directives in effective config..."

grep -qE '^[[:space:]]*include[[:space:]]+/etc/nginx/conf\.d/\*\.conf;' "${CONF_T}" || {
  echo "nginx -T (first 160 lines):"
  sed -n '1,160p' "${CONF_T}" || true
  fail "Missing: include /etc/nginx/conf.d/*.conf; (must be in http{})."
}

grep -qE '^[[:space:]]*include[[:space:]]+/etc/nginx/server\.d/\*\.conf;' "${CONF_T}" || {
  echo "nginx -T (first 200 lines):"
  sed -n '1,200p' "${CONF_T}" || true
  fail "Missing: include /etc/nginx/server.d/*.conf; (must be in server{})."
}

info "[OK] include directives found."

info "[7/7] Validating that modular files are actually loaded..."
grep -qE 'location[[:space:]]+/modular([[:space:]]*|\{)' "${CONF_T}" || {
  echo "nginx -T (search hint): /modular location not found."
  fail "Location /modular not found in effective config. Is server.d/20-modular.conf included?"
}

grep -qE 'location[[:space:]]+/modular2([[:space:]]*|\{)' "${CONF_T}" || {
  echo "nginx -T (search hint): /modular2 location not found."
  fail "Location /modular2 not found in effective config. Is server.d/30-modular-extra.conf included?"
}

rm -f "${CONF_T}" >/dev/null 2>&1 || true
CONF_T=""
info "[OK] modular locations found in effective config."

echo
echo "[PASS] 13 – Include & Modular Config Design looks good."