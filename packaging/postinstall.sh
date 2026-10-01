#!/bin/sh
# Runs after the nginxlings package is installed or upgraded. Most exercises
# run NGINX in Docker, so start the Docker service and keep it on across
# restarts. Debian and Ubuntu start it themselves; Fedora, openSUSE, Arch and
# Alpine do not. Without a running init system (containers, chroots) this
# does nothing. It never changes group membership: nginxlings asks the
# learner before adding them to the docker group.

if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
  if systemctl list-unit-files docker.service >/dev/null 2>&1; then
    systemctl enable --now docker.service >/dev/null 2>&1 || true
  fi
elif command -v rc-service >/dev/null 2>&1 && [ -d /run/openrc ]; then
  rc-update add docker default >/dev/null 2>&1 || true
  rc-service docker start >/dev/null 2>&1 || true
fi

echo "nginxlings is installed. Open it from the application menu or type: nginxlings"
echo "If Docker still needs a step, nginxlings shows it and can run it for you (press d)."
exit 0
