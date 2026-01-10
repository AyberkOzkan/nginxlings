#!/usr/bin/env sh
set -eu

# Simple startup script for the HTTP load balancer lab.
# - Starts two backend HTTP servers on 127.0.0.1:8081 and 127.0.0.1:8082
# - Then starts nginx in the foreground (daemon off)

# Start backend 1
python3 -m http.server 8081 --bind 127.0.0.1 >/dev/null 2>&1 &
BACKEND1_PID="$!"

# Start backend 2
python3 -m http.server 8082 --bind 127.0.0.1 >/dev/null 2>&1 &
BACKEND2_PID="$!"

# If nginx exits, also stop backends
cleanup() {
    if [ -n "${BACKEND1_PID}" ]; then
        kill "${BACKEND1_PID}" 2>/dev/null || true
    fi
    if [ -n "${BACKEND2_PID}" ]; then
        kill "${BACKEND2_PID}" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Start nginx in the foreground
exec nginx -g 'daemon off;'

