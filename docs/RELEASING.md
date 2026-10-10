# Releasing Nginxlings

Releases are created from semantic version tags such as `v1.2.3`.

Pushing a tag runs `.github/workflows/release.yml`. GoReleaser builds Linux binaries for AMD64 and ARM64 and packs each one into a `tar.gz` archive together with `exercises/` (including the shared `exercises/lib/` check helpers), `i18n/`, `README.md`, `LICENSE`, `THIRD_PARTY_NOTICES.md`, `install.sh` and the menu entry and icon from `packaging/`, so asset discovery works after extraction and `install.sh` can install from it. The tag version is injected into `main.version`.

For each architecture it also builds Linux packages with nfpm: `.deb`, `.rpm` and Arch `.pkg.tar.zst`. They install the binary in `/usr/bin`, the exercises and translations in `/usr/share/nginxlings`, the menu entry and icon, `LICENSE` and `THIRD_PARTY_NOTICES.md` in `/usr/share/licenses/nginxlings`, and depend on bash, curl, NGINX, Python and Docker under each distribution's package names (Podman's `docker` on RHEL-like systems). `packaging/postinstall.sh` starts the Docker service. Alpine `.apk` packages are not built: apk-tools 2.14 and 3 reject nfpm's unsigned apk; Alpine and other distributions use `install.sh`.

The release publishes `checksums.txt` (SHA-256), and the workflow attests the artifacts with GitHub artifact attestations. Attestations need a public repository; while the repository is private the workflow skips that step, so such a release has checksums but no attestation.

Only GitHub Releases are published. Native macOS and Windows builds, npm packages, and Homebrew formulas are intentionally not produced.

## Before Tagging

1. Confirm that the release contents remain compatible with `AGPL-3.0-only`. `THIRD_PARTY_NOTICES.md` names the license of every module in the binary, and `go test ./...` fails when it is out of date.
2. Run the validation commands from `CONTRIBUTING.md`.
3. Run `goreleaser check` and `goreleaser release --snapshot --clean` with GoReleaser v2 in a clean checkout, so local files such as `exercises/*/logs/` do not end up in the packages.
4. Confirm that `./dist` contains two archives (`linux_amd64` and `linux_arm64`), two each of `.deb`, `.rpm` and `.pkg.tar.zst`, and `checksums.txt`.
5. Extract one archive into an empty directory and confirm that `./nginxlings --list` prints all exercises from there.
6. Install the packages in every supported distribution: `NGINXLINGS_PACKAGES=$PWD/dist go test ./tests/packages -v -count=1 -timeout 40m` (see `tests/TESTING.md`).
7. `go test ./...` fails when an exercise or translation file is not covered by an archive glob in `.goreleaser.yml`, or when a glob matches nothing. After adding a new asset file type, also confirm that the extracted archive contains the file.

Push an annotated tag only after those checks pass:

```sh
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```
