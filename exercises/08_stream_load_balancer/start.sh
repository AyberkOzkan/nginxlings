#!/usr/bin/env sh
set -eu

# Two simple TCP backends:
# - 127.0.0.1:9001 returns "backend=9001"
# - 127.0.0.1:9002 returns "backend=9002"

backend1() {
    socat TCP-LISTEN:9001,bind=127.0.0.1,reuseaddr,fork SYSTEM:"printf 'backend=9001\n'"
}

backend2() {
    socat TCP-LISTEN:9002,bind=127.0.0.1,reuseaddr,fork SYSTEM:"printf 'backend=9002\n'"
}

backend1 >/dev/null 2>&1 &
B1_PID="$!"

backend2 >/dev/null 2>&1 &
B2_PID="$!"

cleanup() {
    kill "$B1_PID" 2>/dev/null || true
    kill "$B2_PID" 2>/dev/null || true
}
trap cleanup EXIT

# Start nginx in the foreground
exec nginx -g 'daemon off;'
