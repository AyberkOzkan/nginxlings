#!/usr/bin/env bash
# Checks exercise 17: the user directive is set and workers run as nginx.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="17_verify_nginx_user"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex17_verify_user_run}"
HOST_PORT="${HOST_PORT:-18170}"
URL="http://127.0.0.1:${HOST_PORT}/"

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

echo "[3/6] Testing nginx config inside the image (nginx -t)..."
if ! T_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t 2>&1)"; then
  fail_nginx_rejected - <<<"${T_OUTPUT}"
fi

echo "[4/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
wait_container_http "${CONTAINER_NAME}" "${URL}"

echo "[5/6] Checking the user directive and the running processes..."
if ! EFFECTIVE="$("${DOCKER_BIN}" exec "${CONTAINER_NAME}" nginx -T 2>&1)"; then
  printf '%s\n' "${EFFECTIVE}" | sed -n '1,160p'
  fail "Could not run nginx -T inside the container."
fi
# Comment-safe; "user nginx;" and "user nginx nginx;" are both valid.
if ! grep -v '^[[:space:]]*#' <<<"${EFFECTIVE}" |
  grep -qE '^[[:space:]]*user[[:space:]]+nginx([[:space:]]+nginx)?[[:space:]]*;'; then
  fail "Missing or incorrect user directive. Expected: user nginx;"
fi

# Workers are forked right after the master starts; poll instead of sleeping.
PS_OUT=""
for _ in $(seq 1 50); do
  PS_OUT="$("${DOCKER_BIN}" exec "${CONTAINER_NAME}" ps aux 2>/dev/null || true)"
  grep -qE 'nginx: worker process' <<<"${PS_OUT}" && break
  sleep 0.2
done

if ! grep -qE '^root[[:space:]].*nginx: master process' <<<"${PS_OUT}"; then
  echo "ps aux (first 60 lines):"
  printf '%s\n' "${PS_OUT}" | sed -n '1,60p'
  fail "Could not find the nginx master process running as root."
fi
if ! grep -qE '^nginx[[:space:]].*nginx: worker process' <<<"${PS_OUT}"; then
  echo "ps aux (first 60 lines):"
  printf '%s\n' "${PS_OUT}" | sed -n '1,60p'
  fail "Could not find an nginx worker process running as the nginx user."
fi

echo "[6/6] HTTP sanity check..."
body="$(curl -s -m 5 "${URL}" || true)"
grep -q "process-user-ok" <<<"${body}" || fail "Unexpected HTTP response body."

pass "17 – Verify NGINX User & Processes"
