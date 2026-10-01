# Nginxlings

Nginxlings is a terminal training app for learning NGINX through small, runnable exercises. It uses a Bubble Tea TUI to show exercise instructions, run each exercise's `check.sh`, and track local progress.

The project currently includes 18 exercises under `exercises/`.

A Turkish installation guide for learners is in `docs/INSTALL.tr.md`.

## Install

Download the file for your distribution from the latest GitHub Release. Most
desktops install it on a double-click; the terminal command is given too.

| Distribution | File | Command |
| --- | --- | --- |
| Ubuntu, Debian, Linux Mint, Pop!_OS | `nginxlings_<version>_amd64.deb` | `sudo apt install ./nginxlings_<version>_amd64.deb` |
| Fedora, Rocky, AlmaLinux, RHEL | `nginxlings-<version>-1.x86_64.rpm` | `sudo dnf install ./nginxlings-<version>-1.x86_64.rpm` |
| openSUSE | `nginxlings-<version>-1.x86_64.rpm` | `sudo zypper install --allow-unsigned-rpm ./nginxlings-<version>-1.x86_64.rpm` |
| Arch, Manjaro | `nginxlings-<version>-1-x86_64.pkg.tar.zst` | `sudo pacman -U ./nginxlings-<version>-1-x86_64.pkg.tar.zst` |
| Any other (Alpine, Void, ...) | `nginxlings_<version>_linux_amd64.tar.gz` | extract it, then `./install.sh` |

ARM64 machines use the `arm64` / `aarch64` files. The package manager also
installs what the exercises run: bash, curl, NGINX, Python 3 and Docker, and
the package starts the Docker service. `install.sh` does the same with the
distribution's package manager and installs nginxlings in `/usr/local`.

Then open Nginxlings from the application menu, or type `nginxlings` in a
terminal. On first start it copies the exercises to `~/nginxlings`, where you
edit them; `o` opens the selected exercise's folder. If something is still
missing, usually Docker access for your user, the status row shows
`! Setup` and `d` explains the fix; `f` runs it.

On Rocky, AlmaLinux and RHEL the package installs Podman's `docker` command
instead of Docker. Every exercise runs with it except 10, which teaches
Docker's embedded DNS server on `127.0.0.11`; install Docker CE from Docker's
repository for that one. The setup check (`d`, `--doctor`) notes this, and
exercise 10's check stops with the same explanation instead of failing later.

To remove nginxlings, remove the package (`sudo apt remove nginxlings`,
`sudo dnf remove nginxlings`, `sudo pacman -R nginxlings`) or run
`./install.sh --uninstall`. Your work in `~/nginxlings` is kept.

## Requirements

To build or run from source:

- Go 1.26 or newer (only to build or run from source)
- A POSIX-like shell environment
- `bash`
- `curl`
- `nginx` for local NGINX exercises
- Docker for container-based exercises
- `python3` for exercises that start local helper servers

Some exercises install or run extra tools inside Docker images, such as `socat`, `jq`, or `openssl`.

## Supported Platforms

The supported runtime and release target is Linux on AMD64 and ARM64: packages
for the Debian, Fedora/RHEL/openSUSE and Arch families, and `install.sh` for
other distributions.

Native macOS and Windows builds are not produced. Windows users may run a Linux release through WSL2 with Docker integration.

## Quick Start

Run the app from the repository root:

```sh
go run ./cmd/nginxlings
```

Inside the TUI:

- `w` / `s`: select an exercise
- `up` / `down`: scroll instructions
- `pgup` / `pgdown`: scroll faster
- `left` / `right`: horizontal scroll
- `c` or `enter`: run the selected exercise check
- `tab`: toggle between instructions and the last check output, which opens at
  its end where the result is
- `h`: show the selected exercise's hint (press again to return)
- `d`: setup check: what the exercises still need and how to fix it
- `?`: list every key in the right panel
- `o`: open the selected exercise's folder in the file manager
- `e`: restore the selected exercise's starting files (installed package only; asks first)
- `l`: switch language
- `r`: reset progress
- `q`: quit

## CLI Usage

Show help or version information without starting the TUI:

```sh
go run ./cmd/nginxlings --help
go run ./cmd/nginxlings --version
```

List exercises in the same stable order used by the TUI:

```sh
go run ./cmd/nginxlings --list
```

Check that the tools the exercises need are installed and that Docker answers; each problem comes with the command that fixes it, and the exit code is 1 while something is missing:

```sh
go run ./cmd/nginxlings --doctor
go run ./cmd/nginxlings --doctor --lang tr
```

Run one exercise check without starting the TUI:

```sh
go run ./cmd/nginxlings --check 01_static_server
```

`--list` and `--check` only resolve application assets; they do not require or create a user state file.

Set the initial TUI language:

```sh
go run ./cmd/nginxlings --lang tr
go run ./cmd/nginxlings --lang en
```

The code must be listed in `i18n/langs.json`; an unknown code exits with an error that lists the available languages.

Override asset and state paths:

```sh
go run ./cmd/nginxlings --root /path/to/nginxlings
go run ./cmd/nginxlings --state /tmp/nginxlings-state.json
```

Flag values take precedence over the matching environment variables for that run.

## Important Runtime Notes

During development, run Nginxlings from the repository root:

```sh
go run ./cmd/nginxlings
```

The app looks for `exercises/` and `i18n/` in `NGINXLINGS_ASSET_ROOT`, the current directory and its parents, the binary's directory and its parents, and finally `<prefix>/share/nginxlings` next to `<prefix>/bin/nginxlings`, where packages and `install.sh` put them. This keeps repository-root development working while installed copies find their assets from anywhere.

When the assets come from `<prefix>/share/nginxlings`, the learner cannot edit them there, so the app copies the exercises to `~/nginxlings` and works on that copy. Every start updates it from the installed version: new exercises and files are added, check scripts, READMEs and the shared library are replaced, and other files are replaced only while the learner has not changed them. `--workspace <path>` or `NGINXLINGS_WORKSPACE` picks another folder and also turns the copy on for the repository or an extracted archive, which otherwise edit the exercises in place.

Progress is stored in:

```text
${XDG_STATE_HOME}/nginxlings/state.json
```

If `XDG_STATE_HOME` is not set, Linux builds use:

```text
~/.local/state/nginxlings/state.json
```

You can override the state location for development or tests:

```sh
NGINXLINGS_STATE_ROOT=/tmp/nginxlings-state go run ./cmd/nginxlings
NGINXLINGS_STATE_FILE=/tmp/nginxlings-state.json go run ./cmd/nginxlings
go run ./cmd/nginxlings --state /tmp/nginxlings-state.json
```

The state file stores completed exercises, the last selected exercise, and the selected language. Older repository-local `.nginxlings_state.json` files are no longer written by the app.

## Running Checks

Each exercise has its own `check.sh`. The TUI runs the selected exercise's script from that exercise directory.

Checks started by Nginxlings have a 10-minute timeout. Quitting the TUI while a
check is running cancels the script and its child processes before the app exits.

You can also run a check manually:

```sh
cd exercises/01_static_server
./check.sh
```

Container-based exercises use exercise-scoped labels, build temporary images named `nginxlings/<exercise-id>:check`, and remove only the containers, networks, and images created for that check. Checks do not prune the global Docker builder cache.

## Release Archives

Tagged releases provide Linux packages (see Install) and archives for AMD64 and ARM64 through GitHub Releases. To run from an archive without installing it, extract the complete archive and keep `exercises/` and `i18n/` beside the `nginxlings` binary:

```sh
tar -xzf nginxlings_<version>_<os>_<arch>.tar.gz
./nginxlings --version
```

Each release includes `checksums.txt` for SHA-256 verification and GitHub artifact attestations. npm and Homebrew packages are not published; this keeps installation independent from npm registry packages and lifecycle scripts.

## Troubleshooting

If the TUI cannot find exercises or translations, make sure you started it from the repository root or set `NGINXLINGS_ASSET_ROOT`:

```sh
pwd
go run ./cmd/nginxlings
NGINXLINGS_ASSET_ROOT=/path/to/nginxlings go run ./cmd/nginxlings
```

If a check fails because a command is missing, install the required command shown in the failure message. Common missing commands are `nginx`, `docker`, `curl`, `bash`, and `python3`.

If a Docker exercise fails to start, check that Docker is running and that the exercise's default host port is not already in use. Several checks allow port overrides through environment variables documented in the script, for example:

```sh
HOST_PORT=18199 ./check.sh
```

If a local NGINX exercise fails on port `8080` or `8081`, stop the process already using that port before rerunning the check.

## Contributing

See `CONTRIBUTING.md` for the development workflow, translation rules, and the checklist for adding exercises. Contributions are accepted under `AGPL-3.0-only`.

## License

Copyright (c) 2026 Kenan Ayberk Özkan. Nginxlings is licensed under `AGPL-3.0-only`. You may use, study, modify, and redistribute it under that license. Modified versions, including versions offered over a network, must make their complete corresponding source available under the same license. This prevents incorporating Nginxlings into a closed-source product without complying with the AGPL terms. See `LICENSE` for the complete terms.

## Project Layout

```text
cmd/nginxlings/   Bubble Tea TUI application
exercises/        NGINX exercise directories
exercises/lib/    Shared helpers sourced by every check.sh
i18n/             UI texts, exercise titles, and hints per language
packaging/        Menu entry, icon and post-install script of the Linux packages
install.sh        Installer for distributions without a package
docs/             Installation, maintainer and release documentation
tests/            Black-box test suites (see tests/TESTING.md)
```

## Development Checks

Before committing changes, run:

```sh
gofmt -w .
go test ./...
go vet ./...
for script in exercises/*/check.sh exercises/lib/*.sh; do bash -n "$script"; done
```
