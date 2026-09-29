#!/usr/bin/env bash
# Checks exercise 09: a limit_req shared memory zone rate-limits requests.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="09_shared_zones_basics"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_shm_zone_09_run}"
HOST_PORT="${HOST_PORT:-19090}"
PARALLEL_N="${PARALLEL_N:-20}"
URL="http://127.0.0.1:${HOST_PORT}/"

echo "[1/6] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in Dockerfile nginx.conf start.sh; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HOST_PORT}" "the exercise container (override with HOST_PORT)"

echo "[2/6] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/6] Validating nginx.conf inside the image (nginx -t)..."
if ! T_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t 2>&1)"; then
  fail_nginx_rejected - <<<"${T_OUTPUT}"
fi

echo "[4/6] Inspecting the effective config (nginx -T) for the zone directives..."
if ! CONF_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T 2>&1)"; then
  printf '%s\n' "${CONF_OUTPUT}"
  fail "nginx -T failed inside the image."
fi
CONF="$(printf '%s\n' "${CONF_OUTPUT}" | grep -v '^[[:space:]]*#' || true)"
ZONE_LINES="$(grep -E '^[[:space:]]*limit_req_zone[[:space:]]' <<<"${CONF}" || true)"
[[ -n "${ZONE_LINES}" ]] || fail "No active limit_req_zone yet: define the shared memory zone in the http block."
REQ_ZONE="$(grep -E 'zone=req_zone:64k([[:space:];]|$)' <<<"${ZONE_LINES}" || true)"
[[ -n "${REQ_ZONE}" ]] ||
  fail "limit_req_zone must create a zone named req_zone with a size of 64k (zone=<name>:<size>)."
# shellcheck disable=SC2016 # the pattern is a regex, not a shell expansion
grep -qE '^[[:space:]]*limit_req_zone[[:space:]]+\$binary_remote_addr[[:space:]]' <<<"${REQ_ZONE}" ||
  fail "The key of req_zone must be the client address in binary form, \$binary_remote_addr."
if ! grep -qE '^[[:space:]]*limit_req[[:space:]]+([^;]*[[:space:]])?zone=req_zone([[:space:];]|$)' <<<"${CONF}"; then
  fail "Nothing uses req_zone yet: apply the zone with limit_req in location /."
fi

echo "[5/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null

# Readiness is probed from inside the container so the probe comes from a
# different client address than the burst below and does not spend the
# host client's rate-limit allowance.
ready=0
for ((i = 0; i < 100; i++)); do
  if [[ "$("${DOCKER_BIN}" inspect -f '{{.State.Running}}' "${CONTAINER_NAME}" 2>/dev/null || true)" != "true" ]]; then
    echo "Container ${CONTAINER_NAME} is not running. Logs:"
    "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
    fail "The container exited during startup; see the logs above."
  fi
  if "${DOCKER_BIN}" exec "${CONTAINER_NAME}" wget -q -O /dev/null http://127.0.0.1:8080/ 2>/dev/null; then
    ready=1
    break
  fi
  sleep 0.2
done
if ((ready == 0)); then
  echo "Container logs:"
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
  fail "NGINX did not answer inside the container within 20s."
fi

echo "[6/6] Sending ${PARALLEL_N} parallel requests to trigger rate limiting..."
temp_file CODES_FILE
pids=()
for ((i = 0; i < PARALLEL_N; i++)); do
  (
    code="$(curl -s -m 2 -o /dev/null -w '%{http_code}' "${URL}" || true)"
    echo "${code:-000}" >>"${CODES_FILE}"
  ) &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "${pid}" || true
done

ok_count="$(grep -c '^200$' "${CODES_FILE}" || true)"
limited_count="$(grep -cE '^(429|503)$' "${CODES_FILE}" || true)"
timeout_count="$(grep -c '^000$' "${CODES_FILE}" || true)"
if ((ok_count == 0 || limited_count == 0)); then
  echo "HTTP codes:"
  sort "${CODES_FILE}" | uniq -c
fi
((ok_count > 0)) || fail "Expected at least one 200 response. Does location / serve a file (for example index.html from the root)?"
if ((limited_count == 0 && timeout_count > 0)); then
  fail "${timeout_count} requests got no answer within 2s instead of being rejected: with a burst and without nodelay, NGINX delays excess requests. Reject them at once instead."
fi
if ((limited_count == 0)); then
  fail "All ${PARALLEL_N} parallel requests got through: lower the rate or the burst. A return in location / would also skip limit_req, because return runs before it."
fi

echo "Rate limiting is active: 200=${ok_count}, 429/503=${limited_count}."
pass "09 – Shared Memory Zones Fundamentals"
