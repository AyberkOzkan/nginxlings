#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME_DEFAULT="nginxlings_shm_zone_09"
IMAGE_NAME="${IMAGE_NAME:-$IMAGE_NAME_DEFAULT}"

CONTAINER_NAME_DEFAULT="nginxlings_shm_zone_09_run"
CONTAINER_NAME="${CONTAINER_NAME:-$CONTAINER_NAME_DEFAULT}"

HOST_PORT="${HOST_PORT:-19090}"

fail() {
  echo "[FAIL] $*"
  exit 1
}

info() {
  echo "$@"
}

command -v docker >/dev/null 2>&1 || fail "docker command not found."
command -v curl   >/dev/null 2>&1 || fail "curl command not found."

[ -f Dockerfile ] || fail "Dockerfile not found in this directory."
[ -f nginx.conf ] || fail "nginx.conf not found in this directory."
[ -f start.sh ]   || fail "start.sh not found in this directory."

chmod +x start.sh >/dev/null 2>&1 || true

info "[1/6] Building Docker image (${IMAGE_NAME})..."
docker build -t "${IMAGE_NAME}" . >/dev/null

info "[2/6] Validating nginx.conf inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! docker run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}"
  rm -f "${TMP_T}"
  fail "nginx -t failed inside the container."
fi
rm -f "${TMP_T}"
info "[OK] nginx -t succeeded."

info "[3/6] Inspecting effective config (nginx -T) for required directives..."
TMP_CONF="$(mktemp)"
docker run --rm "${IMAGE_NAME}" nginx -T >"${TMP_CONF}" 2>&1 || true

if ! grep -qE "limit_req_zone[[:space:]]+\\\$binary_remote_addr[[:space:]]+zone=req_zone:64k[[:space:]]+rate=" "${TMP_CONF}"; then
  echo "nginx -T output (first 120 lines):"
  sed -n '1,120p' "${TMP_CONF}" || true
  rm -f "${TMP_CONF}"
  fail "Missing or incorrect limit_req_zone. Expected zone=req_zone:64k."
fi

if ! grep -qE "limit_req[[:space:]]+zone=req_zone" "${TMP_CONF}"; then
  echo "nginx -T output (first 120 lines):"
  sed -n '1,120p' "${TMP_CONF}" || true
  rm -f "${TMP_CONF}"
  fail "Missing limit_req usage with zone=req_zone."
fi

rm -f "${TMP_CONF}"
info "[OK] Required shared memory zone directives found."

info "[4/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null

cleanup() {
  docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

sleep 2

info "[5/6] Sending quick requests to trigger rate limiting..."

codes_file="$(mktemp)"
rm -f "${codes_file}"
touch "${codes_file}"

# Fire a burst of requests in parallel to reliably trigger limit_req
PARALLEL_N="${PARALLEL_N:-20}"

pids=()
for i in $(seq 1 "${PARALLEL_N}"); do
  (
    code="$(curl -s -m 2 -o /dev/null -w '%{http_code}' "http://127.0.0.1:${HOST_PORT}/" || echo "000")"
    echo "${code}" >> "${codes_file}"
  ) &
  pids+=($!)
done

# wait all
for pid in "${pids[@]}"; do
  wait "${pid}" || true
done

ok_count="$(grep -c '^200$' "${codes_file}" || true)"
limited_count="$(grep -cE '^(429|503)$' "${codes_file}" || true)"

if [ "${ok_count}" -eq 0 ]; then
  echo "HTTP codes:"
  cat "${codes_file}" || true
  rm -f "${codes_file}"
  fail "Expected at least one 200 response."
fi

if [ "${limited_count}" -eq 0 ]; then
  echo "HTTP codes:"
  cat "${codes_file}" || true
  rm -f "${codes_file}"
  fail "Expected at least one rate-limited response (429 or 503)."
fi

rm -f "${codes_file}"
info "[OK] Rate limiting is active (200 and 429/503 observed)."

info "[6/6] Done."
echo
echo "[PASS] 09 – Shared Memory Zones Fundamentals looks good."
