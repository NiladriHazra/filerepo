#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";

const binaryName = process.platform === "win32" ? "filerepo.exe" : "filerepo";

function resolveBinary(): string | null {
  const candidates = [
    path.resolve(__dirname, "..", binaryName),
    path.resolve(__dirname, "..", "dist", binaryName),
    path.join(__dirname, binaryName),
  ];

  return (
    candidates.find((candidate) => {
      try {
        return fs.statSync(candidate).isFile();
      } catch {
        return false;
      }
    }) ?? null
  );
}

const binary = resolveBinary();

if (!binary) {
  console.error(
    "filerepo binary not found. Build with `go build -o bin/filerepo ./cmd/filerepo` or reinstall.",
  );
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  console.error(`Failed to launch filerepo: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 0);
