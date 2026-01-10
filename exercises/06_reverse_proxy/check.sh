#!/usr/bin/env bash
set -euo pipefail

NGINX_BIN="${NGINX_BIN:-nginx}"
PREFIX="$(pwd)"
LOG_DIR="${PREFIX}/logs"

mkdir -p "${LOG_DIR}"

fail() {
  echo "[FAIL] $*"
  exit 1
}

NGINX_PID=""
BACKEND_PID=""

cleanup() {
  local code=$?
  if [[ -n "${NGINX_PID}" ]]; then
    # Try graceful stop; ignore errors
    "${NGINX_BIN}" -s stop -c "${PREFIX}/nginx.conf" -p "${PREFIX}" >/dev/null 2>&1 || true
    sleep 0.3 || true
  fi
  if [[ -n "${BACKEND_PID}" ]]; then
    kill "${BACKEND_PID}" >/dev/null 2>&1 || true
  fi
  exit "${code}"
}
trap cleanup EXIT

echo "[1/4] Checking required files..."

if [ ! -f "${PREFIX}/nginx.conf" ]; then
  fail "nginx.conf not found in ${PREFIX}"
fi

if [ ! -f "${PREFIX}/backend_app.py" ]; then
  fail "backend_app.py not found in ${PREFIX}"
fi

if ! command -v python3 >/dev/null 2>&1; then
  fail "python3 is required to run backend_app.py"
fi

echo "[OK] nginx.conf and backend_app.py are present."

echo "[2/4] Starting backend on 127.0.0.1:8080..."

python3 "${PREFIX}/backend_app.py" >/dev/null 2>&1 &
BACKEND_PID=$!

# Wait for backend to become ready
backend_ready=0
for i in $(seq 1 20); do
  if curl -s -o /dev/null "http://127.0.0.1:8080/"; then
    backend_ready=1
    break
  fi
  sleep 0.2
done

if [ "${backend_ready}" -ne 1 ]; then
  fail "Backend on 127.0.0.1:8080 did not respond. Is the port already in use?"
fi

echo "[OK] Backend is responding."

echo "[3/4] Testing nginx.conf with nginx -t..."

TMP_T_LOG="$(mktemp)"
if ! "${NGINX_BIN}" -t -c "${PREFIX}/nginx.conf" -p "${PREFIX}" >"${TMP_T_LOG}" 2>&1; then
  echo "nginx -t output:"
  cat "${TMP_T_LOG}"
  rm -f "${TMP_T_LOG}"
  fail "nginx -t failed."
fi
rm -f "${TMP_T_LOG}"
echo "[OK] nginx -t passed."

echo "[4/4] Starting nginx reverse proxy on 127.0.0.1:8081..."

"${NGINX_BIN}" -c "${PREFIX}/nginx.conf" -p "${PREFIX}" >/dev/null 2>&1 &
NGINX_PID=$!

# Wait for nginx to respond
proxy_ready=0
for i in $(seq 1 20); do
  if curl -s -o /dev/null "http://127.0.0.1:8081/"; then
    proxy_ready=1
    break
  fi
  sleep 0.2
done

if [ "${proxy_ready}" -ne 1 ]; then
  fail "Nginx on 127.0.0.1:8081 did not respond."
fi

echo "[OK] Nginx is responding on 127.0.0.1:8081."

echo "[4/4] Verifying that the response comes from the backend..."

RESP="$(curl -s -D - "http://127.0.0.1:8081/")"

# Check for our special marker from backend_app.py
if ! printf '%s\n' "${RESP}" | grep -q "backend-06-ok"; then
  echo "Full response via proxy:"
  printf '%s\n' "${RESP}"
  fail "Response via Nginx does not contain 'backend-06-ok'. Is proxy_pass configured correctly?"
fi

echo "[PASS] 06 – HTTP Reverse Proxy Basics looks good."

