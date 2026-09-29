#!/usr/bin/env bash
# Checks exercise 14: two server blocks serve two static sites by Host header.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="14_multi_domain_static"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex14_multi_domain_run}"
HOST_PORT="${HOST_PORT:-18140}"
URL="http://127.0.0.1:${HOST_PORT}/"

echo "[1/5] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in nginx.conf Dockerfile sites/site1/index.html sites/site2/index.html; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HOST_PORT}" "the exercise container (override with HOST_PORT)"

echo "[2/5] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/5] Testing nginx config and its server blocks inside the image (nginx -T)..."
temp_file DUMP
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T >"${DUMP}" 2>&1; then
  cat "${DUMP}"
  fail "NGINX rejected nginx.conf; fix the errors above."
fi
# Effective configuration without comment lines.
temp_file ACTIVE
grep -v '^[[:space:]]*#' "${DUMP}" >"${ACTIVE}" || true
for n in 1 2; do
  grep -qE "^[[:space:]]*server_name[[:space:]]+([^;]*[[:space:]])?site${n}\.local[[:space:]]*;" "${ACTIVE}" ||
    fail "No active server block with 'server_name site${n}.local;' yet."
done

echo "[4/5] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
if ! try_container_http "${CONTAINER_NAME}" "${URL}" 10; then
  fail "Nothing answered on port 8080. Add the site1.local and site2.local server blocks with 'listen 8080;'."
fi

echo "[5/5] Checking each Host header serves its own site..."
for n in 1 2; do
  body="$(curl -s -m 5 -H "Host: site${n}.local" "${URL}" || true)"
  echo "GET / with Host=site${n}.local returned:"
  printf '%s\n' "${body}"
  grep -q "site${n}-static-ok" <<<"${body}" ||
    fail "site${n}.local did not return the expected static content (site${n}-static-ok)."
done

pass "14 – Multiple Domains, Static Sites"
