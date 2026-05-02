#!/usr/bin/env bash
set -euo pipefail

DOCKER_BIN="${DOCKER_BIN:-docker}"
PYTHON_BIN="${PYTHON_BIN:-python3}"

IMAGE_NAME="${IMAGE_NAME:-nginxlings_ex18_osi_mapping}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex18_osi_mapping_run}"

HTTP_PORT="${HTTP_PORT:-18180}"
TCP_PORT="${TCP_PORT:-19080}"

PREFIX="$(pwd)"

TMP_T=""
CONF_T=""
RESP_FILE=""
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
  [ -n "${RESP_FILE}" ] && rm -f "${RESP_FILE}" >/dev/null 2>&1 || true

  "${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

  if [ "${BUILT_IMAGE}" = "1" ]; then
    "${DOCKER_BIN}" image rm -f "${IMAGE_NAME}" >/dev/null 2>&1 || true
  fi

  "${DOCKER_BIN}" builder prune -f >/dev/null 2>&1 || true
}
trap cleanup EXIT

command -v "${DOCKER_BIN}" >/dev/null 2>&1 || fail "docker not found in PATH."
command -v curl >/dev/null 2>&1 || fail "curl not found in PATH."
command -v "${PYTHON_BIN}" >/dev/null 2>&1 || fail "python3 not found (PYTHON_BIN=${PYTHON_BIN})."

[ -f "${PREFIX}/Dockerfile" ] || fail "Dockerfile not found."
[ -f "${PREFIX}/nginx.conf" ] || fail "nginx.conf not found."
[ -f "${PREFIX}/start.sh" ] || fail "start.sh not found."

info "[1/7] Building Docker image..."
DOCKER_BUILDKIT=0 "${DOCKER_BIN}" build -t "${IMAGE_NAME}" "${PREFIX}" >/dev/null 2>&1 \
  || fail "docker build failed."
BUILT_IMAGE="1"

info "[2/7] Testing nginx config inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! "${DOCKER_BIN}" run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}" || true
  fail "nginx -t failed."
fi
rm -f "${TMP_T}" >/dev/null 2>&1 || true
TMP_T=""
info "[OK] nginx -t succeeded."

info "[3/7] Starting container..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" run -d --name "${CONTAINER_NAME}" \
  -p "${HTTP_PORT}:8080" \
  -p "${TCP_PORT}:9000" \
  "${IMAGE_NAME}" >/dev/null \
  || fail "Failed to start container."

sleep 2

info "[4/7] Checking L7 HTTP upstream distribution..."
l7_8081=0
l7_8082=0

for _ in 1 2 3 4 5 6 7 8; do
  hdrs="$(curl -s -D - -o /tmp/ex18_body.$$ "http://127.0.0.1:${HTTP_PORT}/l7" | tr -d '\r' || true)"
  up="$(printf '%s\n' "${hdrs}" | awk -F': ' '/^X-L7-Upstream:/ {print $2}' | tail -n 1)"
  printf '%s' "${up}" | grep -q ":8081" && l7_8081=$((l7_8081 + 1)) || true
  printf '%s' "${up}" | grep -q ":8082" && l7_8082=$((l7_8082 + 1)) || true
done
rm -f /tmp/ex18_body.$$ >/dev/null 2>&1 || true

if [ "${l7_8081}" -eq 0 ] || [ "${l7_8082}" -eq 0 ]; then
  fail "Expected both HTTP backends to appear in X-L7-Upstream header. Got 8081=${l7_8081}, 8082=${l7_8082}."
fi
info "[OK] L7 distribution observed: 8081=${l7_8081}, 8082=${l7_8082}"

info "[5/7] Checking L4 TCP upstream distribution..."
RESP_FILE="$(mktemp)"
: > "${RESP_FILE}"

for _ in 1 2 3 4 5 6 7 8 9 10; do
  "${PYTHON_BIN}" - <<PY >> "${RESP_FILE}" 2>/dev/null
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.settimeout(2.0)
s.connect(("127.0.0.1", int("${TCP_PORT}")))
data = s.recv(1024)
s.close()
print(data.decode("utf-8", "ignore").strip())
PY
done

l4_9001="$(grep -c '^l4-backend=9001$' "${RESP_FILE}" || true)"
l4_9002="$(grep -c '^l4-backend=9002$' "${RESP_FILE}" || true)"

if [ "${l4_9001}" -eq 0 ] || [ "${l4_9002}" -eq 0 ]; then
  echo "TCP responses:"
  cat "${RESP_FILE}" || true
  fail "Expected both TCP backends to appear. Got 9001=${l4_9001}, 9002=${l4_9002}."
fi
info "[OK] L4 distribution observed: 9001=${l4_9001}, 9002=${l4_9002}"

info "[6/7] Validating effective config..."
CONF_T="$(mktemp)"
if ! "${DOCKER_BIN}" exec "${CONTAINER_NAME}" sh -lc "nginx -T 2>&1" > "${CONF_T}"; then
  echo "nginx -T output (first 200 lines):"
  sed -n '1,200p' "${CONF_T}" || true
  fail "Could not read effective config with nginx -T."
fi

grep -qE 'load_module[[:space:]]+/usr/lib/nginx/modules/ngx_stream_module\.so;' "${CONF_T}" \
  || fail "Missing stream module load line."

grep -qE 'upstream[[:space:]]+http_pool' "${CONF_T}" \
  || fail "Missing upstream http_pool."

grep -qE 'proxy_pass[[:space:]]+http://http_pool;' "${CONF_T}" \
  || fail "Missing proxy_pass http://http_pool;"

grep -qE 'add_header[[:space:]]+X-L7-Upstream[[:space:]]+\$upstream_addr' "${CONF_T}" \
  || fail "Missing X-L7-Upstream header."

grep -qE 'stream[[:space:]]*\{' "${CONF_T}" \
  || fail "Missing stream block."

grep -qE 'upstream[[:space:]]+tcp_pool' "${CONF_T}" \
  || fail "Missing upstream tcp_pool."

grep -qE 'proxy_pass[[:space:]]+tcp_pool;' "${CONF_T}" \
  || fail "Missing stream proxy_pass tcp_pool;"

rm -f "${CONF_T}" >/dev/null 2>&1 || true
CONF_T=""
info "[OK] Effective config contains both L7 and L4 structures."

info "[7/7] Done."
echo
echo "[PASS] 18 – OSI Layers & Use-Case Mapping looks good."