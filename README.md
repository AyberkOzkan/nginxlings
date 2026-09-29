# Nginxlings

Nginxlings is a terminal training app for learning NGINX through small, runnable exercises. It uses a Bubble Tea TUI to show exercise instructions, run each exercise's `check.sh`, and track local progress.

The project currently includes 18 exercises under `exercises/`.

## Requirements

- Go 1.25 or newer (only to build or run from source)
- A POSIX-like shell environment
- `bash`
- `curl`
- `nginx` for local NGINX exercises
- Docker for container-based exercises
- `python3` for exercises that start local helper servers

Some exercises install or run extra tools inside Docker images, such as `socat`, `jq`, or `openssl`.

## Supported Platforms

The supported runtime and release target is Linux on AMD64 and ARM64.

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
- `tab`: toggle between instructions and the last check output
- `h`: show the selected exercise's hint (press again to return)
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

The app looks for `exercises/` and `i18n/` from the current directory, parent directories, or `NGINXLINGS_ASSET_ROOT`. This keeps repository-root development working while allowing packaged layouts to provide assets next to the app.

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

Tagged releases provide Linux archives for AMD64 and ARM64 through GitHub Releases. Extract the complete archive and keep `exercises/` and `i18n/` beside the `nginxlings` binary:

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
docs/             Maintainer and release documentation
```

## Development Checks

Before committing changes, run:

```sh
gofmt -w .
go test ./...
go vet ./...
for script in exercises/*/check.sh exercises/lib/*.sh; do bash -n "$script"; done
```
