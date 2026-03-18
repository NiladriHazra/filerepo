"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const node_child_process_1 = require("node:child_process");
const fs = __importStar(require("node:fs"));
const https = __importStar(require("node:https"));
const path = __importStar(require("node:path"));
const promises_1 = require("node:stream/promises");
const package_json_1 = __importDefault(require("../package.json"));
const VERSION = package_json_1.default.version;
const REPO = "NiladriHazra/filerepo";
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
    const extension = process.platform === "win32" ? ".zip" : ".tar.gz";
    return `https://github.com/${REPO}/releases/download/v${VERSION}/filerepo-${target}${extension}`;
}
function fetchStream(url) {
    return new Promise((resolve, reject) => {
        const request = https.get(url, { headers: { "User-Agent": "filerepo-installer" } }, (response) => {
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
        });
        request.on("error", reject);
    });
}
async function downloadToFile(stream, filePath) {
    await (0, promises_1.pipeline)(stream, fs.createWriteStream(filePath));
}
async function extractArchive(stream, dest, tempFile, tarFlag) {
    await downloadToFile(stream, tempFile);
    try {
        (0, node_child_process_1.execFileSync)("tar", [tarFlag, tempFile, "-C", dest], { stdio: "ignore" });
    }
    finally {
        if (fs.existsSync(tempFile)) {
            fs.unlinkSync(tempFile);
        }
    }
}
async function install() {
    const dest = path.join(__dirname, "..", "bin");
    const binaryPath = path.join(dest, binaryName());
    if (fs.existsSync(binaryPath)) {
        return;
    }
    fs.mkdirSync(dest, { recursive: true });
    const url = downloadUrl();
    console.log(`Downloading filerepo from ${url}`);
    const stream = await fetchStream(url);
    const tempFile = path.join(dest, process.platform === "win32" ? "_download.zip" : "_download.tar.gz");
    await extractArchive(stream, dest, tempFile, process.platform === "win32" ? "-xf" : "-xzf");
    fs.chmodSync(binaryPath, 0o755);
    console.log("filerepo installed successfully.");
}
install().catch((error) => {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`Failed to install filerepo: ${message}`);
    process.exit(1);
});
