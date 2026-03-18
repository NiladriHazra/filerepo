import { execFileSync } from "node:child_process";
import * as fs from "node:fs";
import type { IncomingMessage } from "node:http";
import * as https from "node:https";
import * as path from "node:path";
import { pipeline } from "node:stream/promises";
import packageJson from "../package.json";

const VERSION = packageJson.version;
const REPO = "NiladriHazra/filerepo";

const PLATFORM_MAP = {
  darwin: "apple-darwin",
  linux: "unknown-linux-gnu",
  win32: "pc-windows-msvc",
} as const;

const ARCH_MAP = {
  x64: "x86_64",
  arm64: "aarch64",
} as const;

function getTarget(): string {
  const platform = PLATFORM_MAP[process.platform as keyof typeof PLATFORM_MAP];
  const arch = ARCH_MAP[process.arch as keyof typeof ARCH_MAP];

  if (!platform || !arch) {
    throw new Error(`Unsupported platform: ${process.platform}-${process.arch}`);
  }

  return `${arch}-${platform}`;
}

function binaryName(): string {
  return process.platform === "win32" ? "filerepo.exe" : "filerepo";
}

function downloadUrl(): string {
  const target = getTarget();
  const extension = process.platform === "win32" ? ".zip" : ".tar.gz";
  return `https://github.com/${REPO}/releases/download/v${VERSION}/filerepo-${target}${extension}`;
}

function fetchStream(url: string): Promise<IncomingMessage> {
  return new Promise((resolve, reject) => {
    const request = https.get(
      url,
      { headers: { "User-Agent": "filerepo-installer" } },
      (response) => {
        const statusCode = response.statusCode ?? 0;
        const location = response.headers.location;

        if (statusCode >= 300 && statusCode < 400 && location) {
          response.resume();
          void fetchStream(location).then(resolve, reject);
          return;
        }

        if (statusCode !== 200) {
          response.resume();
          reject(new Error(`Download failed: HTTP ${statusCode} from ${url}`));
          return;
        }

        resolve(response);
      },
    );

    request.on("error", reject);
  });
}

async function downloadToFile(stream: IncomingMessage, filePath: string): Promise<void> {
  await pipeline(stream, fs.createWriteStream(filePath));
}

async function extractArchive(
  stream: IncomingMessage,
  dest: string,
  tempFile: string,
  tarFlag: "-xf" | "-xzf",
): Promise<void> {
  await downloadToFile(stream, tempFile);

  try {
    execFileSync("tar", [tarFlag, tempFile, "-C", dest], { stdio: "ignore" });
  } finally {
    if (fs.existsSync(tempFile)) {
      fs.unlinkSync(tempFile);
    }
  }
}

async function install(): Promise<void> {
  const dest = path.join(__dirname, "..", "bin");
  const binaryPath = path.join(dest, binaryName());

  if (fs.existsSync(binaryPath)) {
    return;
  }

  fs.mkdirSync(dest, { recursive: true });

  const url = downloadUrl();
  console.log(`Downloading filerepo from ${url}`);

  const stream = await fetchStream(url);
  const tempFile = path.join(
    dest,
    process.platform === "win32" ? "_download.zip" : "_download.tar.gz",
  );

  await extractArchive(stream, dest, tempFile, process.platform === "win32" ? "-xf" : "-xzf");
  fs.chmodSync(binaryPath, 0o755);
  console.log("filerepo installed successfully.");
}

install().catch((error: unknown) => {
  const message = error instanceof Error ? error.message : String(error);
  console.error(`Failed to install filerepo: ${message}`);
  process.exit(1);
});
