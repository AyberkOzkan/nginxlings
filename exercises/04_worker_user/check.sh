#!/usr/bin/env bash
# Checks exercise 04: unprivileged workers can read the web root the Dockerfile prepares.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="04_worker_user"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex04_worker_user_run}"
HOST_PORT="${HOST_PORT:-18040}"
URL="http://127.0.0.1:${HOST_PORT}/"

# The Dockerfile's web root: owner, group and mode must stay as shipped.
EXPECTED_PERMS=$'/srv/site webapp:webapp 750\n/srv/site/index.html webapp:webapp 640'

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

echo "[4/6] Checking that /srv/site keeps the Dockerfile's owner and mode..."
if ! PERMS="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" \
  stat -c '%n %U:%G %a' /srv/site /srv/site/index.html 2>&1)"; then
  printf '%s\n' "${PERMS}"
  fail "Could not inspect /srv/site in the image. Keep the Dockerfile as it is."
fi
if [[ "${PERMS}" != "${EXPECTED_PERMS}" ]]; then
  echo "Found:"
  printf '%s\n' "${PERMS}"
  echo "Expected:"
  printf '%s\n' "${EXPECTED_PERMS}"
  fail "The owner or mode of /srv/site changed. Keep the Dockerfile as it is and solve this in nginx.conf."
fi

echo "[5/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
wait_container_http "${CONTAINER_NAME}" "${URL}"

echo "[6/6] Checking the worker account and the response..."
# Workers are forked right after the master starts; poll instead of sleeping.
PS_OUT=""
for _ in $(seq 1 50); do
  PS_OUT="$("${DOCKER_BIN}" exec "${CONTAINER_NAME}" ps -o user,args 2>/dev/null || true)"
  grep -q 'nginx: worker process' <<<"${PS_OUT}" && break
  sleep 0.2
done
WORKERS="$(grep 'nginx: worker process' <<<"${PS_OUT}" || true)"
if [[ -z "${WORKERS}" ]]; then
  printf '%s\n' "${PS_OUT}"
  fail "No nginx worker process found in the container."
fi
echo "Worker processes (account, command):"
printf '%s\n' "${WORKERS}"
if grep -qE '^root[[:space:]]' <<<"${WORKERS}"; then
  fail "The workers run as root. Choose an unprivileged account that can read /srv/site."
fi

temp_file BODY
CODE="$(curl -s -m 5 -o "${BODY}" -w '%{http_code}' "${URL}" || true)"
if [[ "${CODE}" == "403" ]]; then
  echo "Error log:"
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | grep -E 'forbidden|Permission denied' | tail -n 3 || true
  fail "403 Forbidden: the workers cannot read /srv/site. Compare their account with the owner, group and mode of the files."
fi
[[ "${CODE}" == "200" ]] || fail "Expected 200 from ${URL}, got ${CODE:-no answer}."
grep -q 'worker-user-ok' "${BODY}" ||
  fail "/ answered 200 but not with the page from /srv/site/index.html."

pass "04 – Worker User & Permissions"
