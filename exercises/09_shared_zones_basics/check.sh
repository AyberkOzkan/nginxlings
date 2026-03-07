#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"

IMAGE_NAME_DEFAULT="nginxlings_shm_zone_09"
IMAGE_NAME="${IMAGE_NAME:-$IMAGE_NAME_DEFAULT}"

CONTAINER_NAME_DEFAULT="nginxlings_shm_zone_09_run"
CONTAINER_NAME="${CONTAINER_NAME:-$CONTAINER_NAME_DEFAULT}"

HOST_PORT="${HOST_PORT:-19090}"
PARALLEL_N="${PARALLEL_N:-20}"

TMP_T=""
TMP_CONF=""
CODES_FILE=""
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
  [ -n "${TMP_CONF}" ] && rm -f "${TMP_CONF}" >/dev/null 2>&1 || true
  [ -n "${CODES_FILE}" ] && rm -f "${CODES_FILE}" >/dev/null 2>&1 || true

  "${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

  if [ "${BUILT_IMAGE}" = "1" ]; then
    "${DOCKER_BIN}" image rm -f "${IMAGE_NAME}" >/dev/null 2>&1 || true
  fi

  "${DOCKER_BIN}" builder prune -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

command -v "${DOCKER_BIN}" >/dev/null 2>&1 || fail "docker command not found."
command -v curl >/dev/null 2>&1 || fail "curl command not found."

[ -f Dockerfile ] || fail "Dockerfile not found in this directory."
[ -f nginx.conf ] || fail "nginx.conf not found in this directory."
[ -f start.sh ]   || fail "start.sh not found in this directory."

chmod +x start.sh >/dev/null 2>&1 || true

info "[1/6] Building Docker image (${IMAGE_NAME})..."
DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build -t "${IMAGE_NAME}" . >/dev/null 2>&1 \
  || fail "docker build failed for image ${IMAGE_NAME}."
BUILT_IMAGE="1"

info "[2/6] Validating nginx.conf inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! "${DOCKER_BIN}" run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}" || true
  fail "nginx -t failed inside the container."
fi
rm -f "${TMP_T}" >/dev/null 2>&1 || true
TMP_T=""
info "[OK] nginx -t succeeded."

info "[3/6] Inspecting effective config (nginx -T) for required directives..."
TMP_CONF="$(mktemp)"
"${DOCKER_BIN}" run --rm "${IMAGE_NAME}" nginx -T >"${TMP_CONF}" 2>&1 || true

if ! grep -qE "limit_req_zone[[:space:]]+\\\$binary_remote_addr[[:space:]]+zone=req_zone:64k[[:space:]]+rate=" "${TMP_CONF}"; then
  echo "nginx -T output (first 160 lines):"
  sed -n '1,160p' "${TMP_CONF}" || true
  fail "Missing or incorrect limit_req_zone. Expected zone=req_zone:64k."
fi

if ! grep -qE "limit_req[[:space:]]+zone=req_zone" "${TMP_CONF}"; then
  echo "nginx -T output (first 160 lines):"
  sed -n '1,160p' "${TMP_CONF}" || true
  fail "Missing limit_req usage with zone=req_zone."
fi

rm -f "${TMP_CONF}" >/dev/null 2>&1 || true
TMP_CONF=""
info "[OK] Required shared memory zone directives found."

info "[4/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container ${CONTAINER_NAME}."

sleep 2

info "[5/6] Sending quick requests to trigger rate limiting..."

CODES_FILE="$(mktemp)"
rm -f "${CODES_FILE}" >/dev/null 2>&1 || true
touch "${CODES_FILE}"

pids=()
for _ in $(seq 1 "${PARALLEL_N}"); do
  (
    code="$(curl -s -m 2 -o /dev/null -w '%{http_code}' "http://127.0.0.1:${HOST_PORT}/" || echo "000")"
    echo "${code}" >> "${CODES_FILE}"
  ) &
  pids+=($!)
done

for pid in "${pids[@]}"; do
  wait "${pid}" || true
done

ok_count="$(grep -c '^200$' "${CODES_FILE}" || true)"
limited_count="$(grep -cE '^(429|503)$' "${CODES_FILE}" || true)"

if [ "${ok_count}" -eq 0 ]; then
  echo "HTTP codes:"
  cat "${CODES_FILE}" || true
  fail "Expected at least one 200 response."
fi

if [ "${limited_count}" -eq 0 ]; then
  echo "HTTP codes:"
  cat "${CODES_FILE}" || true
  fail "Expected at least one rate-limited response (429 or 503)."
fi

rm -f "${CODES_FILE}" >/dev/null 2>&1 || true
CODES_FILE=""
info "[OK] Rate limiting is active (200 and 429/503 observed)."

info "[6/6] Done."
echo
echo "[PASS] 09 – Shared Memory Zones Fundamentals looks good."