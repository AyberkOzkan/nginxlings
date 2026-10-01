#!/bin/sh
# Installs nginxlings on any Linux distribution, including those without a
# nginxlings package (.deb, .rpm, Arch).
#
#   From an extracted release archive:   ./install.sh
#   Without one, from the latest release: curl -fsSL https://raw.githubusercontent.com/AyberkOzkan/nginxlings/main/install.sh | sh
#   To remove nginxlings:                 ./install.sh --uninstall
#
# It installs the program in /usr/local/bin and the exercises in
# /usr/local/share/nginxlings, adds a menu entry, installs what the exercises
# run (bash, curl, NGINX, Python, Docker) and tmux for the terminal pane with
# the distribution's package manager, and starts Docker. Your own copy of the exercises in ~/nginxlings
# is never touched.

set -eu

REPO="AyberkOzkan/nginxlings"
PREFIX="/usr/local"
SHARE="${PREFIX}/share/nginxlings"

say() { printf '%s\n' "$*"; }
die() { printf 'nginxlings install: %s\n' "$*" >&2; exit 1; }

# as_root runs a command as root, through sudo when needed.
as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  elif command -v doas >/dev/null 2>&1; then
    doas "$@"
  else
    die "run this as root, or install sudo"
  fi
}

uninstall() {
  as_root rm -rf "${SHARE}"
  as_root rm -f "${PREFIX}/bin/nginxlings" \
    "${PREFIX}/share/applications/nginxlings.desktop" \
    "${PREFIX}/share/icons/hicolor/scalable/apps/nginxlings.svg"
  say "nginxlings is removed. Your exercises in ~/nginxlings are kept; delete that folder yourself if you no longer need it."
}

# os_value KEY prints a value from /etc/os-release.
os_value() {
  [ -r /etc/os-release ] || return 0
  sed -n "s/^$1=//p" /etc/os-release | tr -d '"' | head -n 1
}

# install_packages installs the packages for the commands the exercises
# need that are missing.
install_packages() {
  ids="$(os_value ID) $(os_value ID_LIKE)"
  missing=""
  for cmd in bash curl nginx python3 docker tmux; do
    command -v "${cmd}" >/dev/null 2>&1 && continue
    # nginx usually lives in /usr/sbin, which a user's PATH may lack.
    [ "${cmd}" = nginx ] && [ -x /usr/sbin/nginx ] && continue
    missing="${missing} ${cmd}"
  done
  [ -n "${missing}" ] || return 0

  manager=""
  case " ${ids} " in
    *" debian "* | *" ubuntu "*) manager=apt ;;
    *" fedora "* | *" rhel "* | *" centos "*) manager=dnf ;;
    *" suse "* | *" opensuse "*) manager=zypper ;;
    *" arch "*) manager=pacman ;;
    *" alpine "*) manager=apk ;;
    *" void "*) manager=xbps ;;
  esac

  packages=""
  for cmd in ${missing}; do
    pkg="${cmd}"
    case "${manager}:${cmd}" in
      apt:docker) pkg=docker.io ;;
      dnf:docker)
        # Fedora packages the Docker daemon; RHEL-like systems ship Podman.
        if [ "$(os_value ID)" = fedora ]; then pkg=moby-engine; else pkg=podman-docker; fi
        ;;
      pacman:python3) pkg=python ;;
    esac
    packages="${packages} ${pkg}"
  done

  say "Installing what the exercises need:${packages}"
  # The package lists are meant to split into words.
  # shellcheck disable=SC2086
  case "${manager}" in
    apt) as_root apt-get update -q && as_root env DEBIAN_FRONTEND=noninteractive apt-get install -y -q ${packages} ;;
    dnf) as_root dnf install -y ${packages} ;;
    zypper) as_root zypper --non-interactive install ${packages} ;;
    pacman) as_root pacman -S --needed --noconfirm ${packages} ;;
    apk) as_root apk add ${packages} ;;
    xbps) as_root xbps-install -Sy ${packages} ;;
    *)
      say "This distribution is not known to the installer. Install these with its package manager:${missing}"
      say "Then run: nginxlings --doctor"
      ;;
  esac
}

# start_docker starts Docker now and at every boot.
start_docker() {
  if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    as_root systemctl enable --now docker >/dev/null 2>&1 || true
  elif command -v rc-service >/dev/null 2>&1 && [ -d /run/openrc ]; then
    as_root rc-update add docker default >/dev/null 2>&1 || true
    as_root rc-service docker start >/dev/null 2>&1 || true
  elif [ -d /var/service ] && [ -d /etc/sv/docker ]; then
    as_root ln -sf /etc/sv/docker /var/service/ || true
  fi
}

# download_release fetches the latest release archive for this machine,
# checks it against checksums.txt and extracts it into $1.
download_release() {
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) die "no release for $(uname -m); nginxlings is built for amd64 and arm64" ;;
  esac
  base="https://github.com/${REPO}/releases/latest/download"
  if command -v curl >/dev/null 2>&1; then
    fetch() { curl -fsSL "$1" -o "$2"; }
  elif command -v wget >/dev/null 2>&1; then
    fetch() { wget -q "$1" -O "$2"; }
  else
    die "install curl or wget first"
  fi
  fetch "${base}/checksums.txt" "$1/checksums.txt" || die "cannot download the release from GitHub (${base})"
  name="$(sed -n "s/^[0-9a-f]*  \(nginxlings_[^ ]*_linux_${arch}\.tar\.gz\)$/\1/p" "$1/checksums.txt" | head -n 1)"
  [ -n "${name}" ] || die "the latest release has no archive for linux ${arch}"
  say "Downloading ${name}..."
  fetch "${base}/${name}" "$1/${name}" || die "cannot download ${name}"
  (cd "$1" && grep "  ${name}\$" checksums.txt | sha256sum -c - >/dev/null) || die "${name} does not match checksums.txt"
  mkdir -p "$1/release"
  tar -xzf "$1/${name}" -C "$1/release"
}

main() {
  [ "$(uname -s)" = Linux ] || die "nginxlings runs on Linux"
  if [ "${1:-}" = "--uninstall" ]; then
    uninstall
    return
  fi

  # Use the archive this script came with, or download the latest one.
  src=""
  case "$0" in
    */*) dir="$(cd "$(dirname "$0")" && pwd)" ;;
    *) dir="" ;;
  esac
  if [ -n "${dir}" ] && [ -x "${dir}/nginxlings" ] && [ -d "${dir}/exercises" ] && [ -d "${dir}/i18n" ]; then
    src="${dir}"
  else
    tmp="$(mktemp -d)"
    trap 'rm -rf "${tmp}"' EXIT
    download_release "${tmp}"
    src="${tmp}/release"
  fi

  install_packages
  say "Installing nginxlings in ${PREFIX}..."
  as_root rm -rf "${SHARE}"
  as_root mkdir -p "${PREFIX}/bin" "${SHARE}" "${PREFIX}/share/applications" "${PREFIX}/share/icons/hicolor/scalable/apps"
  as_root cp -R "${src}/exercises" "${src}/i18n" "${SHARE}/"
  as_root cp "${src}/nginxlings" "${PREFIX}/bin/nginxlings"
  as_root chmod 0755 "${PREFIX}/bin/nginxlings"
  if [ -f "${src}/packaging/nginxlings.desktop" ]; then
    as_root cp "${src}/packaging/nginxlings.desktop" "${PREFIX}/share/applications/"
    as_root cp "${src}/packaging/nginxlings.svg" "${PREFIX}/share/icons/hicolor/scalable/apps/"
  fi
  start_docker

  say ""
  say "nginxlings is installed. Open it from the application menu, or open a terminal and type: nginxlings"
  say "Your exercises will be in ~/nginxlings. If something is still missing, nginxlings shows it and how to fix it (press d)."
}

main "$@"
