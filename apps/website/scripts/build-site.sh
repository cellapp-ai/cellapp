#!/bin/bash
set -euo pipefail
TASK_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="$TASK_ROOT/node_modules/.bin:$PATH"
cd "$TASK_ROOT"
node scripts/embed-fonts.mjs
tsc -b
vite build --outDir .build/vite
parcel build index.html --dist-dir .build/parcel --no-source-maps
node scripts/bundle-site.mjs
node --import tsx scripts/export-static.tsx
