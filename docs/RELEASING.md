# Releasing Nginxlings

Releases are created from semantic version tags such as `v1.2.3`.

Pushing a tag runs `.github/workflows/release.yml`. GoReleaser builds Linux binaries for AMD64 and ARM64 and packs each one into a `tar.gz` archive together with `exercises/` (including the shared `exercises/lib/` check helpers), `i18n/`, `README.md`, and `LICENSE`, so asset discovery works after extraction. The tag version is injected into `main.version`. The release publishes `checksums.txt` (SHA-256), and the workflow attests the archives with GitHub artifact attestations.

Only GitHub Releases are published. Native macOS and Windows builds, npm packages, and Homebrew formulas are intentionally not produced.

## Before Tagging

1. Confirm that the release contents remain compatible with `AGPL-3.0-only`.
2. Run the development checks from `README.md`.
3. Run `goreleaser check` and `goreleaser release --snapshot --clean` with GoReleaser v2.
4. Confirm that `./dist` contains two archives (`linux_amd64` and `linux_arm64`) and `checksums.txt`.
5. Extract one archive into an empty directory and confirm that `./nginxlings --list` prints all exercises from there.
6. `go test ./...` fails when an exercise or translation file is not covered by an archive glob in `.goreleaser.yml`, or when a glob matches nothing. After adding a new asset file type, also confirm that the extracted archive contains the file.

Push an annotated tag only after those checks pass:

```sh
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```
