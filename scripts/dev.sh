#!/usr/bin/env bash
# 并发启动前后端开发服务器
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT/backend" && go run ./cmd/pichub -config "$ROOT/config.dev.yaml" &
BACKEND_PID=$!

cd "$ROOT/frontend"
if [ ! -d node_modules ]; then
  pnpm install
fi
pnpm dev &
FRONTEND_PID=$!

trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null" EXIT
wait
