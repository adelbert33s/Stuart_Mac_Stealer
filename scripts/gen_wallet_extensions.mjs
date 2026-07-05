import fs from "fs";
import https from "https";
import path from "path";
import { fileURLToPath } from "url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const EXCLUDE = new Set([
  // Password managers / authenticators / non-wallet extensions
  "aeblfdkhhhdcdjpifhhbdiojplfjncoa", // 1Password
  "khgocmkkpikpnmmkgmdnfckapcdkgfaf",
  "gejiddohjgogedgjnonbofjigllpkmbf",
  "hdokiejnpimakedhajhdlcegeplioahd", // LastPass
  "ghmbeldphafepmbegfdlkpapadhbakde", // Proton Pass
  "fdjamakpfbbddfjaooikfcpapjohcfmg", // Dashlane
  "bhghoamapcdpbohphigoooaddinpkbai", // Google Authenticator
  "pioclpoplcdbaefihamjohnefbikjilc", // Evernote Web Clipper
  "nngceckbapebfimnlniiiahkandclblb", // Bitwarden
  "ilgcnhelpchnceeipipijaljkblbcobl", // GAuth Authenticator
  "bfogiafebfohielmmehodmfbbebbbpei", // Keeper Password Manager
  "cnlhokffphohmfcddnibpohmkdfafdli", // MultiPassword
  "eiaeiblijfjekdanodkjadfinkhbfgcd", // NordPass
  "admmjipmmciaobhojoghlmleefbicajg", // Norton Password Manager
  "caljgklbbfbcjjanaijlacgncafpegll", // Avira Password Manager
  "pnlccmojcmeohlpggmfnbbiapkmbliob", // RoboForm
  "lgbjhdkjmpgjgcbcdlhkokkckpjmedgc", // DualSafe Password Manager
  "mmhlniccooihdimnnjhamobppdhaolme", // Kee Password Manager
  "epanfjkfahimkgomnigadpkobaefekcd", // IronVest (vault)
]);

const NAMES = {
  nkbihfbeogaeaoehlefnkodbefgpgknn: "MetaMask",
  ejbalbakoplchlghecdalmeeeajnimhm: "MetaMask",
  bfnaelmomeimhlpmgjnjophhpkkoljpa: "Phantom",
  egjidjbpglichdcondbcbdnbeeppgdph: "Trust Wallet",
  hnfanknocfeofbddgcijnmhnfnkdnaad: "Coinbase Wallet",
  acmacodkjbdgmoleebolmdjonilkdbch: "Rabby",
  mcohilncbfahbmgdjkbpemcciiolgcge: "OKX Wallet",
  ppbibelpcjmhbdihakflkdcoccbgbkpo: "UniSat",

  aflkmfhebedbjioipglgcbcmnbpgliof: "Backpack",
  ldinpeekobnhjjdofggfgjlcehhmanlj: "Leather",
  onhogfjeacnfoofkfgppdlbmlmnplgbn: "SubWallet",
  jiidiaalihmmhddjgbnbgdfflelocpak: "Bitget Wallet",
  pdliaogehgdbhbnmkklieghmmjkpigpa: "Bybit Wallet",
  hifafgmccdpekplomjjkcfgodnhcellj: "Crypto.com Wallet",
  dlcobpjiigpikoobohmabehhmhfoodbb: "Argent X",
  klghhnkeealcohjjanjjdaeeggmfmlpl: "Zerion",
  dmkamcknogkgcdfhhbddcghachkejeap: "Keplr",
  fnjhmkhhmkbjkkabndcnnogagogbneec: "Ronin",
  aholpfdialjgjfhomihkjbmgjidlcdno: "Exodus Web3",
  ibnejdfjmmkpcnlpebklmnkoeoihofec: "TronLink",
  ffnbelfdoeiohenkjibnmadjiehjhajb: "Yoroi",
  ookjlbkiijinhpmnjffcofjonbfbgaoc: "Temple Tezos",
  bhhhlbepdkbapadjdnnojkbgioiodbic: "Solflare",
  lgmpcpglpngdoalbgeoldeajfclnhafa: "SafePal",
  mfgccjchihfkkindfppnaooecgfneiii: "TokenPocket",
  nphplpgoakhhjchkkhmiggakijnkhfnd: "Ton Wallet",
  idnnbdplmphpflfnlkomgpfbpcgelopg: "Xverse",
  kkpllkodjeloidieedojogacfhpaihoh: "Enkrypt",
  cphhlgmgameodnhkjdmkpanlelnlohao: "NeoLine",
  nhnkbkgjikgcigadomkphalanndcapjk: "CLV Wallet",
  mkpegjkblkkefacfnmkajcjmabijhclg: "Magic Eden",
  fcfcfllfndlomdhbehjjcoimbgofdncg: "Leap Cosmos",
  aijcbedoijmgnlmjeegjaglmepbmpkpi: "Leap Terra",
  khpkpbbcccdmmclmpigdgddabeilkdpd: "Suiet",
  loinekcabhlmhjjbocijdoimmejangoa: "Glass Wallet Sui",
  ocjdpmoallmgmjbbogfiiaofphbjgchh: "Elli Sui",
  ehgjhhccekdedpbkifaojjaefeohnoea: "Ambire",
  eaeecbmeajhliilmacefcgjnnijkkfki: "Trust Wallet Beta",
  fhbohimaelbohpjbbldcngcnapndodjp: "Binance Chain Wallet",
  fihkakfobkmkjojpchpfgcmhfjnmnfpi: "BitApp Wallet",
  aodkkagnadcbobfpggfnjeongemjbjca: "BoltX",
  aeachknmefphepccionboohckonoeemg: "Coin98",
  agoakfejjabomempkjlepdflaleeobhb: "Core Wallet",
  pnlfjmlcjdjgkddecgincndfgegkecke: "Crocobit",
  blnieiiffboillknjnepogjhkgnoapac: "Equal Wallet",
  cgeeodpfagjceefieflmdfphplkenlfk: "Ever Wallet",
  ebfidpplhabeedpnhjnobghokpiioolj: "Fewcha",
  cjmkndjhnagcfbpiemnkdpomccnjblmj: "Finnie",
  hpglfhgfnhbgpjdenjgmdgoeiappafln: "Guarda",
  nanjmdknhkinifnkgdcggcfnhdaammmj: "Guild Wallet",
  fnnegphlobjdpkhecapkijjdkgcjhkib: "Harmony Wallet",
  flpiciilemghbmfalicajoolhkkenfel: "Iconex",
  cjelfplplebdjjenllpjcblmjkfcffne: "Jaxx Liberty",
  jblndlipeogpafnldhgmapagcccfchpi: "Kaikas",
  pdadjkfkgcafgbceimcpbkalnfnepbnk: "KardiaChain",
  kpfopkelmapcoipemfendmdcghnegimn: "Liquality",
  nlbmnnijcnlegkjjpcfjclmcfggfefdm: "MEW CX",
  dngmlblcodfobpdpecaadgfbcggfjfnm: "Maiar DeFi",
  efbglgofoippbgcjepnhiblaibcnclgk: "Martian",
  afbcbjpbpfadlkmhmclhkeeodmamcflc: "Math Wallet",
  fcckkdbjnoikooededlapcalpionmalo: "Mobox",
  lpfcbjknijpeeillifnkikgncikgfhdo: "Nami",
  jbdaocneiiinmjbjlgalhcelgbejmnid: "Nifty Wallet",
  fhilaheimglignddkjgofkcbgekhenbh: "Oxygen Wallet",
  mgffkfbidihjpoaomajlbgchddlicgpn: "Pali Wallet",
  ejjladinnckdgjemekebdpeokbikhfci: "Petra",
  phkbamefinggmakgklpkljjmgibohnba: "Pontem",
  nkddgncdjgjfcddamfgcmfnlhccnimig: "Saturn Wallet",
  pocmplpaccanhmnllbbkpgfliimjljgo: "Slope",
  fhmfendgdocmcbmfikdcogofphimnkno: "Sollet",
  mfhbebgoclkghebffdldpobeajmbecfk: "Starcoin",
  cmndjbecilbocjfkibfbifhngkdmjgog: "Swash",
  aiifbnbfobpmeekipheeijimdpnlpgpp: "Terra Station",
  amkmjjmmflddogmhpjloimipbofnfjih: "Wombat",
  hmeobnfnfcmdkdcmlblgagmfpfboieaf: "XDEFI",
  eigblbgjknlfbajkfhopmcojidlgcehm: "XMR.PT",
  bocpokimicclpaiekenaeelehdjllofo: "XinPay",
  kncchdigobghenbbaddojjnnaogfppfj: "iWallet",
  opcgpfmipidbgpenhmajoajpbobppdil: "Sui Wallet",
  penjlddjkjgpnkllboccdgccekpkcbin: "OpenMask TON",
  lnnnmfcpbkafcpgdilckhmhbkkbpkmid: "Koala Wallet",
  dbgnhckhnppddckangcjbkjnlddbjkna: "Fin Wallet Sei",
  bifidjkcdpgfnlbcjpdkdcnbiooooblg: "Fuelet Fuel",
  anokgmphncpekkhclmingpimjmcooifb: "Compass Sei",
  fpkhgmpbidmiogeglndfbkegfdlnajnf: "Cosmostation",
  ohjgojhmjldjfningdelbffpnddmiphh: "NEAR Wallet",
  lccbohhgfkdikahanoclbdmaolidjdfl: "Wigwam",
  kmphdnilpmdejikjdnlbcnmnabepfgkh: "OsmWallet XRP",
  bfeplaecgkoeckiidkgkmlllfbaeplgm: "Radix Connector",
  penjlddjkjgpnkllboccdgccekpkcbin: "OpenMask",
  ldinpeekobnhjjdofggfgjlcehhmanlj: "Leather",
  cphhlgmgameodnhkjdmkpanlelnlohao: "NeoLine",
  oebgglckkdmdcphmbdcbdlkedjbbinii: "Sender",
  hifafgmccdpekplomjjkcfgodnhcellj: "Crypto.com Wallet",
};

// Extra wallet IDs not present in the reference extension list.
const EXTRA = {
  ldinpeekobnhjjdofggfgjlcehhmanlj: "Leather",
  cphhlgmgameodnhkjdmkpanlelnlohao: "NeoLine",
  oebgglckkdmdcphmbdcbdlkedjbbinii: "Sender",
};

function fetchText(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, (res) => {
        let body = "";
        res.on("data", (c) => (body += c));
        res.on("end", () => resolve(body));
      })
      .on("error", reject);
  });
}

function fetchJson(url) {
  return fetchText(url).then((body) => JSON.parse(body));
}

function cleanWalletLabel(raw) {
  return String(raw || "")
    .replace(/[^A-Za-z0-9 |&.'()-]/g, "")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 56);
}

/** Split HyperHives-style "MetaMasknkbihfbeogaeaoehlefnkodbefgpgknn" into name + id. */
function extractNameFromConcatenated(value) {
  const m = String(value).toLowerCase().match(/([a-z]{32})$/);
  if (!m) return null;
  const eid = m[1];
  const prefix = cleanWalletLabel(String(value).slice(0, -32));
  if (prefix.length < 3) return null;
  return { eid, name: prefix };
}

function setName(map, eid, name, force = false) {
  const label = cleanWalletLabel(name);
  if (!label || label.length < 3) return;
  const existing = map[eid];
  if (force || !existing || existing === "Wallet" || existing.startsWith("Extension ")) {
    map[eid] = label;
  }
}

function parseCsvLine(line) {
  const fields = [];
  let cur = "";
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const c = line[i];
    if (c === '"') {
      inQuotes = !inQuotes;
      continue;
    }
    if (c === "," && !inQuotes) {
      fields.push(cur);
      cur = "";
      continue;
    }
    cur += c;
  }
  fields.push(cur);
  return fields;
}

/** jiayuchann/targeted_extensions.csv — ExtensionID,ExtensionName */
function parseGistExtensions(csvText) {
  const out = {};
  const lines = String(csvText).trim().split(/\r?\n/);
  for (let i = 1; i < lines.length; i++) {
    const [rawId, rawName] = parseCsvLine(lines[i]);
    const eid = String(rawId || "").trim().toLowerCase();
    let name = String(rawName || "").trim().replace(/\s*\(removed\)\s*$/i, "");
    if (!/^[a-z]{32}$/.test(eid) || name.length < 2) continue;
    setName(out, eid, name, true);
  }
  return out;
}

/** Parse extension_ids() from JayGLXR/MacOS-Stealer-in-Rust browsers.rs */
function parseRustWalletExtensions(rustSource) {
  const out = {};
  const re = /"([a-z]{32})"\s*,?\s*(?:\/\/\s*([^\n]+))?/g;
  let m;
  while ((m = re.exec(rustSource)) !== null) {
    const eid = m[1];
    let name = cleanWalletLabel((m[2] || "").replace(/\s*\(.*$/, ""));
    if (!name) continue;
    setName(out, eid, name);
  }
  return out;
}

function ingestHyperHivesNames(data, nameById) {
  for (const val of Object.values(data.all_strings || {})) {
    const hit = extractNameFromConcatenated(val);
    if (hit) setName(nameById, hit.eid, hit.name);
  }
  for (const val of data.wallets || []) {
    const hit = extractNameFromConcatenated(val);
    if (hit) setName(nameById, hit.eid, hit.name);
  }
  for (const val of Object.values(data.other || {})) {
    const hit = extractNameFromConcatenated(val);
    if (hit) setName(nameById, hit.eid, hit.name);
  }
}

function isKnownWalletLabel(label) {
  return Boolean(label && label !== "Wallet" && !label.startsWith("Extension "));
}

const hyperUrl =
  "https://raw.githubusercontent.com/Darksp33d/hyperhives-macos-infostealer-analysis/main/output/full_decrypted_config.json";
const rustUrl =
  "https://raw.githubusercontent.com/JayGLXR/MacOS-Stealer-in-Rust/main/src/browsers.rs";
const gistUrl =
  "https://gist.githubusercontent.com/jiayuchann/ba3cd9f4f430a9351fdff75869959853/raw/targeted_extensions.csv";

const [data, rustSource, gistCsv] = await Promise.all([
  fetchJson(hyperUrl),
  fetchText(rustUrl),
  fetchText(gistUrl),
]);
const rustWallets = parseRustWalletExtensions(rustSource);
const gistWallets = parseGistExtensions(gistCsv);

const nameById = { ...rustWallets };
ingestHyperHivesNames(data, nameById);
for (const [eid, name] of Object.entries(NAMES)) {
  setName(nameById, eid, name, true);
}
for (const [eid, name] of Object.entries(gistWallets)) {
  if (!EXCLUDE.has(eid)) setName(nameById, eid, name, true);
}
for (const [eid, name] of Object.entries(EXTRA)) {
  setName(nameById, eid, name, true);
}

// Only keep extension IDs with a resolved wallet title (drop unknown/fallback IDs).
const final = {};
for (const [eid, label] of Object.entries(nameById)) {
  if (EXCLUDE.has(eid)) continue;
  if (!isKnownWalletLabel(label)) continue;
  final[eid] = label;
}

const lines = [
  "package scanner",
  "",
  "// knownWalletExtensions maps Chromium extension IDs to crypto wallet display names.",
  "// Used when manifest.json has no human-readable title; scan also reads manifest on disk.",
  "// Sources: jiayuchann targeted_extensions.csv gist, HyperHives config, MacOS-Stealer-in-Rust, curated overrides.",
  `var knownWalletExtensions = map[string]string{`,
];
for (const eid of Object.keys(final).sort()) {
  const name = final[eid].replace(/"/g, '\\"');
  lines.push(`\t"${eid}": "${name}",`);
}
lines.push("}", "");

const out = path.join(root, "recovery/scanner/wallet_extensions.go");
fs.writeFileSync(out, lines.join("\n"));
const skipped = Object.keys(nameById).filter(
  (eid) => !EXCLUDE.has(eid) && !isKnownWalletLabel(nameById[eid]),
).length;
console.log(
  `Wrote ${Object.keys(final).length} named wallet IDs (skipped ${skipped} unknown, gist ${Object.keys(gistWallets).length}) to ${out}`,
);