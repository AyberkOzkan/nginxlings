# Contributing to Nginxlings

Contributions to the official Nginxlings project are welcome. Open an issue before substantial changes so the scope and exercise learning goal can be agreed with the maintainers.

By submitting a contribution, you agree that it is licensed under `AGPL-3.0-only`, as described in `LICENSE`. Only submit work that you created or have the right to contribute under these terms.

## Running From Source

You need:

- Go 1.26 or newer
- A POSIX-like shell environment
- `bash`
- `curl`
- `nginx` for local NGINX exercises
- Docker for container-based exercises
- `python3` for exercises that start local helper servers

Some exercises install or run extra tools inside Docker images, such as `socat`, `jq`, or `openssl`.

Run the app from the repository root:

```sh
go run ./cmd/nginxlings
```

It takes the same flags as an installed copy, for example
`go run ./cmd/nginxlings --doctor`; the README's Command Line section describes
the ones a learner uses, and `--help` lists all of them. `--list` only reads the
application assets, and `--check` does not create or change a state file.

### Where assets and progress live

The app looks for `exercises/` and `i18n/` in `NGINXLINGS_ASSET_ROOT`, the current directory and its parents, the binary's directory and its parents, and finally `<prefix>/share/nginxlings` next to `<prefix>/bin/nginxlings`, where packages and `install.sh` put them. This keeps repository-root development working while installed copies find their assets from anywhere.

If the TUI cannot find exercises or translations, make sure you started it from the repository root or set `NGINXLINGS_ASSET_ROOT`:

```sh
NGINXLINGS_ASSET_ROOT=/path/to/nginxlings go run ./cmd/nginxlings
```

When the assets come from `<prefix>/share/nginxlings`, the learner cannot edit them there, so the app copies the exercises to `~/nginxlings` and works on that copy. Every start updates it from the installed version: new exercises and files are added, check scripts, READMEs and the shared library are replaced, and other files are replaced only while the learner has not changed them. `--workspace <path>` or `NGINXLINGS_WORKSPACE` picks another folder and also turns the copy on for the repository or an extracted archive, which otherwise edit the exercises in place.

Progress is stored in:

```text
${XDG_STATE_HOME}/nginxlings/state.json
```

If `XDG_STATE_HOME` is not set, Linux builds use:

```text
~/.local/state/nginxlings/state.json
```

You can override the asset and state locations for development or tests:

```sh
go run ./cmd/nginxlings --root /path/to/nginxlings
NGINXLINGS_STATE_ROOT=/tmp/nginxlings-state go run ./cmd/nginxlings
NGINXLINGS_STATE_FILE=/tmp/nginxlings-state.json go run ./cmd/nginxlings
go run ./cmd/nginxlings --state /tmp/nginxlings-state.json
```

Flag values take precedence over the matching environment variables for that run.

The state file stores the result of each exercise's last check, the last selected exercise, and the selected language. Older repository-local `.nginxlings_state.json` files are no longer written by the app.

### Running checks

Each exercise has its own `check.sh`. The TUI runs the selected exercise's script from that exercise directory.

Checks started by Nginxlings have a 10-minute timeout. Quitting the TUI while a
check is running cancels the script and its child processes before the app exits.

You can also run a check manually:

```sh
cd exercises/01_static_server
./check.sh
```

Container-based exercises use exercise-scoped labels, build temporary images named `nginxlings/<exercise-id>:check`, and remove only the containers, networks, and images created for that check. Checks do not prune the global Docker builder cache.

## Project Layout

```text
cmd/nginxlings/   Bubble Tea TUI application
exercises/        NGINX exercise directories
exercises/lib/    Shared helpers sourced by every check.sh
i18n/             UI texts, exercise titles, and hints per language
packaging/        Menu entry, icon and post-install script of the Linux packages
install.sh        Installer for distributions without a package
docs/             Turkish installation guide, glossary, release notes for maintainers; the README demo
tests/            Black-box test suites (see tests/TESTING.md)
```

## Development Workflow

1. Fork the repository and create a focused branch.
2. Keep changes limited to one behavior or backlog item.
3. Add or update tests for behavior changes.
4. Run the validation commands below.
5. Open a pull request that describes the behavior, tests, and remaining risks.

Do not commit generated runtime files such as logs, PID files, `.check_body`, `response.html`, or release output under `dist/`.

## Adding an Exercise

Use this checklist when adding an exercise:

1. Create `exercises/NN_short_name/`, using the next two-digit number. Directory names determine stable discovery order.
2. Add non-empty `README.en.md` and `README.tr.md` files with equivalent heading structure and learning objectives.
3. Add an executable `check.sh` that sources `exercises/lib/check-lib.sh`, runs from its exercise directory, and returns zero only when the exercise passes. Use the library helpers for cleanup, temporary files, port checks, image builds, and readiness waits instead of fixed `sleep` calls.
4. Add the exercise to `exercises/manifest.json` with an ID matching the directory and a useful `default_name` fallback.
5. Add the same exercise ID to every `i18n/exercises/<language>.json` file with a non-empty translated title, and to every `i18n/hints/<language>.json` file with a one-sentence hint that points at the concept without giving away the complete solution.
6. Keep NGINX configuration syntax and comments in English. Keep English instructions in `README.en.md` and Turkish instructions in `README.tr.md`.
7. For Docker exercises, use the scoped resource labels and image naming conventions already used by existing checks. Do not prune global Docker resources.
8. Pin Docker base image patch versions consistently with existing exercises, and pin images that a check runs directly by digest (see Container Images).
9. If the exercise introduces a new asset file type, add its glob to `.goreleaser.yml` and inspect a snapshot archive.
10. Do not commit logs, generated certificates, PID files, check response bodies, or other runtime artifacts.
11. Make each README title equal the exercise's list name in that language (`NN - Name` becomes `# NN – Name`); a repository test enforces it.
12. Add the solution to your solutions directory, never to the repository, and run the exercise tests described in `tests/TESTING.md`. If the starting files could be mistaken for a solution, add an entry to `tests/repo/starting_state_test.go`.

Exercise checks should remain understandable on failure and must not require Docker, NGINX, curl, or Python merely to run the Go unit test suite.

## Translations

- `i18n/langs.json` defines supported languages.
- `i18n/ui/<language>.json` files must contain the same UI keys.
- `i18n/exercises/<language>.json` and `i18n/hints/<language>.json` files must contain exactly the manifest exercise IDs.
- Missing runtime translations still fall back safely, but repository validation requires complete checked-in translations.
- Keep translated README documents structurally aligned when adding or removing sections.
- Turkish text is fully Turkish: a technical term appears once with its English form in parentheses, then in Turkish only. Follow `docs/glossary.tr.md` for terms and titles.

## Validation

`tests/TESTING.md` describes every test layer, the release checks, and how to add tests. Run all checks before opening a pull request:

```sh
gofmt -w .
go test ./...
go vet ./...
for script in exercises/*/check.sh exercises/lib/*.sh; do bash -n "$script"; done
```

CI also runs ShellCheck and validates the GoReleaser configuration.

## GitHub Actions

Workflows pin every action to a full commit SHA followed by its version tag, for example `actions/checkout@<sha> # v4.4.0`, and CI runs with read-only repository permissions. To update an action, review its release notes, resolve the new tag to its commit with `git ls-remote https://github.com/<owner>/<action> 'refs/tags/<tag>^{}' 'refs/tags/<tag>'`, and update both the SHA and the comment. A repository test rejects unpinned actions.

## Container Images

Exercise `Dockerfile`s use patch-pinned base image tags such as `nginx:1.28.3-alpine`, which keeps them readable for learners. Images that a check runs directly, such as the BusyBox backends in exercise 10, are pinned by digest as `<name>:<tag>@sha256:<digest>`. To update one, resolve the new tag with `docker buildx imagetools inspect <name>:<tag>`, copy the top-level `Digest` of the multi-platform index, and update both the tag and the digest.

Packages installed with `apk add` or `apt-get install` come from the distribution mirror at build time, so their exact versions can change between builds even with pinned base images.
