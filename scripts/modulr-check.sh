#!/usr/bin/env bash
# Обязательная проверка перед коммитом (PROJECT_RULES.md §7).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== modulr (root module) — fmt, vet, build ./... =="
go fmt ./...
go vet ./...
go build ./...

echo "== gofmt: organizer, telegram, databases (legacy mixed main+lib) =="
for dir in organizer telegram databases; do
  if [[ -d "$ROOT/$dir" ]]; then
    while IFS= read -r -d '' f; do gofmt -w "$f"; done < <(find "$ROOT/$dir" -name '*.go' -print0)
  fi
done

echo "✅ modulr-check: OK (vet+build только корень modulr; подмодули — gofmt)"
