import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const file = path.join(path.dirname(fileURLToPath(import.meta.url)), "..", "recovery/scanner/wallet_extensions.go");
const text = fs.readFileSync(file, "utf8");
const total = (text.match(/"[a-z]{32}":/g) || []).length;
const fallback = (text.match(/": "Extension [a-z]{8}",/g) || []).length;
const generic = (text.match(/": "Wallet",/g) || []).length;
console.log({ total, named: total - fallback - generic, fallback, generic, ok: fallback === 0 && generic === 0 });