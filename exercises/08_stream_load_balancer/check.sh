#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"

IMAGE_NAME_DEFAULT="nginxlings_tcp_lb_08"
IMAGE_NAME="${IMAGE_NAME:-$IMAGE_NAME_DEFAULT}"

CONTAINER_NAME_DEFAULT="nginxlings_tcp_lb_08_run"
CONTAINER_NAME="${CONTAINER_NAME:-$CONTAINER_NAME_DEFAULT}"

HOST_PORT="${HOST_PORT:-19000}"
PYTHON_BIN="${PYTHON_BIN:-python3}"

TMP_T=""
RESP_FILE=""
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
  [ -n "${RESP_FILE}" ] && rm -f "${RESP_FILE}" >/dev/null 2>&1 || true

  "${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

  if [ "${BUILT_IMAGE}" = "1" ]; then
    "${DOCKER_BIN}" image rm -f "${IMAGE_NAME}" >/dev/null 2>&1 || true
  fi

  "${DOCKER_BIN}" builder prune -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

command -v "${DOCKER_BIN}" >/dev/null 2>&1 || fail "docker command not found."
command -v "${PYTHON_BIN}" >/dev/null 2>&1 || fail "python3 not found (PYTHON_BIN=${PYTHON_BIN})."

[ -f Dockerfile ] || fail "Dockerfile not found in this directory."
[ -f nginx.conf ] || fail "nginx.conf not found in this directory."
[ -f start.sh ] || fail "start.sh not found in this directory."

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

info "[3/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 9000)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:9000" "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container ${CONTAINER_NAME}."

sleep 2

info "[4/6] Sending multiple TCP connections and collecting responses..."
RESP_FILE="$(mktemp)"
rm -f "${RESP_FILE}" >/dev/null 2>&1 || true
touch "${RESP_FILE}"

for _ in 1 2 3 4 5 6 7 8 9 10; do
  "${PYTHON_BIN}" - <<PY >> "${RESP_FILE}" 2>/dev/null
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.settimeout(2.0)
s.connect(("127.0.0.1", int("${HOST_PORT}")))
data = s.recv(1024)
s.close()
print(data.decode("utf-8", "ignore").strip())
PY
done

info "[5/6] Checking that both backends were reached..."

c1="$(grep -c '^backend=9001$' "${RESP_FILE}" || true)"
c2="$(grep -c '^backend=9002$' "${RESP_FILE}" || true)"

if [ "${c1}" -eq 0 ] || [ "${c2}" -eq 0 ]; then
  echo "Responses collected:"
  cat "${RESP_FILE}" || true
  fail "Expected responses from both backends. Got backend=9001 count=${c1}, backend=9002 count=${c2}."
fi

info "[OK] Distribution looks good: backend=9001 count=${c1}, backend=9002 count=${c2}"

rm -f "${RESP_FILE}" >/dev/null 2>&1 || true
RESP_FILE=""

info "[6/6] Done."
echo
echo "[PASS] 08 – TCP/UDP Load Balancer (Layer 4, stream) looks good."