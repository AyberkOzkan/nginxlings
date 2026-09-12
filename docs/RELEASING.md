# Releasing Nginxlings

Releases are created from semantic version tags such as `v1.2.3`.

The release workflow builds Linux and macOS binaries for AMD64 and ARM64. Each archive contains the binary together with `exercises/`, `i18n/`, and the root README and LICENSE so asset discovery works after extraction. GoReleaser injects the tag version into `main.version` and publishes SHA-256 checksums.

The workflow also prepares two package-manager artifacts:

- an npm package containing all four supported binaries and the shared assets;
- a Homebrew formula with platform-specific archive URLs and checksums.

These artifacts are attached to the GitHub release. Publishing to the npm registry and pushing to a Homebrew tap are intentionally disabled until the final repository owner, package ownership, and registry credentials are configured.

## Before Tagging

1. Confirm the repository has a GitHub remote and that release contents remain compatible with `AGPL-3.0-only`.
2. Run the project validation commands from `README.md`.
3. Run `node --test packaging/prepare-release.test.mjs`.
4. Run `goreleaser check` and `goreleaser release --snapshot --clean` with GoReleaser v2.
5. Confirm `./dist` contains four archives and `checksums.txt`.

6. If an exercise adds a new asset file type, include its glob in `.goreleaser.yml` and inspect one snapshot archive.

Push an annotated tag only after those checks pass:

```sh
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```

## Package Publication

The generated npm package is under `dist/npm`, and the generated formula is under `dist/homebrew/Formula`. Registry publication should be added as a separate authenticated workflow step after ownership decisions are finalized. Keep GitHub release creation independent from external registry availability so npm or tap outages cannot block binary releases.
