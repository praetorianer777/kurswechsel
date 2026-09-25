#!/usr/bin/env bash
# Builds the website into the Go binary: the frontend goes into
# backend/internal/web/dist, which the binary embeds.
set -euo pipefail
cd "$(dirname "$0")/.."

(cd frontend && npx vite build --logLevel warn --outDir ../backend/internal/web/dist --emptyOutDir)
# --emptyOutDir removes the placeholder git keeps the directory with.
touch backend/internal/web/dist/.keep
mkdir -p backend/bin
(cd backend && go build -o bin/kurswechsel ./cmd/kurswechsel)
echo "built backend/bin/kurswechsel"
