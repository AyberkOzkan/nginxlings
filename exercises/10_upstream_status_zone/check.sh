#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="nginxlings_upstream_zone_10"
LB_NAME="nginxlings_upstream_zone_10_lb"
NET_NAME="nginxlings_ex10_net"

B1_NAME="ex10_backend1"
B2_NAME="ex10_backend2"

HOST_PORT="${HOST_PORT:-19100}"

fail() { echo "[FAIL] $*"; exit 1; }
info() { echo "$@"; }

command -v docker >/dev/null 2>&1 || fail "docker command not found."
command -v curl  >/dev/null 2>&1 || fail "curl command not found."

[ -f Dockerfile ] || fail "Dockerfile not found."
[ -f nginx.conf ] || fail "nginx.conf not found."

cleanup() {
  docker rm -f "${LB_NAME}" "${B1_NAME}" "${B2_NAME}" >/dev/null 2>&1 || true
  docker network rm "${NET_NAME}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

info "[1/7] Building Docker image (${IMAGE_NAME})..."
docker build -t "${IMAGE_NAME}" . >/dev/null

info "[2/7] Validating nginx.conf inside the image (nginx -t)..."
TMP_T="$(mktemp)"
if ! docker run --rm "${IMAGE_NAME}" nginx -t >/dev/null 2>"${TMP_T}"; then
  echo "nginx -t output:"
  cat "${TMP_T}"
  rm -f "${TMP_T}"
  fail "nginx -t failed inside the container."
fi
rm -f "${TMP_T}"
info "[OK] nginx -t succeeded."

info "[3/7] Inspecting effective config (nginx -T) for required directives..."
TMP_CONF="$(mktemp)"
docker run --rm "${IMAGE_NAME}" nginx -T >"${TMP_CONF}" 2>&1 || true

grep -qE "zone[[:space:]]+up_mem[[:space:]]+64k;" "${TMP_CONF}" \
  || { rm -f "${TMP_CONF}"; fail "Missing: zone up_mem 64k;"; }

grep -qE "add_header[[:space:]]+X-Upstream[[:space:]]+\\\$upstream_addr" "${TMP_CONF}" \
  || { rm -f "${TMP_CONF}"; fail "Missing: add_header X-Upstream \$upstream_addr always;"; }

rm -f "${TMP_CONF}"
info "[OK] Required upstream zone and header directives found."

info "[4/7] Creating Docker network (${NET_NAME})..."
docker network rm "${NET_NAME}" >/dev/null 2>&1 || true
docker network create "${NET_NAME}" >/dev/null

info "[5/7] Starting two backend containers..."

docker rm -f "${B1_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${B1_NAME}" --network "${NET_NAME}" \
  busybox sh -lc "mkdir -p /www && echo 'backend=8081' > /www/index.html && httpd -f -p 8081 -h /www" \
  >/dev/null

docker rm -f "${B2_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${B2_NAME}" --network "${NET_NAME}" \
  busybox sh -lc "mkdir -p /www && echo 'backend=8082' > /www/index.html && httpd -f -p 8082 -h /www" \
  >/dev/null

sleep 1

info "[6/7] Starting NGINX LB container (host ${HOST_PORT} -> container 8080)..."
docker rm -f "${LB_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${LB_NAME}" --network "${NET_NAME}" -p "${HOST_PORT}:8080" "${IMAGE_NAME}" >/dev/null

sleep 2

get_upstream() {
  curl -s -m 2 -o /dev/null -D - -w "HTTP_CODE:%{http_code}\n" "http://127.0.0.1:${HOST_PORT}/" \
    | awk -F': ' '
      /^X-Upstream:/ {gsub(/\r/,"",$2); up=$2}
      /^HTTP_CODE:/ {code=$2}
      END {
        if (code == "") code="000"
        if (up == "") up="-"
        print code "|" up
      }'
}

info "[7/7] Verifying distribution, then stopping backend2 and verifying failover..."

before_8081=0
before_8082=0

for i in 1 2 3 4 5 6 7 8 9 10; do
  r="$(get_upstream || true)"
  up="${r#*|}"
  echo "${up}" | grep -q ":8081" && before_8081=$((before_8081+1)) || true
  echo "${up}" | grep -q ":8082" && before_8082=$((before_8082+1)) || true
done

if [ "${before_8081}" -eq 0 ] || [ "${before_8082}" -eq 0 ]; then
  fail "Before stopping, expected both upstream ports to appear. Got 8081=${before_8081}, 8082=${before_8082}."
fi
info "[OK] Distribution observed: 8081=${before_8081}, 8082=${before_8082}"

info "[INFO] Stopping backend2 container (${B2_NAME})..."
docker rm -f "${B2_NAME}" >/dev/null 2>&1 || true
sleep 1

after_8081=0
after_only_8082=0

for i in 1 2 3 4 5 6 7 8 9 10 11 12; do
  r="$(get_upstream || true)"         # CODE|UPSTREAM
  code="${r%%|*}"
  up="${r#*|}"

  if [ "${i}" -le 3 ]; then
    echo "[DEBUG] after-stop req ${i}: code=${code} upstream=${up}"
  fi

  # Success if 8081 appears anywhere (including retry chains)
  echo "${up}" | grep -q ":8081" && after_8081=$((after_8081+1)) || true

  # Reject only-8082 (no retry to 8081)
  if echo "${up}" | grep -q ":8082" && ! echo "${up}" | grep -q ":8081"; then
    after_only_8082=$((after_only_8082+1))
  fi
done

info "[INFO] After-stop summary: 8081=${after_8081}, only-8082=${after_only_8082}"

if [ "${after_8081}" -eq 0 ]; then
  fail "After stopping backend2, expected upstream port 8081 to be used."
fi

if [ "${after_only_8082}" -ne 0 ]; then
  fail "After stopping backend2, some requests were served only by 8082 (no retry to 8081)."
fi

echo
echo "[PASS] 10 – Upstream Status in Shared Memory (zone up_mem) looks good."
