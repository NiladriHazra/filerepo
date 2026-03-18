#!/usr/bin/env node
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
Object.defineProperty(exports, "__esModule", { value: true });
const node_child_process_1 = require("node:child_process");
const fs = __importStar(require("node:fs"));
const path = __importStar(require("node:path"));
const binaryName = process.platform === "win32" ? "filerepo.exe" : "filerepo";
function resolveBinary() {
    const candidates = [
        path.resolve(__dirname, "..", "target", "release", binaryName),
        path.resolve(__dirname, "..", "target", "debug", binaryName),
        path.join(__dirname, binaryName),
    ];
    return candidates.find((candidate) => fs.existsSync(candidate)) ?? null;
}
const binary = resolveBinary();
if (!binary) {
    console.error("filerepo binary not found. Build with `cargo build --release` or reinstall.");
    process.exit(1);
}
const result = (0, node_child_process_1.spawnSync)(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
    console.error(`Failed to launch filerepo: ${result.error.message}`);
    process.exit(1);
}
process.exit(result.status ?? 0);
