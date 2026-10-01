#!/bin/sh
# Runs inside a distribution container after nginxlings is installed; see
# packages_test.go. EXPECTED_EXERCISES is the number of shipped exercises.
set -u
fail() { echo "VERIFY FAIL: $*"; exit 1; }
cd /
v=$(nginxlings --version) || fail "--version"
echo "$v"
n=$(nginxlings --list | wc -l)
[ "$n" -eq "${EXPECTED_EXERCISES}" ] || fail "--list printed $n exercises, want ${EXPECTED_EXERCISES}"
[ -f /usr/share/applications/nginxlings.desktop ] || [ -f /usr/local/share/applications/nginxlings.desktop ] || fail "no desktop entry"
[ -f /usr/share/icons/hicolor/scalable/apps/nginxlings.svg ] || [ -f /usr/local/share/icons/hicolor/scalable/apps/nginxlings.svg ] || fail "no icon"
export HOME=/tmp/learner
mkdir -p "$HOME"
out=$(nginxlings --check 01_static_server 2>&1) && fail "01 passed from its starting files"
echo "$out" | tail -2
echo "$out" | grep -q 'missing prerequisite' && fail "01 lacks a prerequisite"
[ -x "$HOME/nginxlings/01_static_server/check.sh" ] || fail "workspace not created"
out=$(nginxlings --check 06_reverse_proxy 2>&1) && fail "06 passed from its starting files"
echo "$out" | grep -q 'missing prerequisite' && fail "06 lacks a prerequisite: $out"
doc=$(nginxlings --doctor 2>&1)
echo "$doc" | head -3
echo "$doc" | grep -q 'is not installed' && fail "doctor reports a missing command"
# A regular user logs in with the distribution's default PATH, which on
# Debian lacks /usr/sbin where nginx lives.
if command -v useradd >/dev/null 2>&1; then useradd -m learner; else adduser -D learner; fi
# $PATH must expand in the learner's login shell, not here.
# shellcheck disable=SC2016
user_out=$(su - learner -c 'echo "PATH=$PATH"; nginxlings --doctor; nginxlings --check 01_static_server' 2>&1)
echo "$user_out" | head -2
echo "$user_out" | grep -q 'is not installed' && fail "doctor misses a command for a regular user"
echo "$user_out" | grep -q 'missing prerequisite' && fail "01 lacks a prerequisite for a regular user"
[ -x /home/learner/nginxlings/01_static_server/check.sh ] || fail "no workspace for the regular user"
command -v docker >/dev/null || fail "no docker command"
command -v dockerd >/dev/null || command -v podman >/dev/null || fail "neither the Docker daemon nor Podman is installed"
echo "VERIFY OK"
