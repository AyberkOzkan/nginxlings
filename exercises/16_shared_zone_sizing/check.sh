#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
EXERCISE_ID="16_shared_zone_sizing"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"

IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex16_zone_sizing_run}"
HOST_PORT="${HOST_PORT:-18160}"

PREFIX="$(pwd)"

TMP_T=""
CONF_T=""
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
  [ -n "${CONF_T}" ] && rm -f "${CONF_T}" >/dev/null 2>&1 || true
  [ -n "${CODES_FILE}" ] && rm -f "${CODES_FILE}" >/dev/null 2>&1 || true

  "${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  if [ "${BUILT_IMAGE}" = "1" ]; then
    "${DOCKER_BIN}" image rm -f "${IMAGE_NAME}" >/dev/null 2>&1 || true
  fi

}
trap cleanup EXIT

command -v "${DOCKER_BIN}" >/dev/null 2>&1 || fail "docker not found in PATH."
command -v curl >/dev/null 2>&1 || fail "curl not found in PATH."

[ -f "${PREFIX}/Dockerfile" ] || fail "Dockerfile not found."
[ -f "${PREFIX}/nginx.conf" ] || fail "nginx.conf not found."

size_to_bytes() {
  local s="$1"
  s="${s,,}"
  if [[ "$s" =~ ^([0-9]+)([km]?)$ ]]; then
    local n="${BASH_REMATCH[1]}"
    local u="${BASH_REMATCH[2]}"
    case "$u" in
      k) echo $((n * 1024)) ;;
      m) echo $((n * 1024 * 1024)) ;;
      *) echo $((n)) ;;
    esac
    return 0
  fi
  echo 0
}

wait_http_ready() {
  local url="http://127.0.0.1:${HOST_PORT}/"
  for _ in $(seq 1 40); do
    if curl -s -m 1 -o /dev/null "${url}"; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

info "[1/7] Building Docker image..."
DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build --label "${DOCKER_LABEL}" -t "${IMAGE_NAME}" "${PREFIX}" >/dev/null 2>&1 \
  || fail "docker build failed."
BUILT_IMAGE="1"

info "[2/7] Testing nginx config inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}" || true
  fail "nginx -t failed."
fi
rm -f "${TMP_T}" >/dev/null 2>&1 || true
TMP_T=""
info "[OK] nginx -t succeeded."

info "[3/7] Checking zone size in effective config (nginx -T)..."
CONF_T="$(mktemp)"
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T >"${CONF_T}" 2>&1 || true

zone_token="$(grep -Eo 'zone=req_zone:[0-9]+[kKmM]?' "${CONF_T}" | head -n 1 || true)"
[ -n "${zone_token}" ] || {
  sed -n '1,160p' "${CONF_T}" || true
  fail "Could not find zone=req_zone:<size> in effective config."
}

zone_size="${zone_token#zone=req_zone:}"
zone_bytes="$(size_to_bytes "${zone_size}")"
min_bytes="$((256 * 1024))"

if [ "${zone_bytes}" -lt "${min_bytes}" ]; then
  fail "Zone size must be >= 256k. Current: ${zone_size}"
fi

grep -qE 'limit_req[[:space:]]+zone=req_zone' "${CONF_T}" \
  || fail "Missing limit_req usage with zone=req_zone."

rm -f "${CONF_T}" >/dev/null 2>&1 || true
CONF_T=""
info "[OK] Zone sizing looks good (req_zone:${zone_size})."

info "[4/7] Starting container..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container."

wait_http_ready || {
  echo "docker logs (first 120 lines):"
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | sed -n '1,120p' || true
  fail "Container did not become ready on time."
}

info "[5/7] Creating many unique keys to stress the shared memory zone..."
for i in $(seq 1 1200); do
  (
    curl -s -m 2 "http://127.0.0.1:${HOST_PORT}/?id=${i}" >/dev/null 2>&1 || true
  ) &
  if (( i % 50 == 0 )); then
    wait || true
  fi
done
wait || true
sleep 0.5

LOGS="$("${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>/dev/null || true)"
echo "${LOGS}" | grep -qiE 'could not allocate|no memory|ngx_slab_alloc' && {
  echo "Sample logs (first 120 lines):"
  printf '%s\n' "${LOGS}" | sed -n '1,120p' || true
  fail "Shared memory allocation errors detected. Increase the zone size."
}

info "[OK] No shared memory allocation errors detected."

info "[6/7] Verifying rate limiting is active (expects some 200 and some 429)..."
CODES_FILE="$(mktemp)"
: > "${CODES_FILE}"

for _ in $(seq 1 30); do
  (
    code="$(curl -s -m 2 -o /dev/null -w '%{http_code}' "http://127.0.0.1:${HOST_PORT}/?id=same" || echo "000")"
    echo "${code}" >> "${CODES_FILE}"
  ) &
done
wait || true

ok_count="$(grep -c '^200$' "${CODES_FILE}" || true)"
limited_count="$(grep -c '^429$' "${CODES_FILE}" || true)"

if [ "${ok_count}" -eq 0 ]; then
  echo "HTTP codes:"
  cat "${CODES_FILE}" || true
  fail "Expected at least one 200 response."
fi

if [ "${limited_count}" -eq 0 ]; then
  echo "HTTP codes:"
  cat "${CODES_FILE}" || true
  fail "Expected at least one 429 response (rate limiting)."
fi

rm -f "${CODES_FILE}" >/dev/null 2>&1 || true
CODES_FILE=""
info "[OK] Rate limiting observed (200 and 429)."

info "[7/7] Done."
echo
echo "[PASS] 16 – Shared Memory Zone Sizing & Limits looks good."