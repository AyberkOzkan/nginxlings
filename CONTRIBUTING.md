# Contributing to Nginxlings

Contributions to the official Nginxlings project are welcome. Open an issue before substantial changes so the scope and exercise learning goal can be agreed with the maintainers.

By submitting a contribution, you agree that it is licensed under `AGPL-3.0-only`, as described in `LICENSE`. Only submit work that you created or have the right to contribute under these terms.

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
