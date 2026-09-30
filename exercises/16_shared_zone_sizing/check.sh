#!/usr/bin/env bash
# Checks exercise 16: req_zone is large enough (>= 256k) to keep remembering a
# rate-limited client while many other distinct keys arrive.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="16_shared_zone_sizing"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex16_zone_sizing_run}"
HOST_PORT="${HOST_PORT:-18160}"
URL="http://127.0.0.1:${HOST_PORT}/"
# Distinct keys sent between two requests of the limited client. A 64k or 128k
# zone cannot hold them all and evicts the limited client; 256k can.
FLOOD_KEYS=1500

size_to_bytes() {
  local s="${1,,}"
  if [[ "${s}" =~ ^([0-9]+)([km]?)$ ]]; then
    local n="${BASH_REMATCH[1]}" u="${BASH_REMATCH[2]}"
    case "${u}" in
      k) echo $((n * 1024)) ;;
      m) echo $((n * 1024 * 1024)) ;;
      *) echo $((n)) ;;
    esac
    return 0
  fi
  echo 0
}

echo "[1/6] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in Dockerfile nginx.conf; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HOST_PORT}" "the exercise container (override with HOST_PORT)"

echo "[2/6] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/6] Testing the config and reading req_zone (nginx -T)..."
temp_file DUMP
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T >"${DUMP}" 2>&1; then
  fail_nginx_rejected "${DUMP}"
fi
# Effective configuration: no comments, one directive per line.
temp_file ACTIVE
active_config "${DUMP}" >"${ACTIVE}"

zone_line="$(grep -E '^[[:space:]]*limit_req_zone[[:space:]]' "${ACTIVE}" | grep -E 'zone=req_zone:' | head -n 1 || true)"
[[ -n "${zone_line}" ]] || fail "No limit_req_zone directive with zone=req_zone:<size> found in nginx.conf."
zone_size="$(grep -Eo 'zone=req_zone:[0-9]+[kKmM]?([[:space:];]|$)' <<<"${zone_line}" | head -n 1 | sed -E 's/^zone=req_zone:([0-9]+[kKmM]?).*/\1/' || true)"
[[ -n "${zone_size}" ]] || fail "Could not read the size in: ${zone_line}"
grep -qE '[[:space:]]rate=1r/m([[:space:];]|$)' <<<"${zone_line}" ||
  fail "Keep rate=1r/m on req_zone; the check needs a limited client to stay limited for a minute."
grep -qE '^[[:space:]]*limit_req[[:space:]]+([^;]*[[:space:]])?zone=req_zone([[:space:];])' "${ACTIVE}" ||
  fail "Missing 'limit_req zone=req_zone ...' in location /."
echo "req_zone size: ${zone_size}"

echo "[4/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
if ! try_container_http "${CONTAINER_NAME}" "${URL}?id=ready" 10; then
  fail "Nothing answered on port 8080. Keep 'listen 8080;' in the server block."
fi

echo "[5/6] Checking that req_zone limits a client (?id=limited)..."
first="$(curl -s -m 5 -o /dev/null -w '%{http_code}' "${URL}?id=limited" || true)"
second="$(curl -s -m 5 -o /dev/null -w '%{http_code}' "${URL}?id=limited" || true)"
echo "First request: ${first}, second request: ${second}"
[[ "${first}" == "200" ]] || fail "Expected HTTP 200 for the first request, got ${first}."
if [[ "${second}" != "429" ]]; then
  fail "Expected HTTP 429 for the second request within a minute, got ${second}. Keep limit_req in location / and serve the file instead of using return."
fi

echo "[6/6] Sending ${FLOOD_KEYS} other distinct keys, then asking again as the limited client..."
temp_file CODES
# One curl process reuses a single keep-alive connection for all keys.
curl -s -m 60 -o /dev/null -w '%{http_code}\n' "${URL}?id=key-[1-${FLOOD_KEYS}]" >"${CODES}" || true
flood_ok="$(grep -c '^200$' "${CODES}" || true)"
if ((flood_ok != FLOOD_KEYS)); then
  echo "HTTP codes for the distinct keys:"
  sort "${CODES}" | uniq -c
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | grep -E '\[(alert|crit|error)\]' | tail -n 20 || true
  fail "Expected HTTP 200 for each of the ${FLOOD_KEYS} distinct keys, got ${flood_ok}."
fi

third="$(curl -s -m 5 -o /dev/null -w '%{http_code}' "${URL}?id=limited" || true)"
echo "Limited client after ${FLOOD_KEYS} other keys: ${third}"
if [[ "${third}" == "200" ]]; then
  fail "req_zone:${zone_size} forgot the limited client: it filled up with ${FLOOD_KEYS} other keys, NGINX evicted the least recently used state, and the client got 200 instead of 429. Increase the zone size."
fi
[[ "${third}" == "429" ]] || fail "Expected HTTP 429 for the limited client, got ${third}."

zone_bytes="$(size_to_bytes "${zone_size}")"
if ((zone_bytes < 256 * 1024)); then
  fail "req_zone:${zone_size} held ${FLOOD_KEYS} keys, but the exercise asks for at least 256k to leave headroom."
fi

pass "16 – req_zone:${zone_size} still limits the client after ${FLOOD_KEYS} other keys."
