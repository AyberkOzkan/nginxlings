#!/usr/bin/env bash
# Checks exercise 11: the zone_sync API endpoint returns valid JSON.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="11_zone_sync_api"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex11_zone_sync_run}"
HOST_PORT="${HOST_PORT:-18110}"
URL="http://127.0.0.1:${HOST_PORT}/"

# in_container runs a shell snippet inside the exercise container.
in_container() {
  "${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -c "$1"
}

echo "[1/5] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in Dockerfile nginx.conf api/zone_sync.json; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HOST_PORT}" "the exercise container (override with HOST_PORT)"

echo "[2/5] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/5] Testing nginx config inside the image (nginx -t)..."
if ! T_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t 2>&1)"; then
  printf '%s\n' "${T_OUTPUT}"
  fail "nginx -t rejected nginx.conf; fix the errors above."
fi

echo "[4/5] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
wait_container_http "${CONTAINER_NAME}" "${URL}"

echo "[5/5] Querying API endpoint and validating JSON..."

# Fetch headers + body inside the container, where curl and jq are installed.
if ! HDRS="$(in_container "curl -s -D - -o /tmp/zone_sync.json http://127.0.0.1:8080/api/8/stream/zone_sync | tr -d '\r'" 2>&1)"; then
  printf '%s\n' "${HDRS}"
  fail "Could not query /api/8/stream/zone_sync inside the container; see the output above."
fi

# Status code must be 200
if ! printf '%s\n' "${HDRS}" | head -n 1 | grep -q " 200"; then
  echo "Response headers:"
  printf '%s\n' "${HDRS}"
  echo "Body snippet:"
  in_container "sed -n '1,40p' /tmp/zone_sync.json" || true
  fail "Expected HTTP 200."
fi

# Content-Type must include application/json
if ! printf '%s\n' "${HDRS}" | grep -i -q "content-type:.*application/json"; then
  echo "Response headers:"
  printf '%s\n' "${HDRS}"
  fail "Expected Content-Type to include application/json."
fi

# JSON must be valid and contain required fields
in_container "jq -e '.zones.sync_zone.state' /tmp/zone_sync.json >/dev/null" \
  || fail "Missing JSON field: .zones.sync_zone.state"
in_container "jq -e '.zones.sync_zone.last_sync' /tmp/zone_sync.json >/dev/null" \
  || fail "Missing JSON field: .zones.sync_zone.last_sync"
in_container "jq -e '.zones.sync_zone.peers | length == 2' /tmp/zone_sync.json >/dev/null" \
  || fail "Expected exactly 2 peers at .zones.sync_zone.peers"

pass "11 – Zone Synchronization & API (zone_sync)"
