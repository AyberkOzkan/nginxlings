#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
IMAGE_TAG="${IMAGE_TAG:-nginxlings-05-install-verify}"

CID=""
TMP_BUILD_LOG=""
TMP_RUN_LOG=""
BUILT_IMAGE="0"

fail() {
  echo "[FAIL] $*"
  exit 1
}

cleanup() {
  set +e

  # remove temp files
  [ -n "${TMP_BUILD_LOG}" ] && rm -f "${TMP_BUILD_LOG}" >/dev/null 2>&1 || true
  [ -n "${TMP_RUN_LOG}" ] && rm -f "${TMP_RUN_LOG}" >/dev/null 2>&1 || true

  # stop/remove container if started
  if [ -n "${CID}" ]; then
    "${DOCKER_BIN}" rm -f "${CID}" >/dev/null 2>&1 || true
  fi

  # remove the exercise image to avoid disk growth
  if [ "${BUILT_IMAGE}" = "1" ]; then
    "${DOCKER_BIN}" image rm -f "${IMAGE_TAG}" >/dev/null 2>&1 || true
  fi

  # prune builder cache (prevents build cache from accumulating)
  "${DOCKER_BIN}" builder prune -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "[1/4] Checking if docker is available..."
if ! command -v "${DOCKER_BIN}" >/dev/null 2>&1; then
  fail "docker command not found. Please install and configure Docker."
fi
echo "[OK] docker command found."

echo "[2/4] Building image from Dockerfile..."
TMP_BUILD_LOG="$(mktemp)"
if ! "${DOCKER_BIN}" build -t "${IMAGE_TAG}" . >"${TMP_BUILD_LOG}" 2>&1; then
  echo "docker build output (first 80 lines):"
  sed -n '1,80p' "${TMP_BUILD_LOG}" || true
  fail "docker build failed."
fi
BUILT_IMAGE="1"
echo "[OK] docker build succeeded (image: ${IMAGE_TAG})"

echo "[3/4] Checking nginx -v and nginx -V inside the container..."
TMP_RUN_LOG="$(mktemp)"

# nginx -v (prints to stderr)
if ! "${DOCKER_BIN}" run --rm "${IMAGE_TAG}" nginx -v >"${TMP_RUN_LOG}" 2>&1; then
  echo "nginx -v output:"
  cat "${TMP_RUN_LOG}" || true
  fail "nginx -v failed inside the container."
fi
VERSION_LINE="$(head -n 1 "${TMP_RUN_LOG}" || true)"
if [ -z "${VERSION_LINE}" ]; then
  echo "nginx -v output:"
  cat "${TMP_RUN_LOG}" || true
  fail "nginx -v did not print a version line."
fi
echo "[OK] nginx -v => ${VERSION_LINE}"

# nginx -V (prints to stderr)
: >"${TMP_RUN_LOG}"
if ! "${DOCKER_BIN}" run --rm "${IMAGE_TAG}" nginx -V >"${TMP_RUN_LOG}" 2>&1; then
  echo "nginx -V output:"
  cat "${TMP_RUN_LOG}" || true
  fail "nginx -V failed inside the container."
fi

if ! grep -q "configure arguments:" "${TMP_RUN_LOG}"; then
  echo "nginx -V output (first 80 lines):"
  sed -n '1,80p' "${TMP_RUN_LOG}" || true
  fail "nginx -V output does not contain a 'configure arguments:' line."
fi
echo "[OK] nginx -V returned configuration and module information."

echo "[4/4] Verifying that nginx can serve HTTP on port 80..."

if ! command -v curl >/dev/null 2>&1; then
  fail "curl command not found. Please install curl to run this check."
fi

CID="$("${DOCKER_BIN}" run -d -P "${IMAGE_TAG}" 2>/dev/null || true)"
if [ -z "${CID}" ]; then
  fail "docker run -d -P ${IMAGE_TAG} failed. Check the Dockerfile CMD/ENTRYPOINT."
fi

PORT_LINE="$("${DOCKER_BIN}" port "${CID}" 80/tcp 2>/dev/null | head -n 1 || true)"
if [ -z "${PORT_LINE}" ]; then
  "${DOCKER_BIN}" logs "${CID}" 2>&1 | sed -n '1,80p' || true
  fail "Could not determine mapped host port for container (docker port returned empty)."
fi

HOST_PORT="${PORT_LINE##*:}"
if [ -z "${HOST_PORT}" ] || [ "${HOST_PORT}" = "${PORT_LINE}" ]; then
  echo "docker port output: ${PORT_LINE}"
  fail "Failed to parse host port from docker port output."
fi

sleep 2

if ! curl -sSf "http://127.0.0.1:${HOST_PORT}/" >/dev/null 2>&1; then
  echo "curl http://127.0.0.1:${HOST_PORT}/ failed."
  echo "docker logs (first 80 lines):"
  "${DOCKER_BIN}" logs "${CID}" 2>&1 | sed -n '1,80p' || true
  fail "nginx did not serve a successful HTTP response on /."
fi

echo "[OK] nginx served an HTTP response on / via host port ${HOST_PORT}."

echo
echo "[PASS] 05 – Install & Verify from Official Repo looks good."