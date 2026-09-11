import assert from "node:assert/strict";
import { mkdir, mkdtemp, readFile, stat, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

import { normalizeVersion, prepareRelease, shouldPackageExerciseFile } from "./prepare-release.mjs";

const projectRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const targets = [
  ["darwin", "amd64"],
  ["darwin", "arm64"],
  ["linux", "amd64"],
  ["linux", "arm64"],
];

test("prepareRelease creates an asset-aware npm package and Homebrew formula", async () => {
  const temporaryRoot = await mkdtemp(join(tmpdir(), "nginxlings-release-"));
  const distDir = join(temporaryRoot, "dist");
  const artifacts = [];

  for (const [goos, goarch] of targets) {
    const binaryPath = join(distDir, "build", `${goos}-${goarch}`, "nginxlings");
    const archiveName = `nginxlings_1.2.3_${goos}_${goarch}.tar.gz`;
    const archivePath = join(distDir, archiveName);
    await mkdir(dirname(binaryPath), { recursive: true });
    await writeFile(binaryPath, `binary-${goos}-${goarch}`);
    await writeFile(archivePath, `archive-${goos}-${goarch}`);
    artifacts.push({ type: "Binary", goos, goarch, path: binaryPath });
    artifacts.push({ type: "Archive", goos, goarch, path: archivePath, name: archiveName });
  }
  await writeFile(join(distDir, "artifacts.json"), JSON.stringify(artifacts));

  const output = await prepareRelease({
    projectRoot,
    distDir,
    version: "v1.2.3",
    tag: "v1.2.3",
    repository: "example/nginxlings",
  });

  const packageJSON = JSON.parse(
    await readFile(join(output.packageRoot, "package.json"), "utf8"),
  );
  assert.equal(packageJSON.version, "1.2.3");
  assert.equal(packageJSON.private, false);
  assert.equal(packageJSON.license, "AGPL-3.0-only");
  assert.equal(packageJSON.author, "Kenan Ayberk Özkan");
  assert.equal(packageJSON.repository.url, "https://github.com/example/nginxlings.git");
  await stat(join(output.packageRoot, "exercises", "manifest.json"));
  await stat(join(output.packageRoot, "LICENSE"));
  await stat(join(output.packageRoot, "i18n", "ui", "en.json"));
  await stat(join(output.packageRoot, "vendor", "linux-amd64", "nginxlings"));
  await assert.rejects(stat(join(output.packageRoot, "exercises", "01_static_server", "logs")));

  const formula = await readFile(output.formulaPath, "utf8");
  assert.match(formula, /version "1\.2\.3"/);
  assert.match(formula, /on_macos do/);
  assert.match(formula, /on_linux do/);
  assert.match(formula, /NGINXLINGS_ASSET_ROOT: libexec/);
  assert.match(formula, /releases\/download\/v1\.2\.3/);
});

test("prepareRelease rejects an incomplete target matrix", async () => {
  const temporaryRoot = await mkdtemp(join(tmpdir(), "nginxlings-release-"));
  const distDir = join(temporaryRoot, "dist");
  await mkdir(distDir, { recursive: true });
  await writeFile(join(distDir, "artifacts.json"), "[]");

  await assert.rejects(
    prepareRelease({
      projectRoot,
      distDir,
      version: "1.2.3",
      tag: "v1.2.3",
      repository: "example/nginxlings",
    }),
    /missing binary for darwin-amd64/,
  );
});

test("shouldPackageExerciseFile excludes runtime artifacts", () => {
  const root = join("tmp", "exercises");
  assert.equal(shouldPackageExerciseFile(root, join(root, "01_static_server", "README.en.md")), true);
  assert.equal(shouldPackageExerciseFile(root, join(root, "01_static_server", "logs", "access.log")), false);
  assert.equal(shouldPackageExerciseFile(root, join(root, "01_static_server", ".check_body.123")), false);
  assert.equal(shouldPackageExerciseFile(root, join(root, "01_static_server", "response.html")), false);
  assert.equal(shouldPackageExerciseFile(root, join(root, "01_static_server", "nginx.pid")), false);
});

test("normalizeVersion accepts release versions and rejects malformed input", () => {
  assert.equal(normalizeVersion("v2.3.4-rc.1"), "2.3.4-rc.1");
  assert.throws(() => normalizeVersion("latest"), /invalid release version/);
});
