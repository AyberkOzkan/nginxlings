#!/usr/bin/env bash
# Checks exercise 10: an upstream with a shared memory zone, X-Upstream header and failover.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="10_upstream_status_zone"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
# Pinned by digest; see "Container Images" in CONTRIBUTING.md to update it.
BUSYBOX_IMAGE="${BUSYBOX_IMAGE:-busybox:1.38.0@sha256:fd7dc98638c8e305f4dc34e979f1c0fdfdcaeb0fbf8fcff77ae834b6da3d7e6e}"
LB_NAME="${LB_NAME:-nginxlings_upstream_zone_10_lb}"
NET_NAME="${NET_NAME:-nginxlings_ex10_net}"
# The learner's nginx.conf resolves these container names, so they are fixed.
B1_NAME="ex10_backend1"
B2_NAME="ex10_backend2"
HOST_PORT="${HOST_PORT:-19100}"
URL="http://127.0.0.1:${HOST_PORT}/"
# Refused connections fail fast, so a short per-request timeout is enough
# and keeps the failover phase quick when a request does hang.
REQUEST_TIMEOUT=3

# get_upstream prints "CODE|X-Upstream" for one request through the LB.
get_upstream() {
  curl -s -m "${REQUEST_TIMEOUT}" -o /dev/null -D - -w "HTTP_CODE:%{http_code}\n" "${URL}" |
    awk -F': ' '
      /^X-Upstream:/ {gsub(/\r/,"",$2); up=$2}
      /^HTTP_CODE:/ {code=substr($0, 11)}
      END {
        if (code == "") code="000"
        if (up == "") up="-"
        print code "|" up
      }' || true
}

# wait_backend NAME PORT waits until the busybox backend answers inside its container.
wait_backend() {
  local name=$1 port=$2 i
  for ((i = 0; i < 50; i++)); do
    if [[ "$("${DOCKER_BIN}" inspect -f '{{.State.Running}}' "${name}" 2>/dev/null || true)" != "true" ]]; then
      echo "Container ${name} is not running. Logs:"
      "${DOCKER_BIN}" logs "${name}" 2>&1 | tail -n 40 || true
      fail "Backend container ${name} exited during startup; see the logs above."
    fi
    if "${DOCKER_BIN}" exec "${name}" wget -q -O /dev/null "http://127.0.0.1:${port}/" 2>/dev/null; then
      return 0
    fi
    sleep 0.2
  done
  "${DOCKER_BIN}" logs "${name}" 2>&1 | tail -n 40 || true
  fail "Backend ${name} did not answer on port ${port} within 10s."
}

echo "[1/7] Checking prerequisites and files..."
require_docker
require_cmd curl
for file in Dockerfile nginx.conf; do
  [[ -f "${EX_DIR}/${file}" ]] || fail "${file} not found in ${EX_DIR}."
done
require_free_port "${HOST_PORT}" "the load balancer container (override with HOST_PORT)"

echo "[2/7] Building Docker image ${IMAGE_NAME}..."
on_exit "\"${DOCKER_BIN}\" image rm -f \"${IMAGE_NAME}\""
docker_build "${IMAGE_NAME}" "${EX_DIR}"

echo "[3/7] Validating nginx.conf inside the image (nginx -t)..."
if ! T_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -t 2>&1)"; then
  fail_nginx_rejected - <<<"${T_OUTPUT}"
fi
if ! CONF_OUTPUT="$("${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T 2>&1)"; then
  printf '%s\n' "${CONF_OUTPUT}"
  fail "nginx -T failed inside the image."
fi
CONF="$(active_config <<<"${CONF_OUTPUT}")"
if ! grep -qE '^[[:space:]]*zone[[:space:]]+up_mem[[:space:]]+64k[[:space:]]*;' <<<"${CONF}"; then
  fail "No shared memory zone named up_mem with a size of 64k in the upstream block yet."
fi
# shellcheck disable=SC2016 # the pattern is a regex, not a shell expansion
if ! grep -qE '^[[:space:]]*add_header[[:space:]]+X-Upstream[[:space:]]+\$upstream_addr([[:space:]]|;)' <<<"${CONF}"; then
  fail "No active X-Upstream response header set to \$upstream_addr yet."
fi

echo "[4/7] Creating Docker network ${NET_NAME}..."
on_exit "\"${DOCKER_BIN}\" network rm \"${NET_NAME}\""
on_exit "\"${DOCKER_BIN}\" rm -f \"${LB_NAME}\" \"${B1_NAME}\" \"${B2_NAME}\""
"${DOCKER_BIN}" rm -f "${LB_NAME}" "${B1_NAME}" "${B2_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" network rm "${NET_NAME}" >/dev/null 2>&1 || true
"${DOCKER_BIN}" network create --label "${DOCKER_LABEL}" "${NET_NAME}" >/dev/null

echo "[5/7] Starting two backend containers..."
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d --name "${B1_NAME}" --network "${NET_NAME}" \
  "${BUSYBOX_IMAGE}" sh -c "mkdir -p /www && echo 'backend=8081' > /www/index.html && httpd -f -p 8081 -h /www" >/dev/null
# httpd runs in the background here so the check can stop it later while the
# container, and with it the backend's IP address, stays up.
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d --name "${B2_NAME}" --network "${NET_NAME}" \
  "${BUSYBOX_IMAGE}" sh -c "mkdir -p /www && echo 'backend=8082' > /www/index.html && httpd -p 8082 -h /www && exec sleep 86400" >/dev/null
wait_backend "${B1_NAME}" 8081
wait_backend "${B2_NAME}" 8082

echo "[6/7] Starting NGINX LB container (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${LB_NAME}" --network "${NET_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
wait_container_http "${LB_NAME}" "${URL}"

echo "[7/7] Verifying distribution, then stopping backend2 and verifying failover..."
before_8081=0
before_8082=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  r="$(get_upstream)"
  up="${r#*|}"
  if [[ "${up}" == *:8081* ]]; then before_8081=$((before_8081 + 1)); fi
  if [[ "${up}" == *:8082* ]]; then before_8082=$((before_8082 + 1)); fi
done
if ((before_8081 == 0 || before_8082 == 0)); then
  echo "Last response (code|X-Upstream): ${r}"
  echo "LB container logs:"
  "${DOCKER_BIN}" logs "${LB_NAME}" 2>&1 | tail -n 20 || true
  fail "Before stopping, expected both upstream ports in X-Upstream. Got 8081=${before_8081}, 8082=${before_8082}."
fi
echo "Distribution observed: 8081=${before_8081}, 8082=${before_8082}"

# Stop backend2's HTTP server but keep its container: connections to 8082 are
# now refused at once, the way a crashed backend process behaves.
"${DOCKER_BIN}" exec "${B2_NAME}" killall httpd
for ((i = 0; i < 25; i++)); do
  "${DOCKER_BIN}" exec "${B2_NAME}" wget -q -O /dev/null "http://127.0.0.1:8082/" 2>/dev/null || break
  sleep 0.2
done

# Every request must still succeed, answered by 8081 (possibly after a retry
# that shows up as "...:8082, ...:8081").
after_bad=0
bad_sample=""
for _ in 1 2 3 4 5 6 7 8 9 10 11 12; do
  r="$(get_upstream)"
  if [[ "${r%%|*}" != "200" || "${r#*|}" != *:8081* ]]; then
    after_bad=$((after_bad + 1))
    bad_sample="${r}"
  fi
done
if ((after_bad != 0)); then
  echo "Last failing response (code|X-Upstream): ${bad_sample}"
  fail "After stopping backend2, ${after_bad} of 12 requests did not succeed through 8081. NGINX should retry the next server when a connection is refused."
fi
echo "Failover observed: all 12 requests answered by 8081."

pass "10 – Upstream State in Shared Memory (zone up_mem)"
