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
3. Add an executable `check.sh` that runs from its exercise directory and returns zero only when the exercise passes.
4. Add the exercise to `exercises/manifest.json` with an ID matching the directory and a useful `default_name` fallback.
5. Add the same exercise ID to every `i18n/exercises/<language>.json` file with a non-empty translated title.
6. Keep NGINX configuration syntax and comments in English. Keep English instructions in `README.en.md` and Turkish instructions in `README.tr.md`.
7. For Docker exercises, use the scoped resource labels and image naming conventions already used by existing checks. Do not prune global Docker resources.
8. Pin Docker base image patch versions consistently with existing exercises.
9. If the exercise introduces a new asset file type, add its glob to `.goreleaser.yml` and inspect a snapshot archive.
10. Do not commit logs, generated certificates, PID files, check response bodies, or other runtime artifacts.

Exercise checks should remain understandable on failure and must not require Docker, NGINX, curl, or Python merely to run the Go unit test suite.

## Translations

- `i18n/langs.json` defines supported languages.
- `i18n/ui/<language>.json` files must contain the same UI keys.
- `i18n/exercises/<language>.json` files must contain exactly the manifest exercise IDs.
- Missing runtime translations still fall back safely, but repository validation requires complete checked-in translations.
- Keep translated README documents structurally aligned when adding or removing sections.

## Validation

Run all checks before opening a pull request:

```sh
gofmt -w .
go test ./...
go vet ./...
bash -n exercises/*/check.sh
node --test packaging/prepare-release.test.mjs
```

CI also runs ShellCheck and validates the GoReleaser configuration.
