#!/usr/bin/env bash
# The one entry point for the whole suite: the branch-guard hook runs this
# before every push, and ci.yml has no other step.
set -euo pipefail
cd "$(dirname "$0")"

echo "🐚 Shell tests"
./.claude/hooks/tests/branch-guard-test.sh

for tool in go node npm; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "❌ $tool not found — see README 'Development' for the toolchain" >&2
    exit 1
  fi
done

echo "🎨 Go formatting"
unformatted="$(gofmt -l backend)"
if [[ -n "$unformatted" ]]; then
  echo "❌ gofmt would change:" >&2
  echo "$unformatted" >&2
  exit 1
fi

echo "🔍 Go static analysis"
(cd backend && go vet ./... && go tool staticcheck ./...)

echo "🧪 Go tests"
(cd backend && go test -race -count=1 ./...)

# A fresh checkout has no node_modules; npm ci keeps the lockfile authoritative
# so the gate never tests dependencies nobody committed.
if [[ ! -d frontend/node_modules ]]; then
  echo "📦 Frontend dependencies"
  (cd frontend && npm ci --no-audit --no-fund)
fi

echo "🎨 Frontend formatting"
(cd frontend && npx prettier --check . >/dev/null)

echo "🔍 Frontend lint and types"
(cd frontend && npx eslint --max-warnings=0 . && npx tsc -b)

echo "🧪 Frontend tests"
(cd frontend && npx vitest run)

echo "🏗️  Frontend build"
(cd frontend && npx vite build --logLevel warn)

echo "✅ All tests passed"
