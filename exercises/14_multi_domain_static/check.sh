#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"

IMAGE_NAME="${IMAGE_NAME:-nginxlings_ex14_multi_domain}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex14_multi_domain_run}"
HOST_PORT="${HOST_PORT:-18140}"

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

  "${DOCKER_BIN}" builder prune -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

command -v "${DOCKER_BIN}" >/dev/null 2>&1 || fail "docker not found in PATH."
command -v curl >/dev/null 2>&1 || fail "curl not found in PATH."

[ -f "${PREFIX}/nginx.conf" ] || fail "nginx.conf not found in ${PREFIX}"
[ -f "${PREFIX}/Dockerfile" ] || fail "Dockerfile not found in ${PREFIX}"
[ -f "${PREFIX}/sites/site1/index.html" ] || fail "sites/site1/index.html not found."
[ -f "${PREFIX}/sites/site2/index.html" ] || fail "sites/site2/index.html not found."

info "[1/6] Building Docker image..."
DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build -t "${IMAGE_NAME}" "${PREFIX}" >/dev/null 2>&1 \
  || fail "docker build failed."
BUILT_IMAGE="1"

info "[2/6] Testing nginx config inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! "${DOCKER_BIN}" run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}" || true
  fail "nginx -t failed."
fi
rm -f "${TMP_T}" >/dev/null 2>&1 || true
TMP_T=""
info "[OK] nginx -t succeeded."

info "[3/6] Starting container..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container."

sleep 1

info "[4/6] Checking site1.local response..."
SITE1_BODY="$(curl -s -H 'Host: site1.local' "http://127.0.0.1:${HOST_PORT}/" || true)"
echo "[DEBUG] GET / with Host=site1.local returned:"
printf '%s\n' "${SITE1_BODY}"
echo "${SITE1_BODY}" | grep -q "site1-static-ok" || fail "site1.local did not return the expected static content."

info "[5/6] Checking site2.local response..."
SITE2_BODY="$(curl -s -H 'Host: site2.local' "http://127.0.0.1:${HOST_PORT}/" || true)"
echo "[DEBUG] GET / with Host=site2.local returned:"
printf '%s\n' "${SITE2_BODY}"
echo "${SITE2_BODY}" | grep -q "site2-static-ok" || fail "site2.local did not return the expected static content."

info "[6/6] Validating effective config..."
CONF_T="$(mktemp)"
if ! "${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc "nginx -T 2>&1" > "${CONF_T}"; then
  echo "nginx -T output:"
  cat "${CONF_T}" || true
  fail "Could not read effective config with nginx -T."
fi

grep -qE 'server_name[[:space:]]+site1\.local;' "${CONF_T}" || fail "server_name site1.local; not found in effective config."
grep -qE 'server_name[[:space:]]+site2\.local;' "${CONF_T}" || fail "server_name site2.local; not found in effective config."
grep -qE 'root[[:space:]]+/usr/share/nginx/site1;' "${CONF_T}" || fail "root /usr/share/nginx/site1; not found in effective config."
grep -qE 'root[[:space:]]+/usr/share/nginx/site2;' "${CONF_T}" || fail "root /usr/share/nginx/site2; not found in effective config."

rm -f "${CONF_T}" >/dev/null 2>&1 || true
CONF_T=""

echo
echo "[PASS] 14 – Multiple Domains, Static Sites looks good."