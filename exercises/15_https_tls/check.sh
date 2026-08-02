#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
EXERCISE_ID="15_https_tls"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"

IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex15_https_tls_run}"
HTTP_PORT="${HTTP_PORT:-18150}"
HTTPS_PORT="${HTTPS_PORT:-18450}"

PREFIX="$(pwd)"

TMP_BUILD=""
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

  [ -n "${TMP_BUILD}" ] && rm -f "${TMP_BUILD}" >/dev/null 2>&1 || true
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

[ -f "${PREFIX}/Dockerfile" ] || fail "Dockerfile not found in ${PREFIX}"
[ -f "${PREFIX}/nginx.conf" ] || fail "nginx.conf not found in ${PREFIX}"

info "[1/7] Building Docker image..."
TMP_BUILD="$(mktemp)"
if ! DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build --label "${DOCKER_LABEL}" -t "${IMAGE_NAME}" "${PREFIX}" >"${TMP_BUILD}" 2>&1; then
  echo "docker build output (first 120 lines):"
  sed -n '1,120p' "${TMP_BUILD}" || true
  fail "docker build failed."
fi
rm -f "${TMP_BUILD}" >/dev/null 2>&1 || true
TMP_BUILD=""
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
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d --name "${CONTAINER_NAME}" \
  -p "${HTTP_PORT}:80" \
  -p "${HTTPS_PORT}:443" \
  "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container."

sleep 2

info "[4/7] Verifying certificate files exist inside container..."
"${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc 'test -s /etc/nginx/certs/self.crt && test -s /etc/nginx/certs/self.key' \
  || fail "Certificate or key file not found at /etc/nginx/certs/."

info "[5/7] Checking HTTP -> HTTPS redirect..."
HTTP_HEADERS="$(curl -sSI -H 'Host: localhost' "http://127.0.0.1:${HTTP_PORT}/" || true)"
printf '%s\n' "${HTTP_HEADERS}"

printf '%s\n' "${HTTP_HEADERS}" | grep -Eq '^HTTP/[0-9.]+ 30[12378]' \
  || fail "HTTP endpoint did not return a redirect status."

printf '%s\n' "${HTTP_HEADERS}" | grep -qi '^Location: https://' \
  || fail "HTTP endpoint did not redirect to an https:// URL."

info "[6/7] Checking HTTPS response..."
HTTPS_BODY="$(curl -sk "https://127.0.0.1:${HTTPS_PORT}/" || true)"
printf '%s\n' "${HTTPS_BODY}"

printf '%s\n' "${HTTPS_BODY}" | grep -q 'tls-ok' \
  || fail "HTTPS endpoint did not return the expected body: tls-ok"

info "[7/7] Validating effective config..."
CONF_T="$(mktemp)"
if ! "${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc 'nginx -T 2>&1' > "${CONF_T}"; then
  echo "nginx -T output (first 200 lines):"
  sed -n '1,200p' "${CONF_T}" || true
  fail "Could not read effective config with nginx -T."
fi

grep -qE 'listen[[:space:]]+443[[:space:]]+ssl;' "${CONF_T}" \
  || fail "listen 443 ssl; not found in effective config."

grep -qE 'ssl_certificate[[:space:]]+/etc/nginx/certs/self\.crt;' "${CONF_T}" \
  || fail "ssl_certificate path is missing or incorrect."

grep -qE 'ssl_certificate_key[[:space:]]+/etc/nginx/certs/self\.key;' "${CONF_T}" \
  || fail "ssl_certificate_key path is missing or incorrect."

grep -qE 'listen[[:space:]]+80;' "${CONF_T}" \
  || fail "HTTP listen 80; not found in effective config."

grep -qE 'return[[:space:]]+301[[:space:]]+https://' "${CONF_T}" \
  || fail "HTTP to HTTPS redirect rule not found."

rm -f "${CONF_T}" >/dev/null 2>&1 || true
CONF_T=""

echo
echo "[PASS] 15 – HTTPS (TLS) Quick Lab looks good."