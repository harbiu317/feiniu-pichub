#!/usr/bin/env bash
# 仅调用 fnpack 打包（假设 app/bin/pichub 已就位）
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FNPACK="$ROOT/../fnpack.exe"

# 自动递增 patch 版本
if [ "${BUMP:-0}" = "1" ]; then
  current=$(grep '^version=' "$ROOT/levis.pichub/manifest" | cut -d= -f2)
  IFS='.' read -r major minor patch <<< "$current"
  patch=$((patch + 1))
  VERSION="$major.$minor.$patch"
  echo "→ 版本号自动递增到 $VERSION"
  sed -i "s/^version=.*/version=$VERSION/" "$ROOT/levis.pichub/manifest"
fi

cd "$ROOT"
"$FNPACK" build -d levis.pichub
echo "✓ 打包完成：$ROOT/levis.pichub.fpk"
