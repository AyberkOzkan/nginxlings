#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
EXERCISE_ID="17_verify_nginx_user"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"

IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex17_verify_user_run}"
HOST_PORT="${HOST_PORT:-18170}"

PREFIX="$(pwd)"

TMP_T=""
CONF_T=""
BUILT_IMAGE="0"

fail() { echo "[FAIL] $*"; exit 1; }
info() { echo "$@"; }

cleanup() {
  set +e
  [ -n "${TMP_T}" ] && rm -f "${TMP_T}" >/dev/null 2>&1 || true
  [ -n "${CONF_T}" ] && rm -f "${CONF_T}" >/dev/null 2>&1 || true

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

info "[1/6] Building Docker image..."
DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build --label "${DOCKER_LABEL}" -t "${IMAGE_NAME}" "${PREFIX}" >/dev/null 2>&1 \
  || fail "docker build failed."
BUILT_IMAGE="1"

info "[2/6] Testing nginx config inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}" || true
  fail "nginx -t failed."
fi
rm -f "${TMP_T}" >/dev/null 2>&1 || true
TMP_T=""
info "[OK] nginx -t succeeded."

info "[3/6] Starting container..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container."
sleep 1

info "[4/6] Checking effective config for user directive..."
CONF_T="$(mktemp)"
if ! "${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc "nginx -T 2>&1" > "${CONF_T}"; then
  echo "nginx -T output (first 160 lines):"
  sed -n '1,160p' "${CONF_T}" || true
  fail "Could not run nginx -T."
fi

grep -qE '^[[:space:]]*user[[:space:]]+nginx[[:space:]]*;' "${CONF_T}" \
  || fail "Missing or incorrect user directive. Expected: user nginx;"

rm -f "${CONF_T}" >/dev/null 2>&1 || true
CONF_T=""
info "[OK] user nginx; found in effective config."

info "[5/6] Verifying master vs worker processes (ps aux)..."
PS_OUT="$("${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc "ps aux" 2>/dev/null || true)"

# Master usually runs as root in container
echo "${PS_OUT}" | grep -qiE 'root[[:space:]].*nginx: master process' \
  || {
    echo "ps aux (first 60 lines):"
    printf '%s\n' "${PS_OUT}" | sed -n '1,60p' || true
    fail "Could not find nginx master process running as root."
  }

# Worker should run as nginx because of user nginx;
echo "${PS_OUT}" | grep -qiE '^nginx[[:space:]].*nginx: worker process' \
  || {
    echo "ps aux (first 80 lines):"
    printf '%s\n' "${PS_OUT}" | sed -n '1,80p' || true
    fail "Could not find nginx worker process running as nginx."
  }

info "[OK] Master/root and worker/nginx processes detected."

info "[6/6] HTTP sanity check..."
body="$(curl -s -m 2 "http://127.0.0.1:${HOST_PORT}/" || true)"
echo "${body}" | grep -q "process-user-ok" || fail "Unexpected HTTP response body."

echo
echo "[PASS] 17 – Verify NGINX User & Processes looks good."