#!/usr/bin/env bash
# 一键构建：前端 → 嵌入 → Go 编译 → fpk 打包
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "→ [1/4] 构建前端"
cd "$ROOT/frontend"
if [ ! -d node_modules ]; then
  pnpm install
fi
pnpm build

echo "→ [2/4] 同步前端到 backend/internal/server/assets/"
rm -rf "$ROOT/backend/internal/server/assets"
mkdir -p "$ROOT/backend/internal/server/assets"
cp -r "$ROOT/frontend/dist/"* "$ROOT/backend/internal/server/assets/"

echo "→ [3/4] 编译 Go 后端（linux/amd64）"
cd "$ROOT/backend"
mkdir -p "$ROOT/levis.pichub/app/bin"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$ROOT/levis.pichub/app/bin/pichub" ./cmd/pichub
cp "$ROOT/config.dev.yaml" "$ROOT/levis.pichub/app/config.default.yaml"

echo "→ [4/4] 打包 fpk"
cd "$ROOT"
"$ROOT/../fnpack.exe" build -d levis.pichub

echo "✓ 构建完成"
ls -lh "$ROOT/levis.pichub.fpk" 2>/dev/null || true
