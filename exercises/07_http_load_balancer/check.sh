#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME_DEFAULT="nginxlings_http_lb_07"
IMAGE_NAME="${IMAGE_NAME:-$IMAGE_NAME_DEFAULT}"
CONTAINER_NAME_DEFAULT="nginxlings_http_lb_07_run"
CONTAINER_NAME="${CONTAINER_NAME:-$CONTAINER_NAME_DEFAULT}"

# Host port for the frontend (mapped to container's 8080)
FRONTEND_PORT="${FRONTEND_PORT:-18080}"

fail() {
  echo "[FAIL] $*"
  exit 1
}

info() {
  echo "$@"
}

PREFIX="$(pwd)"

# Cleanup: stop container if still running and remove temp file
container_started=0
TMP_LOG=""

cleanup() {
  if [ "${container_started}" -eq 1 ]; then
    docker stop "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  fi
  if [ -n "${TMP_LOG}" ] && [ -f "${TMP_LOG}" ]; then
    rm -f "${TMP_LOG}" || true
  fi
}
trap cleanup EXIT

info "[1/6] Checking required files and commands..."

command -v docker >/dev/null 2>&1 || fail "docker command not found."
command -v curl   >/dev/null 2>&1 || fail "curl command not found."

[ -f "${PREFIX}/Dockerfile" ] || fail "Dockerfile not found in ${PREFIX}."
[ -f "${PREFIX}/nginx.conf" ] || fail "nginx.conf not found in ${PREFIX}."
[ -f "${PREFIX}/start.sh"   ] || fail "start.sh not found in ${PREFIX}."

info "[OK] Files and commands look fine."

info "[2/6] Building Docker image (${IMAGE_NAME})..."

if ! docker build -t "${IMAGE_NAME}" "${PREFIX}" >/dev/null; then
  fail "docker build failed for image ${IMAGE_NAME}."
fi

info "[OK] Image built successfully."

info "[3/6] Testing nginx configuration inside the image (nginx -t)..."

TMP_T="$(mktemp)"
if ! docker run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}"
  rm -f "${TMP_T}"
  fail "nginx -t failed inside the container."
fi
rm -f "${TMP_T}"

info "[OK] nginx -t succeeded."

info "[4/6] Starting container ${CONTAINER_NAME} (frontend on host port ${FRONTEND_PORT} -> container 8080)..."

# Make sure there is no stale container with the same name
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

if ! docker run --rm -d \
  --name "${CONTAINER_NAME}" \
  -p "${FRONTEND_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null; then
  fail "Failed to start container ${CONTAINER_NAME}."
fi
container_started=1

# Give container a moment to start backends + nginx
sleep 3

# Check container is still running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}\$"; then
  echo "Container ${CONTAINER_NAME} is not running. Logs:"
  docker logs "${CONTAINER_NAME}" || true
  fail "Container exited unexpectedly."
fi

info "[OK] Container is running."

info "[5/6] Sending HTTP requests through http://127.0.0.1:${FRONTEND_PORT}/ ..."

for i in 1 2 3 4 5 6 7 8; do
  if ! curl -s -o /dev/null "http://127.0.0.1:${FRONTEND_PORT}/"; then
    fail "curl request ${i} to http://127.0.0.1:${FRONTEND_PORT}/ failed."
  fi
done

# Give nginx a moment to flush logs
sleep 1

info "[6/6] Inspecting container logs for upstream distribution..."

TMP_LOG="$(mktemp)"
docker logs "${CONTAINER_NAME}" > "${TMP_LOG}" 2>&1 || true

BACK1="127.0.0.1:8081"
BACK2="127.0.0.1:8082"

count1="$(grep -c "upstream=${BACK1}" "${TMP_LOG}" || true)"
count2="$(grep -c "upstream=${BACK2}" "${TMP_LOG}" || true)"

if [ "${count1}" -eq 0 ] || [ "${count2}" -eq 0 ]; then
  echo
  echo "Sample logs (first 40 lines):"
  sed -n '1,40p' "${TMP_LOG}" || true
  echo
  fail "Expected both ${BACK1} and ${BACK2} in logs via upstream=\$upstream_addr, got counts: ${BACK1}=${count1}, ${BACK2}=${count2}."
fi

echo
echo "[PASS] HTTP load balancing looks correct."
echo "       Requests are distributed between ${BACK1} and ${BACK2} (see upstream=\$upstream_addr in Docker logs)."
echo "[PASS] 07 – HTTP Load Balancer (Layer 7)"
