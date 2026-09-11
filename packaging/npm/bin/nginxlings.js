#!/usr/bin/env node

import { existsSync } from "node:fs";
import { spawn } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const platformNames = new Map([
  ["linux", "linux"],
  ["darwin", "darwin"],
]);
const architectureNames = new Map([
  ["x64", "amd64"],
  ["arm64", "arm64"],
]);

const platform = platformNames.get(process.platform);
const architecture = architectureNames.get(process.arch);

if (!platform || !architecture) {
  console.error(`nginxlings does not support ${process.platform}/${process.arch}`);
  process.exit(1);
}

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const binary = resolve(
  packageRoot,
  "vendor",
  `${platform}-${architecture}`,
  "nginxlings",
);

if (!existsSync(binary)) {
  console.error(`nginxlings binary is missing for ${platform}/${architecture}`);
  process.exit(1);
}

const child = spawn(binary, process.argv.slice(2), {
  stdio: "inherit",
  env: {
    ...process.env,
    NGINXLINGS_ASSET_ROOT: packageRoot,
  },
});

child.on("error", (error) => {
  console.error(`failed to start nginxlings: ${error.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
