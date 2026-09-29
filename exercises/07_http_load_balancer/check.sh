#!/usr/bin/env bash
# Checks exercise 07: NGINX balances HTTP requests across two backends.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="07_http_load_balancer"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_http_lb_07_run}"

# Host port for the frontend (mapped to container's 8080)
FRONTEND_PORT="${FRONTEND_PORT:-18080}"
URL="http://127.0.0.1:${FRONTEND_PORT}/"
REQUESTS=8

BACK1="127.0.0.1:8081"
BACK2="127.0.0.1:8082"

echo "[1/6] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in Dockerfile nginx.conf start.sh; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${FRONTEND_PORT}" "the exercise container (override with FRONTEND_PORT)"

echo "[2/6] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/6] Testing nginx.conf inside the image (nginx -t)..."
if ! T_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t 2>&1)"; then
  fail_nginx_rejected - <<<"${T_OUTPUT}"
fi

echo "[4/6] Starting container ${CONTAINER_NAME} (host ${FRONTEND_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${FRONTEND_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
wait_container_http "${CONTAINER_NAME}" "${URL}"

echo "[5/6] Sending ${REQUESTS} HTTP requests through ${URL} ..."
for ((i = 1; i <= REQUESTS; i++)); do
  curl -s -o /dev/null "${URL}" || fail "Request ${i} to ${URL} failed."
done

echo "[6/6] Inspecting container logs for upstream distribution..."
temp_file LOGS
# Access log lines reach docker logs asynchronously; wait until the
# requests are there instead of sleeping for a fixed time.
for ((i = 0; i < 50; i++)); do
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" >"${LOGS}" 2>&1 || true
  (($(grep -c 'upstream=' "${LOGS}" || true) >= REQUESTS)) && break
  sleep 0.1
done

count1="$(grep -c "upstream=${BACK1}" "${LOGS}" || true)"
count2="$(grep -c "upstream=${BACK2}" "${LOGS}" || true)"
if ((count1 == 0 || count2 == 0)); then
  echo "Sample logs (first 60 lines):"
  sed -n '1,60p' "${LOGS}"
  fail "Expected both ${BACK1} and ${BACK2} in upstream=\$upstream_addr, got ${BACK1}=${count1}, ${BACK2}=${count2}."
fi

echo "Requests were distributed: ${BACK1}=${count1}, ${BACK2}=${count2}."
pass "07 – HTTP Load Balancer (Layer 7)"
