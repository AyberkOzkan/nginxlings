#!/usr/bin/env bash
# Checks exercise 12: a custom access log format is written to stdout.

EX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=SCRIPTDIR/../lib/check-lib.sh
. "${EX_DIR}/../lib/check-lib.sh"

EXERCISE_ID="12_logs_and_variables"
DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"
IMAGE_NAME="${IMAGE_NAME:-nginxlings/${EXERCISE_ID}:check}"
CONTAINER_NAME="${CONTAINER_NAME:-nginxlings_ex12_logs_vars_run}"
HOST_PORT="${HOST_PORT:-18120}"
URL="http://127.0.0.1:${HOST_PORT}/"
UA="nginxlings-test-agent/1.0"

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

echo "[3/6] Testing nginx config and log_format main inside the image (nginx -T)..."
temp_file DUMP
if ! "${DOCKER_BIN}" run --label "${DOCKER_LABEL}" --rm "${IMAGE_NAME}" nginx -T >"${DUMP}" 2>&1; then
  fail_nginx_rejected "${DUMP}"
fi
# Effective configuration without comment lines.
temp_file ACTIVE
grep -v '^[[:space:]]*#' "${DUMP}" >"${ACTIVE}" || true

# The official image already logs the built-in "combined" format to stdout,
# so the log lines alone do not prove that the learner's format is used.
temp_file FORMAT
awk '/^[[:space:]]*log_format[[:space:]]+main([[:space:]]|$)/ { on = 1 }
  on { print; if (/;[[:space:]]*$/) exit }' "${ACTIVE}" >"${FORMAT}"
[[ -s "${FORMAT}" ]] || fail "No active 'log_format main ...;' found in the http block of nginx.conf."
for var in remote_addr status request http_user_agent; do
  grep -qE "\\\$\\{?${var}([^a-z_]|$)" "${FORMAT}" || fail "log_format main does not include \$${var}."
done
if ! grep -qE '^[[:space:]]*access_log[[:space:]]+[^;[:space:]]+[[:space:]]+main([[:space:]][^;]*)?;' "${ACTIVE}"; then
  fail "No active 'access_log ... main;' directive found. Use: access_log /dev/stdout main;"
fi

echo "[4/6] Starting container ${CONTAINER_NAME} (host ${HOST_PORT} -> container 8080)..."
"${DOCKER_BIN}" rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
on_exit "\"${DOCKER_BIN}\" rm -f \"${CONTAINER_NAME}\""
"${DOCKER_BIN}" run --label "${DOCKER_LABEL}" -d \
  --name "${CONTAINER_NAME}" \
  -p "127.0.0.1:${HOST_PORT}:8080" \
  "${IMAGE_NAME}" >/dev/null
wait_container_http "${CONTAINER_NAME}" "${URL}" 10 -H "User-Agent: ${UA}"

echo "[5/6] Sending HTTP requests..."
for _ in 1 2 3 4 5; do
  if ! curl -sS -o /dev/null -m 5 -H "User-Agent: ${UA}" "${URL}"; then
    echo "docker logs (last 80 lines):"
    "${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>&1 | tail -n 80 || true
    fail "HTTP request to ${URL} failed; see the curl error above."
  fi
done

echo "[6/6] Validating access log output (docker logs)..."
# Log lines reach docker logs asynchronously; poll instead of sleeping.
MATCHED=""
for ((i = 0; i < 50; i++)); do
  LOGS="$("${DOCKER_BIN}" logs "${CONTAINER_NAME}" 2>/dev/null || true)"
  MATCHED="$(printf '%s\n' "${LOGS}" | grep -F "${UA}" || true)"
  [[ -n "${MATCHED}" ]] && break
  sleep 0.1
done

if [[ -z "${MATCHED}" ]]; then
  echo "Sample logs (last 120 lines):"
  printf '%s\n' "${LOGS}" | tail -n 120
  fail "No access log line contains the test User-Agent. Make sure access_log /dev/stdout uses a log_format that includes \$http_user_agent."
fi

if ! grep -q "GET /" <<<"${MATCHED}"; then
  echo "Matched UA lines:"
  printf '%s\n' "${MATCHED}" | head -n 30
  fail "UA-matched lines do not contain 'GET /'. Make sure \$request is included."
fi

if ! grep -Eq '(^|[[:space:]])200([[:space:]]|$)' <<<"${MATCHED}"; then
  echo "Matched UA lines:"
  printf '%s\n' "${MATCHED}" | head -n 30
  fail "UA-matched lines do not contain HTTP status 200. Make sure \$status is included."
fi

if ! grep -Eq '([0-9]{1,3}\.){3}[0-9]{1,3}' <<<"${MATCHED}"; then
  echo "Matched UA lines:"
  printf '%s\n' "${MATCHED}" | head -n 30
  fail "UA-matched lines do not contain an IP address. Make sure \$remote_addr is included."
fi

pass "12 – Logs & Variables"
