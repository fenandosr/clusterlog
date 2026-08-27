#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${CLUSTERLOG_BIN:-$ROOT/bin/clusterlog}"

"$BIN" --root "$ROOT" --json sync >/dev/null
"$BIN" --root "$ROOT" --json sync --check >/dev/null
"$BIN" --root "$ROOT" --json validate --skip-zola >/dev/null
"$BIN" --json schema memory >/dev/null
"$BIN" --json schema task >/dev/null
"$BIN" --json schema project >/dev/null
"$BIN" --json schema content >/dev/null

echo "Smoke test correcto."
