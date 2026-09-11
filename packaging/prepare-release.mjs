#!/usr/bin/env node

import { createHash } from "node:crypto";
import { chmod, copyFile, cp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const targets = [
  { goos: "darwin", goarch: "amd64" },
  { goos: "darwin", goarch: "arm64" },
  { goos: "linux", goarch: "amd64" },
  { goos: "linux", goarch: "arm64" },
];

export async function prepareRelease({ projectRoot, distDir, version, tag, repository }) {
  const normalizedVersion = normalizeVersion(version);
  validateTag(tag);
  validateRepository(repository);

  const root = resolve(projectRoot);
  const dist = resolve(distDir);
  const packageRoot = join(dist, "npm", "package");
  const formulaPath = join(dist, "homebrew", "Formula", "nginxlings.rb");
  assertChildPath(dist, packageRoot);
  assertChildPath(dist, formulaPath);

  const artifactData = await readFile(join(dist, "artifacts.json"), "utf8");
  const artifacts = JSON.parse(artifactData);

  await rm(packageRoot, { recursive: true, force: true });
  await mkdir(join(packageRoot, "bin"), { recursive: true });
  const exercisesRoot = join(root, "exercises");
  await cp(exercisesRoot, join(packageRoot, "exercises"), {
    recursive: true,
    filter: (source) => shouldPackageExerciseFile(exercisesRoot, source),
  });
  await cp(join(root, "i18n"), join(packageRoot, "i18n"), { recursive: true });
  await copyFile(join(root, "LICENSE"), join(packageRoot, "LICENSE"));
  await copyFile(join(root, "README.md"), join(packageRoot, "README.md"));
  await copyFile(
    join(root, "packaging", "npm", "bin", "nginxlings.js"),
    join(packageRoot, "bin", "nginxlings.js"),
  );
  await chmod(join(packageRoot, "bin", "nginxlings.js"), 0o755);

  for (const target of targets) {
    const binary = findArtifact(artifacts, target, "Binary");
    const targetDir = join(packageRoot, "vendor", `${target.goos}-${target.goarch}`);
    await mkdir(targetDir, { recursive: true });
    const binaryPath = join(targetDir, "nginxlings");
    await copyFile(resolveArtifactPath(root, binary.path), binaryPath);
    await chmod(binaryPath, 0o755);
  }

  const packageTemplate = JSON.parse(
    await readFile(join(root, "packaging", "npm", "package.json"), "utf8"),
  );
  packageTemplate.version = normalizedVersion;
  packageTemplate.private = false;
  packageTemplate.repository = {
    type: "git",
    url: `https://github.com/${repository}.git`,
  };
  packageTemplate.homepage = `https://github.com/${repository}`;
  packageTemplate.bugs = `https://github.com/${repository}/issues`;
  await writeFile(
    join(packageRoot, "package.json"),
    `${JSON.stringify(packageTemplate, null, 2)}\n`,
  );

  const archives = new Map();
  for (const target of targets) {
    const artifact = findArtifact(artifacts, target, "Archive");
    const archivePath = resolveArtifactPath(root, artifact.path);
    archives.set(targetKey(target), {
      name: artifact.name ?? artifact.path.split(/[\\/]/).at(-1),
      sha256: await fileSHA256(archivePath),
    });
  }

  await mkdir(dirname(formulaPath), { recursive: true });
  await writeFile(
    formulaPath,
    renderFormula({ version: normalizedVersion, tag, repository, archives }),
  );

  return { packageRoot, formulaPath };
}

export function normalizeVersion(version) {
  const normalized = String(version).replace(/^v/, "");
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(normalized)) {
    throw new Error(`invalid release version: ${version}`);
  }
  return normalized;
}

function validateTag(tag) {
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(tag)) {
    throw new Error(`invalid release tag: ${tag}`);
  }
}

function validateRepository(repository) {
  if (!/^[0-9A-Za-z_.-]+\/[0-9A-Za-z_.-]+$/.test(repository)) {
    throw new Error(`invalid GitHub repository: ${repository}`);
  }
}

function assertChildPath(parent, child) {
  const path = relative(parent, child);
  if (path === "" || path === ".." || path.startsWith(`..${sep}`) || isAbsolute(path)) {
    throw new Error(`refusing to write outside dist directory: ${child}`);
  }
}

export function shouldPackageExerciseFile(exercisesRoot, source) {
  const path = relative(exercisesRoot, source);
  if (path === "") {
    return true;
  }

  const parts = path.split(sep);
  const name = parts.at(-1);
  return (
    !parts.includes("logs") &&
    !name.startsWith(".check_body") &&
    name !== "response.html" &&
    name !== "nginx_test.log" &&
    !name.endsWith(".pid")
  );
}

function findArtifact(artifacts, target, type) {
  const artifact = artifacts.find(
    (candidate) =>
      candidate.type === type &&
      candidate.goos === target.goos &&
      candidate.goarch === target.goarch,
  );
  if (!artifact) {
    throw new Error(`missing ${type.toLowerCase()} for ${targetKey(target)}`);
  }
  return artifact;
}

function resolveArtifactPath(projectRoot, artifactPath) {
  return isAbsolute(artifactPath) ? artifactPath : resolve(projectRoot, artifactPath);
}

function targetKey(target) {
  return `${target.goos}-${target.goarch}`;
}

async function fileSHA256(path) {
  return createHash("sha256").update(await readFile(path)).digest("hex");
}

function renderFormula({ version, tag, repository, archives }) {
  const targetBlock = (goos, rubyOS) => {
    const amd64 = archives.get(`${goos}-amd64`);
    const arm64 = archives.get(`${goos}-arm64`);
    return `  on_${rubyOS} do
    if Hardware::CPU.arm?
      url "https://github.com/${repository}/releases/download/${tag}/${arm64.name}"
      sha256 "${arm64.sha256}"
    else
      url "https://github.com/${repository}/releases/download/${tag}/${amd64.name}"
      sha256 "${amd64.sha256}"
    end
  end`;
  };

  return `class Nginxlings < Formula
  desc "Terminal training app for learning NGINX through runnable exercises"
  homepage "https://github.com/${repository}"
  version "${version}"

${targetBlock("darwin", "macos")}

${targetBlock("linux", "linux")}

  def install
    libexec.install "nginxlings", "exercises", "i18n"
    (bin/"nginxlings").write_env_script libexec/"nginxlings",
      NGINXLINGS_ASSET_ROOT: libexec
  end

  test do
    assert_match "nginxlings #{version}", shell_output("#{bin}/nginxlings --version")
    assert_match "01_static_server", shell_output("#{bin}/nginxlings --list")
  end
end
`;
}

function parseArguments(args) {
  const options = {};
  for (let index = 0; index < args.length; index += 2) {
    const key = args[index];
    const value = args[index + 1];
    if (!key?.startsWith("--") || value === undefined) {
      throw new Error(`invalid argument: ${key ?? ""}`);
    }
    options[key.slice(2)] = value;
  }
  for (const required of ["version", "tag", "repository", "dist"]) {
    if (!options[required]) {
      throw new Error(`missing --${required}`);
    }
  }
  return options;
}

const entrypoint = process.argv[1] ? pathToFileURL(resolve(process.argv[1])).href : "";
if (import.meta.url === entrypoint) {
  try {
    const options = parseArguments(process.argv.slice(2));
    const projectRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
    const output = await prepareRelease({
      projectRoot,
      distDir: resolve(projectRoot, options.dist),
      version: options.version,
      tag: options.tag,
      repository: options.repository,
    });
    console.log(`npm package prepared in ${output.packageRoot}`);
    console.log(`Homebrew formula written to ${output.formulaPath}`);
  } catch (error) {
    console.error(error.message);
    process.exit(1);
  }
}
