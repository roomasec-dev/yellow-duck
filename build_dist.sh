#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed or not in PATH"
  exit 1
fi

mkdir -p dist

echo "==> build darwin amd64"
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
  go build -o dist/rm-ai-agent-darwin-amd64 .

echo "==> build darwin arm64"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
  go build -o dist/rm-ai-agent-darwin-arm64 .

echo "==> build linux amd64"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o dist/rm-ai-agent-linux-amd64 .

echo "==> build windows amd64"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  go build -o dist/rm-ai-agent-windows-amd64.exe .

echo "==> done"
ls -lh dist/rm-ai-agent-darwin-amd64 dist/rm-ai-agent-darwin-arm64 dist/rm-ai-agent-linux-amd64 dist/rm-ai-agent-windows-amd64.exe
