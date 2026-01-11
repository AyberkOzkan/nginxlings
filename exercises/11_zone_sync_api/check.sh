#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="${IMAGE_NAME:-nginxlings_ex11_zone_sync}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex11_zone_sync_run}"
HOST_PORT="${HOST_PORT:-18110}"

fail() {
    echo "[FAIL] $*"
    exit 1
}

info() {
    echo "$@"
}

info "[1/5] Checking required commands..."
command -v docker >/dev/null 2>&1 || fail "docker not found in PATH."

info "[2/5] Building Docker image..."
docker build -t "${IMAGE_NAME}" . >/dev/null

info "[3/5] Testing nginx config inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! docker run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
    echo "nginx -t output:"
    cat "${TMP_T}" || true
    rm -f "${TMP_T}"
    fail "nginx -t failed."
fi
rm -f "${TMP_T}"
info "[OK] nginx -t succeeded."

info "[4/5] Starting container..."
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null

cleanup() {
    docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# give nginx a moment
sleep 1

info "[5/5] Querying API endpoint and validating JSON..."

# Fetch headers + body (inside container to ensure curl/jq are available)
HDRS="$(docker exec "${CONTAINER_NAME}" sh -c "curl -s -D - -o /tmp/zone_sync.json http://127.0.0.1:8080/api/8/stream/zone_sync | tr -d '\r'")"

# Status code must be 200
echo "${HDRS}" | head -n 1 | grep -q " 200" || {
    echo "Response headers:"
    echo "${HDRS}"
    echo "Body snippet:"
    docker exec "${CONTAINER_NAME}" sh -c "sed -n '1,40p' /tmp/zone_sync.json" || true
    fail "Expected HTTP 200."
}

# Content-Type must include application/json
echo "${HDRS}" | grep -i -q "content-type:.*application/json" || {
    echo "Response headers:"
    echo "${HDRS}"
    fail "Expected Content-Type to include application/json."
}

# JSON must be valid and contain required fields
docker exec "${CONTAINER_NAME}" sh -c "jq -e '.zones.sync_zone.state' /tmp/zone_sync.json >/dev/null" \
    || fail "Missing JSON field: .zones.sync_zone.state"

docker exec "${CONTAINER_NAME}" sh -c "jq -e '.zones.sync_zone.last_sync' /tmp/zone_sync.json >/dev/null" \
    || fail "Missing JSON field: .zones.sync_zone.last_sync"

docker exec "${CONTAINER_NAME}" sh -c "jq -e '.zones.sync_zone.peers | length == 2' /tmp/zone_sync.json >/dev/null" \
    || fail "Expected exactly 2 peers at .zones.sync_zone.peers"

echo
echo "[PASS] 11 – Zone Synchronization & API (zone_sync) looks good."
