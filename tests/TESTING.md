# Testing Nginxlings

This document lists every test layer, the commands to run, and the steps each
layer covers. A Turkish version is in `TESTING.tr.md`.

## Test layers

| Layer | Location | What it tests | Needs | Time | In CI |
| --- | --- | --- | --- | --- | --- |
| Unit | `cmd/nginxlings/*_test.go` | Program internals: layout math, update logic, state, CLI, i18n loading | Go | ~1 s | yes |
| Repository | `tests/repo` | Contracts between files: manifest, translations, README structure and titles, check script conventions, starting states, release and CI configuration | Go | <1 s | yes |
| Check library | `tests/checklib` | `exercises/lib/check-lib.sh` with bash and fake binaries | Go, bash | ~1 s | yes |
| TUI | `tests/tui` | The built program in a pseudo-terminal, read through a terminal emulator | Go, Linux pty | ~3 s | yes |
| Exercises | `tests/exercises` | Every exercise's real `check.sh`: starting files fail, the solution passes | Docker, host NGINX, solutions | ~90 s | no (skipped) |

Unit tests stay next to the code because Go tests of unexported code must be in
the same package. Everything under `tests/` treats the repository and the
program as a black box and shares helpers from `tests/testutil`.

## Every change

```sh
gofmt -l .                      # must print nothing
go vet ./...
go test ./...                   # unit, repository, check library, TUI
for script in exercises/*/check.sh exercises/lib/*.sh; do bash -n "$script"; done
```

`go test ./...` skips the exercise tests unless `NGINXLINGS_SOLUTIONS` is set.

## Before a release

```sh
go test -race ./cmd/nginxlings ./tests/tui
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
NGINXLINGS_SOLUTIONS=$HOME/projects/nginxlings-solutions/solutions \
  go test ./tests/exercises -v -count=1 -timeout 30m
```

The exercise tests need a running Docker daemon for the container exercises
and `nginx` on the host for 01, 02, 03 and 06. A missing prerequisite skips the
affected exercise and prints the reason; run with `-v` to see it. Host checks
run in a private network namespace (`unshare -rn`) when the kernel allows it,
so services on the machine's ports 8080 and 8081 do not interfere.

Solutions are never committed. `NGINXLINGS_SOLUTIONS` points at a directory
with one subdirectory per exercise ID that holds only the files that differ
from the starting files.

## Test steps

### TUI (`tests/tui`)

| Step | Test |
| --- | --- |
| Start at 15 sizes from 30x6 to 220x60, in English and Turkish; the first frame is complete | `TestTUIFrameAtEachSize` |
| Resize one running program through growing and shrinking sizes; each frame settles and stays intact | `TestTUIResizeKeepsFrameIntact` |
| The frame check itself rejects damaged screens | `TestFrameCheckRejectsDamagedScreens` |
| `s`/`w` move the selection; keys typed faster than a redraw are all counted | `TestTUINavigationCountsEveryKeyPress` |
| `h` opens and closes the hint | `TestTUIHintTogglesWithInstructions` |
| `l` switches the language of titles, list names and the README | `TestTUILanguageSwitchTranslatesTheScreen` |
| PgDn/PgUp scroll the instructions | `TestTUIScrollMovesTheInstructions` |
| `c` runs a passing and a failing check; output panel, `Tab`, list marks, progress and the saved state follow | `TestTUICheckFlowShowsResultAndProgress` |
| At 80x24 a long check output opens at its end and a long verdict wraps in the status rows | `TestTUILongCheckResultStaysReadable` |
| `r` asks for confirmation; `n` cancels, `y` clears progress | `TestTUIResetClearsProgress` |
| A restart restores the last selected exercise | `TestTUIRestartRestoresTheSelectedExercise` |

A valid frame means: the title on the first row; under the panels the status
rows, then the key help rows down to the last row, with no empty row between
(the status wraps onto up to 4 rows and the key help onto up to 3 when the
instructions keep 6 rows); two closed panels whose right border reaches the
last column; panels side by side from 80 columns and stacked below;
and at most three rows without borders when the terminal is shorter than 9 rows,
or narrower than 80 and shorter than 14 rows.

The check flow uses a two-exercise fake asset root whose checks answer at once,
so it needs neither Docker nor NGINX.

### Exercises (`tests/exercises`)

For each exercise directory:

1. Copy the exercise and `exercises/lib` to a temporary directory and run
   `check.sh`: the last line must start with `[FAIL]` and the exit code must be
   non-zero.
2. Copy again, apply the solution, run `check.sh`: `[PASS]` and exit code 0.
3. Apply the solution, rewrite `nginx.conf`, `conf.d/*.conf` and
   `server.d/*.conf` onto one line without comments, run `check.sh`: `[PASS]`.

### Repository (`tests/repo`)

Run on every `go test ./...`. Among other things: manifest and directories
match; every translation file covers every exercise; README titles equal the
list names in each language; README heading levels match across languages;
starting files do not contain the finished solution; checks use the shared
library and pinned images; release archives include every asset.

## Watching the TUI live

To watch a manual session, start the program in a tmux session on a socket you
can reach and attach read-only from another terminal:

```sh
go build -o /tmp/nginxlings ./cmd/nginxlings
tmux -S /tmp/nginxlings.sock new-session -d -s nlx -x 80 -y 24 \
  "/tmp/nginxlings --state /tmp/nginxlings-state.json"
tmux -S /tmp/nginxlings.sock set-option -t nlx window-size manual
tmux -S /tmp/nginxlings.sock attach -t nlx -r    # in another terminal; Ctrl+b d detaches
```

Use a separate `--state` file so a test session does not change your own
progress.

## Adding tests

- **New exercise:** add its solution to the solutions directory and run the
  exercise tests; add an entry to `tests/repo/starting_state_test.go` when the
  starting files could be mistaken for a solution.
- **New key or view:** add a flow to `tests/tui/flow_test.go`. Wait for text with
  `waitForText` or for a full frame with `waitForFrame`; never sleep for a fixed
  time.
- **Layout change:** add sizes to `frameSizes` in `tests/tui/frame_test.go` and
  keep `isCompact` equal to the rule in `cmd/nginxlings/view.go`.
- **New file contract:** add a test to the matching file in `tests/repo`.

## Troubleshooting

- `Port ... is already in use`: another program holds a check's port. Stop it or
  override the port with the variable the check names.
- `this check needs a running Docker daemon`: start Docker; the exercise was
  skipped, not failed.
- Host checks fail with bind errors although the ports look free: `unshare` may
  be blocked (for example inside some containers); the checks then run on the
  host network.
