#!/usr/bin/env node

"use strict";

const { execFileSync } = require("node:child_process");
const path = require("node:path");
const fs = require("node:fs");

const binary = path.join(__dirname, process.platform === "win32" ? "filerepo.exe" : "filerepo");

if (!fs.existsSync(binary)) {
  console.error("filerepo binary not found. Run `npm rebuild filerepo` or reinstall.");
  process.exit(1);
}

try {
  execFileSync(binary, process.argv.slice(2), { stdio: "inherit" });
} catch (err) {
  process.exit(err.status ?? 1);
}
