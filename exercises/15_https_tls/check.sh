#!/usr/bin/env bash
# Checks exercise 15: HTTP redirects to HTTPS and HTTPS answers with tls-ok.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="15_https_tls"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex15_https_tls_run}"
HTTP_PORT="${HTTP_PORT:-18150}"
HTTPS_PORT="${HTTPS_PORT:-18450}"

echo "[1/7] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in Dockerfile nginx.conf; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HTTP_PORT}" "the HTTP listener (override with HTTP_PORT)"
require_free_port "${HTTPS_PORT}" "the HTTPS listener (override with HTTPS_PORT)"

echo "[2/7] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/7] Testing nginx config and its listeners inside the image (nginx -T)..."
temp_file DUMP
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T >"${DUMP}" 2>&1; then
  fail_nginx_rejected "${DUMP}"
fi
# Effective configuration: no comments, one directive per line.
temp_file ACTIVE
active_config "${DUMP}" >"${ACTIVE}"
grep -qE '^[[:space:]]*listen[[:space:]]+([^;]*:)?80[[:space:];]' "${ACTIVE}" ||
  fail "No active 'listen 80;' yet. Add the HTTP server block that redirects to HTTPS."
grep -qE '^[[:space:]]*listen[[:space:]]+([^;]*:)?443[[:space:]]+([^;]*[[:space:]])?ssl[[:space:];]' "${ACTIVE}" ||
  fail "No active 'listen 443 ssl;' yet. Add the HTTPS server block."

echo "[4/7] Starting container ${CONTAINER_NAME} (host ${HTTP_PORT} -> 80, ${HTTPS_PORT} -> 443)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HTTP_PORT}:80" \
  -p "127.0.0.1:${HTTPS_PORT}:443" \
  "${IMAGE_NAME}" >/dev/null
if ! try_container_http "${CONTAINER_NAME}" "http://127.0.0.1:${HTTP_PORT}/" 10; then
  fail "Nothing answered on port 80. Add the HTTP server block with 'listen 80;' that redirects to HTTPS."
fi

echo "[5/7] Verifying certificate files exist inside the container..."
"${DOCKER_BIN}" exec "${CONTAINER_NAME}" test -s /etc/nginx/certs/self.crt ||
  fail "Certificate /etc/nginx/certs/self.crt not found in the container."
"${DOCKER_BIN}" exec "${CONTAINER_NAME}" test -s /etc/nginx/certs/self.key ||
  fail "Key /etc/nginx/certs/self.key not found in the container."

echo "[6/7] Checking HTTP -> HTTPS redirect..."
HTTP_HEADERS="$(curl -sSI -m 5 -H 'Host: localhost' "http://127.0.0.1:${HTTP_PORT}/" || true)"
printf '%s\n' "${HTTP_HEADERS}"
grep -Eq '^HTTP/[0-9.]+ 30[12378]' <<<"${HTTP_HEADERS}" ||
  fail "HTTP endpoint did not return a redirect status."
grep -qi '^Location: https://' <<<"${HTTP_HEADERS}" ||
  fail "HTTP endpoint did not redirect to an https:// URL."

echo "[7/7] Checking HTTPS response..."
if ! wait_http "https://127.0.0.1:${HTTPS_PORT}/" 10 -k; then
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
  fail "HTTPS endpoint on port ${HTTPS_PORT} did not answer (is 'listen 443 ssl;' configured?)."
fi
HTTPS_BODY="$(curl -sk -m 5 "https://127.0.0.1:${HTTPS_PORT}/" || true)"
printf '%s\n' "${HTTPS_BODY}"
grep -q 'tls-ok' <<<"${HTTPS_BODY}" ||
  fail "HTTPS endpoint did not return the expected body: tls-ok"

pass "15 – HTTPS (TLS) Quick Lab"
