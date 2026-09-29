#!/usr/bin/env bash
# Checks exercise 13: NGINX loads modular config through include directives.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="13_include_modular_config"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex13_include_modular_run}"
HOST_PORT="${HOST_PORT:-18130}"
BASE_URL="http://127.0.0.1:${HOST_PORT}"

# expect_body PATH EXPECTED HINT fetches BASE_URL/PATH and requires HTTP 200
# with EXPECTED as the body (trailing newlines are ignored); HINT says where
# the learner should look when it does not match.
expect_body() {
  local path=$1 expected=$2 hint=$3 code body
  if ! code="$(curl -sS -m 5 -o "${BODY_FILE}" -w '%{http_code}' "${BASE_URL}${path}")"; then
    fail "GET ${path} failed; see the curl error above."
  fi
  body="$(<"${BODY_FILE}")"
  if [[ "${code}" != "200" || "${body}" != "${expected}" ]]; then
    echo "GET ${path} returned HTTP ${code} with body:"
    printf '%s\n' "${body}"
    fail "Expected GET ${path} to return HTTP 200 with exactly: ${expected}. ${hint}"
  fi
  echo "GET ${path} -> ${expected}"
}

echo "[1/5] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in Dockerfile nginx.conf conf.d/10-server.conf server.d/20-modular.conf server.d/30-modular-extra.conf; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HOST_PORT}" "the exercise container (override with HOST_PORT)"

echo "[2/5] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/5] Testing the config and its include directives inside the image (nginx -T)..."
temp_file DUMP
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T >"${DUMP}" 2>&1; then
  fail_nginx_rejected "${DUMP}"
fi
# Effective configuration without comment lines.
temp_file ACTIVE
grep -v '^[[:space:]]*#' "${DUMP}" >"${ACTIVE}" || true
if ! grep -qE '^[[:space:]]*include[[:space:]]+/etc/nginx/conf\.d/\*\.conf;' "${ACTIVE}"; then
  fail "Missing: include /etc/nginx/conf.d/*.conf; (must be in http{} of nginx.conf)."
fi
if ! grep -qE '^[[:space:]]*include[[:space:]]+/etc/nginx/server\.d/\*\.conf;' "${ACTIVE}"; then
  fail "Missing: include /etc/nginx/server.d/*.conf; (must be in server{} of conf.d/10-server.conf)."
fi

echo "[4/5] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
if ! try_container_http "${CONTAINER_NAME}" "${BASE_URL}/" 10; then
  fail "Nothing answered on port 8080. Does the server block in conf.d/10-server.conf still have 'listen 8080;'?"
fi

echo "[5/5] Checking endpoints..."
temp_file BODY_FILE
expect_body "/" "base-ok" \
  "location / in conf.d/10-server.conf answers it."
expect_body "/modular" "include-ok" \
  "Is location /modular defined (not commented out) in server.d/20-modular.conf? Otherwise location / answers it."
expect_body "/modular2" "include2-ok" \
  "Is location /modular2 defined (not commented out) in server.d/30-modular-extra.conf? Otherwise location / answers it."

pass "13 – Include & Modular Config Design"
