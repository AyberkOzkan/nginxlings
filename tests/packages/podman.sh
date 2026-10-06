#!/bin/sh
# Runs inside a privileged Rocky Linux container; see packages_test.go.
# Installs the .rpm, which brings Podman's docker command on RHEL-like
# systems, then runs every Docker exercise from its starting files and with
# its solution from /solutions, printing "OK <id>" or "BAD <id>" per exercise.
set -u
dnf install -y -q /dist/nginxlings-*."${RPM_ARCH}".rpm >/dev/null 2>&1 || { echo "BAD install"; exit 1; }
touch /etc/containers/nodocker
# Podman inside a container cannot use the pids cgroup controller.
printf '[containers]\ncgroups="disabled"\npids_limit=0\n' >/etc/containers/containers.conf
export HOME=/root
nginxlings --list >/dev/null
nginxlings --check 01_static_server >/dev/null 2>&1 # creates ~/nginxlings
for dir in /usr/share/nginxlings/exercises/*/; do
  id=$(basename "${dir}")
  grep -q '^[[:space:]]*require_docker' "${dir}check.sh" 2>/dev/null || continue
  start=$(nginxlings --check "${id}" 2>&1)
  started=$?
  cp -r "/solutions/${id}/." "${HOME}/nginxlings/${id}/"
  solved=$(nginxlings --check "${id}" 2>&1)
  passed=$?
  if [ "${started}" -ne 0 ] && [ "${passed}" -eq 0 ] && ! echo "${start}" | grep -q 'missing prerequisite\|Cannot reach'; then
    echo "OK ${id}"
  else
    echo "BAD ${id}: start: $(echo "${start}" | grep FAIL | tail -n 1) solved: $(echo "${solved}" | tail -n 2 | tr '\n' ' ')"
  fi
done
