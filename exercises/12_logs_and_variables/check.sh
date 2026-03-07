#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"

IMAGE_NAME="${IMAGE_NAME:-nginxlings_ex12_logs_vars}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex12_logs_vars_run}"
HOST_PORT="${HOST_PORT:-18120}"

UA="nginxlings-test-agent/1.0"

PREFIX="$(pwd)"

TMP_T=""
CONF_T=""
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

  "${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

  if [ "${BUILT_IMAGE}" = "1" ]; then
    "${DOCKER_BIN}" image rm -f "${IMAGE_NAME}" >/dev/null 2>&1 || true
  fi

  "${DOCKER_BIN}" builder prune -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

command -v "${DOCKER_BIN}" >/dev/null 2>&1 || fail "docker not found in PATH."
command -v curl >/dev/null 2>&1 || fail "curl not found in PATH."

[ -f "${PREFIX}/nginx.conf" ] || fail "nginx.conf not found in ${PREFIX}"
[ -f "${PREFIX}/Dockerfile" ] || fail "Dockerfile not found in ${PREFIX}"

info "[1/6] Building Docker image..."
DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build -t "${IMAGE_NAME}" "${PREFIX}" >/dev/null 2>&1 \
  || fail "docker build failed."
BUILT_IMAGE="1"

info "[2/6] Testing nginx config inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! "${DOCKER_BIN}" run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}" || true
  fail "nginx -t failed."
fi
rm -f "${TMP_T}" >/dev/null 2>&1 || true
TMP_T=""
info "[OK] nginx -t succeeded."

info "[3/6] Starting container..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run -d --name "${CONTAINER_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container."

sleep 1

info "[4/6] Sending HTTP requests..."
for _ in 1 2 3 4 5; do
  curl -s -H "User-Agent: ${UA}" "http://127.0.0.1:${HOST_PORT}/" >/dev/null || {
    echo "docker logs (first 80 lines):"
    "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | sed -n '1,80p' || true
    fail "HTTP request failed."
  }
done
sleep 1

info "[5/6] Extracting effective config (nginx -T) and validating directives..."

CONF_T="$(mktemp)"
if ! "${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc "nginx -T 2>&1" > "${CONF_T}"; then
  echo "nginx -T output (first 120 lines):"
  sed -n '1,120p' "${CONF_T}" || true
  fail "Could not run nginx -T inside container."
fi

# Must have log_format main somewhere (allow end-of-line)
grep -qE '^[[:space:]]*log_format[[:space:]]+main([[:space:]]|$)' "${CONF_T}" \
  || { echo "nginx -T output (first 120 lines):"; sed -n '1,120p' "${CONF_T}" || true; fail "log_format main not found in effective config."; }

# Required variables must appear somewhere in effective config
grep -q '\$remote_addr' "${CONF_T}"     || fail "Missing \$remote_addr in effective config."
grep -q '\$status' "${CONF_T}"          || fail "Missing \$status in effective config."
grep -q '\$request' "${CONF_T}"         || fail "Missing \$request in effective config."
grep -q '\$http_user_agent' "${CONF_T}" || fail "Missing \$http_user_agent in effective config."

# Must have access_log using main format (http or server context)
grep -qE '^[[:space:]]*access_log[[:space:]]+[^;]+[[:space:]]+main[[:space:]]*;' "${CONF_T}" \
  || { echo "nginx -T output (first 120 lines):"; sed -n '1,120p' "${CONF_T}" || true; fail "No access_log directive using format 'main' found in effective config."; }

rm -f "${CONF_T}" >/dev/null 2>&1 || true
CONF_T=""
info "[OK] log_format main and access_log main are present in effective config."

info "[6/6] Validating access log output (docker logs)..."
LOGS="$("${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>/dev/null || true)"

MATCHED="$(printf '%s\n' "${LOGS}" | grep -F "${UA}" || true)"
if [ -z "${MATCHED}" ]; then
  echo "Sample logs (first 120 lines):"
  printf '%s\n' "${LOGS}" | sed -n '1,120p' || true
  fail "No access log line contains the test User-Agent. Make sure access_log is enabled and logs include \$http_user_agent."
fi

echo "${MATCHED}" | grep -q "GET /" || {
  echo "Matched UA lines:"
  echo "${MATCHED}" | sed -n '1,30p' || true
  fail "UA-matched lines do not contain 'GET /'. Make sure \$request is included."
}

echo "${MATCHED}" | grep -Eq '(^|[[:space:]])200([[:space:]]|$)' || {
  echo "Matched UA lines:"
  echo "${MATCHED}" | sed -n '1,30p' || true
  fail "UA-matched lines do not contain HTTP status 200. Make sure \$status is included."
}

echo "${MATCHED}" | grep -Eq '([0-9]{1,3}\.){3}[0-9]{1,3}' || {
  echo "Matched UA lines:"
  echo "${MATCHED}" | sed -n '1,30p' || true
  fail "UA-matched lines do not contain an IP address. Make sure \$remote_addr is included."
}

echo
echo "[PASS] 12 – Logs & Variables looks good."