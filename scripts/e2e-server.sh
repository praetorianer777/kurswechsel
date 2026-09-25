#!/usr/bin/env bash
# Started by Playwright: the real binary with the website embedded, serving
# fictional demo data from a throwaway database.
set -euo pipefail
cd "$(dirname "$0")/.."

port="${E2E_PORT:-8099}"
db="$(mktemp -d)/demo.db"
./scripts/build.sh >/dev/null
./backend/bin/kurswechsel seed-demo -db "$db" >/dev/null
exec ./backend/bin/kurswechsel serve -addr "127.0.0.1:$port" -db "$db"
