#!/usr/bin/env bash
# Обязательная проверка перед коммитом (PROJECT_RULES.md §7).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export GOCACHE="${GOCACHE:-/tmp/modulr-go-build-cache}"

echo "== modulr (root module) — fmt, vet, build ./... =="
go fmt ./...
go vet ./...
go build ./...

echo "== telegram submodule — go test ./... =="
if [[ -d "$ROOT/telegram" ]]; then
  (cd "$ROOT/telegram" && go test ./...)
fi

echo "== databases submodule — go test ./... =="
if [[ -d "$ROOT/databases" ]]; then
  (cd "$ROOT/databases" && go test ./...)
fi

echo "== gofmt: organizer (legacy mixed main+lib) =="
for dir in organizer; do
  if [[ -d "$ROOT/$dir" ]]; then
    while IFS= read -r -d '' f; do gofmt -w "$f"; done < <(find "$ROOT/$dir" -name '*.go' -print0)
  fi
done

echo "✅ modulr-check: OK (root vet+build; telegram/databases go test; organizer — gofmt)"
