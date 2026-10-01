#!/usr/bin/env bash
# Checks exercise 05: the image installs NGINX from the distribution package and serves HTTP.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="05_install_verify"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_TAG="${IMAGE_TAG:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_install_verify_05_run}"

echo "[1/4] Checking prerequisites..."
require_docker
require_cmd curl
[[ -f "${EX_DIR}/Dockerfile" ]] || fail "Dockerfile not found in ${EX_DIR}."

echo "[2/4] Building image from Dockerfile..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_TAG}\""
docker_build "${IMAGE_TAG}" "${EX_DIR}"

echo "[3/4] Checking the NGINX installation inside the image..."
temp_file OUT
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm --entrypoint sh "${IMAGE_TAG}" -c 'command -v nginx' >"${OUT}" 2>&1; then
  cat "${OUT}"
  fail "The image has no nginx yet. Install it with apt-get in the Dockerfile."
fi

if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_TAG}" nginx -v >"${OUT}" 2>&1; then
  cat "${OUT}"
  fail "nginx -v failed inside the container."
fi
VERSION_LINE="$(head -n 1 "${OUT}")"
[[ -n "${VERSION_LINE}" ]] || fail "nginx -v did not print a version line."
echo "nginx -v => ${VERSION_LINE}"

if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_TAG}" nginx -V >"${OUT}" 2>&1; then
  cat "${OUT}"
  fail "nginx -V failed inside the container."
fi
if ! grep -q "configure arguments:" "${OUT}"; then
  sed -n '1,80p' "${OUT}"
  fail "nginx -V output does not contain a 'configure arguments:' line."
fi

# The exercise is about installing from the distribution repository.
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm --entrypoint dpkg "${IMAGE_TAG}" -s nginx >"${OUT}" 2>&1 ||
  ! grep -q '^Status: install ok installed' "${OUT}"; then
  cat "${OUT}"
  fail "NGINX is not installed as a package (dpkg -s nginx failed). Install it with apt-get install nginx."
fi

echo "[4/4] Verifying that nginx serves HTTP on port 80..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d --name "${CONTAINER_NAME}" -p 127.0.0.1::80 "${IMAGE_TAG}" >"${OUT}" 2>&1; then
  cat "${OUT}"
  fail "docker run failed. Check the Dockerfile CMD/ENTRYPOINT."
fi

PORT_LINE="$("${DOCKER_BIN}" port "${CONTAINER_NAME}" 80/tcp 2>/dev/null | head -n 1 || true)"
if [[ -z "${PORT_LINE}" ]]; then
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
  fail "Could not determine the mapped host port (is port 80 exposed and NGINX running?)."
fi
HOST_PORT="${PORT_LINE##*:}"
URL="http://127.0.0.1:${HOST_PORT}/"
wait_container_http "${CONTAINER_NAME}" "${URL}"

if ! curl -sSf -m 5 -o /dev/null "${URL}"; then
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
  fail "nginx did not serve a successful HTTP response on ${URL}."
fi

pass "05 – Install & Verify from Official Repo looks good."
