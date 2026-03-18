"use strict";

const https = require("node:https");
const fs = require("node:fs");
const path = require("node:path");
const { createGunzip } = require("node:zlib");

const VERSION = require("../package.json").version;
const REPO = "niladri/filerepo"; // ← change to your GitHub username/repo

const PLATFORM_MAP = {
  darwin: "apple-darwin",
  linux: "unknown-linux-gnu",
  win32: "pc-windows-msvc",
};

const ARCH_MAP = {
  x64: "x86_64",
  arm64: "aarch64",
};

function getTarget() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    throw new Error(`Unsupported platform: ${process.platform}-${process.arch}`);
  }

  return `${arch}-${platform}`;
}

function binaryName() {
  return process.platform === "win32" ? "filerepo.exe" : "filerepo";
}

function downloadUrl() {
  const target = getTarget();
  const ext = process.platform === "win32" ? ".zip" : ".tar.gz";
  return `https://github.com/${REPO}/releases/download/v${VERSION}/filerepo-${target}${ext}`;
}

function fetch(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { "User-Agent": "filerepo-installer" } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return fetch(res.headers.location).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`Download failed: HTTP ${res.statusCode} from ${url}`));
      }
      resolve(res);
    }).on("error", reject);
  });
}

async function extractTarGz(stream, dest) {
  const { execSync } = require("node:child_process");
  const tmp = path.join(dest, "_download.tar.gz");

  await new Promise((resolve, reject) => {
    const file = fs.createWriteStream(tmp);
    stream.pipe(file);
    file.on("finish", resolve);
    file.on("error", reject);
  });

  execSync(`tar -xzf "${tmp}" -C "${dest}"`, { stdio: "ignore" });
  fs.unlinkSync(tmp);
}

async function extractZip(stream, dest) {
  const { execSync } = require("node:child_process");
  const tmp = path.join(dest, "_download.zip");

  await new Promise((resolve, reject) => {
    const file = fs.createWriteStream(tmp);
    stream.pipe(file);
    file.on("finish", resolve);
    file.on("error", reject);
  });

  execSync(`tar -xf "${tmp}" -C "${dest}"`, { stdio: "ignore" });
  fs.unlinkSync(tmp);
}

async function install() {
  const dest = path.join(__dirname, "..", "bin");
  const binaryPath = path.join(dest, binaryName());

  if (fs.existsSync(binaryPath)) {
    return;
  }

  const url = downloadUrl();
  console.log(`Downloading filerepo from ${url}`);

  const stream = await fetch(url);

  if (process.platform === "win32") {
    await extractZip(stream, dest);
  } else {
    await extractTarGz(stream, dest);
  }

  fs.chmodSync(binaryPath, 0o755);
  console.log("filerepo installed successfully.");
}

install().catch((err) => {
  console.error(`Failed to install filerepo: ${err.message}`);
  process.exit(1);
});
