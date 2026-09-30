#!/usr/bin/env bash
# Checks exercise 18: NGINX balances HTTP (L7) and raw TCP (L4) across two backends each.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="18_osi_layers_usecases"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
PYTHON_BIN="${PYTHON_BIN:-python3}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex18_osi_mapping_run}"
HTTP_PORT="${HTTP_PORT:-18180}"
TCP_PORT="${TCP_PORT:-19080}"

echo "[1/6] Checking prerequisites and files..."
require_docker
require_cmd curl
require_cmd "${PYTHON_BIN}" "set PYTHON_BIN to override"
for file in Dockerfile nginx.conf start.sh; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HTTP_PORT}" "the L7 listener (override with HTTP_PORT)"
require_free_port "${TCP_PORT}" "the L4 listener (override with TCP_PORT)"

echo "[2/6] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/6] Testing nginx config and its L7/L4 parts inside the image (nginx -T)..."
temp_file DUMP
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T >"${DUMP}" 2>&1; then
  if grep -q 'unknown directive "stream"' "${DUMP}"; then
    cat "${DUMP}"
    fail "NGINX does not know the stream directive yet: load the stream module with load_module at the top of nginx.conf (the README names the module file)."
  fi
  fail_nginx_rejected "${DUMP}"
fi
# Effective configuration without comment lines; these give quick, specific
# hints before the behaviour checks below.
temp_file ACTIVE
active_config "${DUMP}" >"${ACTIVE}"
grep -qE '^[[:space:]]*load_module[[:space:]]+[^;]*ngx_stream_module\.so' "${ACTIVE}" ||
  fail "The stream module is not loaded yet: add load_module at the top of nginx.conf (the README names the module file)."
grep -qE '^[[:space:]]*stream[[:space:]]*\{' "${ACTIVE}" ||
  fail "No stream {} block yet: add one at the top level, next to http {}, that forwards port 9000 to the TCP backends."
grep -qiE '^[[:space:]]*add_header[[:space:]]+X-L7-Upstream[[:space:]]' "${ACTIVE}" ||
  fail "No X-L7-Upstream header yet: in location /l7, add it with the address of the backend that answered."

echo "[4/6] Starting container ${CONTAINER_NAME} (host ${HTTP_PORT} -> 8080, ${TCP_PORT} -> 9000)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HTTP_PORT}:8080" \
  -p "127.0.0.1:${TCP_PORT}:9000" \
  "${IMAGE_NAME}" >/dev/null
wait_container_http "${CONTAINER_NAME}" "http://127.0.0.1:${HTTP_PORT}/l7"

# nginx answers before the python/socat backends (started by start.sh) may be
# listening; wait for the backends through the proxy instead of a fixed sleep.
echo "[5/6] Checking L7 HTTP upstream distribution..."
l7_8081=0
l7_8082=0
l7_seen=0
plain_200=0
for ((n = 0; n < 60; n++)); do
  hdrs="$(curl -s -m 5 -D - -o /dev/null "http://127.0.0.1:${HTTP_PORT}/l7" | tr -d '\r' || true)"
  up="$(awk -F': ' 'tolower($1) == "x-l7-upstream" {print $2}' <<<"${hdrs}" | tail -n 1)"
  if [[ -n "${up}" ]]; then l7_seen=$((l7_seen + 1)); fi
  # A 200 without the header means NGINX answered /l7 itself: backends that
  # are still starting would give 502, not 200.
  if [[ -z "${up}" && "$(awk 'NR == 1 {print $2}' <<<"${hdrs}")" == "200" ]]; then
    plain_200=$((plain_200 + 1))
    ((plain_200 < 3)) || break
  fi
  if [[ "${up}" == *:8081* ]]; then l7_8081=$((l7_8081 + 1)); fi
  if [[ "${up}" == *:8082* ]]; then l7_8082=$((l7_8082 + 1)); fi
  if ((l7_8081 > 0 && l7_8082 > 0)); then break; fi
  sleep 0.2
done
if ((l7_8081 == 0 || l7_8082 == 0)); then
  echo "Last response headers:"
  printf '%s\n' "${hdrs}"
  if ((l7_seen == 0)); then
    fail "/l7 answered without an X-L7-Upstream value. Is /l7 proxied to the HTTP upstream with no return left in it, and is add_header inside location /l7 itself?"
  fi
  fail "Expected both HTTP backends to appear in the X-L7-Upstream header. Got 8081=${l7_8081}, 8082=${l7_8082}."
fi
echo "L7 distribution observed: 8081=${l7_8081}, 8082=${l7_8082}"

echo "[6/6] Checking L4 TCP upstream distribution..."
if ! wait_tcp 127.0.0.1 "${TCP_PORT}" 15; then
  "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 40 || true
  fail "Nothing accepted TCP connections on port ${TCP_PORT} (is the stream block configured?)."
fi
temp_file RESP_FILE
for _ in $(seq 1 40); do
  "${PYTHON_BIN}" - "${TCP_PORT}" >>"${RESP_FILE}" 2>/dev/null <<'PY' || true
import socket
import sys

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.settimeout(2.0)
s.connect(("127.0.0.1", int(sys.argv[1])))
data = s.recv(1024)
s.close()
print(data.decode("utf-8", "ignore").strip())
PY
  # Stop as soon as both TCP backends have answered (socat may start late).
  if grep -q '^l4-backend=9001$' "${RESP_FILE}" && grep -q '^l4-backend=9002$' "${RESP_FILE}"; then break; fi
  sleep 0.1
done

l4_9001="$(grep -c '^l4-backend=9001$' "${RESP_FILE}" || true)"
l4_9002="$(grep -c '^l4-backend=9002$' "${RESP_FILE}" || true)"
if ((l4_9001 == 0 || l4_9002 == 0)); then
  echo "TCP responses:"
  cat "${RESP_FILE}"
  fail "Expected both TCP backends to appear. Got 9001=${l4_9001}, 9002=${l4_9002}."
fi
echo "L4 distribution observed: 9001=${l4_9001}, 9002=${l4_9002}"

pass "18 – OSI Layers & Use-Case Mapping"
