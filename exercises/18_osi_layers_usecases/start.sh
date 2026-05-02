#!/usr/bin/env sh
set -eu

mkdir -p /tmp/http1 /tmp/http2

printf 'l7-backend=8081\n' > /tmp/http1/index.html
printf 'l7-backend=8082\n' > /tmp/http2/index.html

# HTTP backends
python3 -m http.server 8081 --bind 127.0.0.1 -d /tmp/http1 >/dev/null 2>&1 &
python3 -m http.server 8082 --bind 127.0.0.1 -d /tmp/http2 >/dev/null 2>&1 &

# TCP backends
socat TCP-LISTEN:9001,reuseaddr,fork SYSTEM:'printf "l4-backend=9001\n"' >/dev/null 2>&1 &
socat TCP-LISTEN:9002,reuseaddr,fork SYSTEM:'printf "l4-backend=9002\n"' >/dev/null 2>&1 &

exec nginx -g 'daemon off;'