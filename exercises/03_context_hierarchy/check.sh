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

echo "[1/3] Testing nginx.conf with nginx -t..."

if [ ! -f "${PREFIX}/nginx.conf" ]; then
  fail "nginx.conf not found in: ${PREFIX}"
fi

TMP_T_LOG="$(mktemp)"
if ! "${NGINX_BIN}" -t -c "${PREFIX}/nginx.conf" -p "${PREFIX}" >"${TMP_T_LOG}" 2>&1; then
  echo "nginx -t output:"
  cat "${TMP_T_LOG}"
  rm -f "${TMP_T_LOG}"
  fail "nginx -t failed. Fix the syntax and context errors above."
fi
rm -f "${TMP_T_LOG}"
echo "[OK] nginx -t succeeded."

echo "[2/3] Checking for http → server → location hierarchy..."

AWK_OUT="$(
  awk '
    /^[ \t]*#/ { next }

    BEGIN {
      sp = 0
      has_http = 0
      has_server_under_http = 0
      has_location_under_server = 0
    }

    {
      line = $0

      if (match(line, /^[ \t]*http[ \t]*\{/)) {
        stack[++sp] = "http"
        has_http = 1
      } else if (match(line, /^[ \t]*server[ \t]*\{/)) {
        stack[++sp] = "server"
        if (sp >= 2 && stack[sp-1] == "http") {
          has_server_under_http = 1
        }
      } else if (match(line, /^[ \t]*location[ \t]+/)) {
        if (index(line, "{") > 0) {
          stack[++sp] = "location"
          if (sp >= 3 && stack[sp-1] == "server" && stack[sp-2] == "http") {
            has_location_under_server = 1
          }
        }
      } else if (match(line, /^[ \t]*\}/)) {
        if (sp > 0) {
          sp--
        }
      }
    }

    END {
      if (!has_http) {
        print "NO_HTTP"
        exit 1
      }
      if (!has_server_under_http) {
        print "NO_SERVER_UNDER_HTTP"
        exit 1
      }
      if (!has_location_under_server) {
        print "NO_LOCATION_UNDER_SERVER"
        exit 1
      }
    }
  ' "${PREFIX}/nginx.conf" 2>/dev/null || true
)"

if [ -n "${AWK_OUT}" ]; then
  case "${AWK_OUT}" in
    NO_HTTP)
      fail "No http { ... } block found at the top level of nginx.conf."
      ;;
    NO_SERVER_UNDER_HTTP)
      fail "No server { ... } block detected inside the http { ... } context."
      ;;
    NO_LOCATION_UNDER_SERVER)
      fail "No location block detected inside a server { ... } context."
      ;;
    *)
      fail "Context hierarchy check failed: ${AWK_OUT}"
      ;;
  esac
fi

echo "[OK] http/server/location hierarchy looks consistent."

echo "[3/3] No additional runtime check is required for this exercise."
echo
echo "[PASS] 03 – Context Hierarchy exercise looks good."

