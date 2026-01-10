#!/bin/sh
set -eu

# Bu egzersiz dizini
EX_DIR="$(cd "$(dirname "$0")" && pwd)"
CONF="$EX_DIR/nginx.conf"
PREFIX="$EX_DIR"
PID_FILE="$PREFIX/logs/nginx.pid"
BODY_FILE="$PREFIX/.check_body"

mkdir -p "$PREFIX/logs"

# Eski NGINX süreci varsa durdur
if [ -f "$PID_FILE" ]; then
    if kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        nginx -c "$CONF" -p "$PREFIX" -s stop >/dev/null 2>&1 || true
    fi
fi

echo "[1/3] Testing nginx.conf..."
if ! nginx -t -c "$CONF" -p "$PREFIX" >/dev/null 2>&1; then
    echo "[FAIL] nginx -t failed for 01_static_server"
    exit 1
fi

echo "[2/3] Starting nginx on 127.0.0.1:8080..."
nginx -c "$CONF" -p "$PREFIX" >/dev/null 2>&1

# Kısa bir bekleme
sleep 0.4

echo "[3/3] Checking HTTP response and body..."
STATUS=0

HTTP_CODE="$(curl -s -o "$BODY_FILE" -w '%{http_code}' http://127.0.0.1:8080/ || true)"

if [ "$HTTP_CODE" != "200" ]; then
    echo "[FAIL] expected HTTP 200 from /, got $HTTP_CODE"
    STATUS=1
else
    if ! grep -q 'Merhaba Nginxlings' "$BODY_FILE"; then
        echo "[FAIL] body does not contain 'Merhaba Nginxlings'"
        STATUS=1
    else
        echo "[OK] 01_static_server: HTTP 200 and body contains 'Merhaba Nginxlings'"
    fi
fi

# Temizlik
rm -f "$BODY_FILE"
nginx -c "$CONF" -p "$PREFIX" -s stop >/dev/null 2>&1 || true

exit "$STATUS"

