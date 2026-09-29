#!/usr/bin/env bash
# Checks exercise 08: NGINX balances raw TCP connections across two backends (stream).

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="08_stream_load_balancer"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_tcp_lb_08_run}"
HOST_PORT="${HOST_PORT:-19000}"
PYTHON_BIN="${PYTHON_BIN:-python3}"
CONNECTIONS=10

# tcp_read PORT connects once and prints the first line the server sends.
# Errors go to stderr so failures stay visible.
tcp_read() {
  "${PYTHON_BIN}" -c '
import socket, sys
s = socket.create_connection(("127.0.0.1", int(sys.argv[1])), timeout=2.0)
try:
    data = s.recv(1024)
finally:
    s.close()
print(data.decode("utf-8", "ignore").strip())
' "$1"
}

echo "[1/6] Checking prerequisites and files..."
require_docker
require_cmd "${PYTHON_BIN}" "the check talks raw TCP with Python (override with PYTHON_BIN)"
for file in Dockerfile nginx.conf start.sh; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HOST_PORT}" "the exercise container (override with HOST_PORT)"

echo "[2/6] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/6] Validating nginx.conf inside the image (nginx -t, nginx -T)..."
if ! T_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t 2>&1)"; then
  printf '%s\n' "${T_OUTPUT}"
  fail "nginx -t rejected nginx.conf; fix the errors above."
fi
if ! CONF_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T 2>&1)"; then
  printf '%s\n' "${CONF_OUTPUT}"
  fail "nginx -T failed inside the image."
fi
CONF="$(printf '%s\n' "${CONF_OUTPUT}" | grep -v '^[[:space:]]*#' || true)"
if ! grep -qE '^[[:space:]]*stream[[:space:]]*\{' <<<"${CONF}"; then
  fail "No active top-level stream {} block found. Layer 4 proxying needs stream, not http."
fi
if ! grep -qE '^[[:space:]]*proxy_pass[[:space:]]+[A-Za-z0-9_.:-]+[[:space:]]*;' <<<"${CONF}"; then
  fail "No active proxy_pass found. Forward the TCP connections to your upstream with proxy_pass."
fi

echo "[4/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 9000)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:9000" \
  "${IMAGE_NAME}" >/dev/null

# Readiness: the published port accepts connections before NGINX is up, so
# wait until a connection actually returns backend data.
ready=0
for ((i = 0; i < 100; i++)); do
  if [[ "$("${DOCKER_BIN}" inspect -f '{{.State.Running}}' "${CONTAINER_NAME}" 2>/dev/null || true)" != "true" ]]; then
    echo "Container ${CONTAINER_NAME} is not running. Logs:"
    "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
    fail "The container exited during startup; see the logs above."
  fi
  if [[ "$(tcp_read "${HOST_PORT}" 2>/dev/null || true)" == backend=* ]]; then
    ready=1
    break
  fi
  sleep 0.2
done
if ((ready == 0)); then
  echo "Container logs:"
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
  fail "Port ${HOST_PORT} returned no backend data within 20s. Is NGINX proxying to the upstream?"
fi

echo "[5/6] Sending ${CONNECTIONS} TCP connections and collecting responses..."
temp_file RESP_FILE
for ((i = 1; i <= CONNECTIONS; i++)); do
  tcp_read "${HOST_PORT}" >>"${RESP_FILE}" || fail "TCP connection ${i} to 127.0.0.1:${HOST_PORT} failed; see the error above."
done

echo "[6/6] Checking that both backends were reached..."
c1="$(grep -c '^backend=9001$' "${RESP_FILE}" || true)"
c2="$(grep -c '^backend=9002$' "${RESP_FILE}" || true)"
if ((c1 == 0 || c2 == 0)); then
  echo "Responses collected:"
  cat "${RESP_FILE}"
  fail "Expected responses from both backends. Got backend=9001 count=${c1}, backend=9002 count=${c2}."
fi

echo "Distribution looks good: backend=9001 count=${c1}, backend=9002 count=${c2}"
pass "08 – TCP/UDP Load Balancer (Layer 4, stream)"
