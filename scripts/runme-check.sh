#!/usr/bin/env bash
# Comprobación de compatibilidad de los runbooks. No escribe nada en el
# repositorio. Las reglas duras (nombres únicos, etiquetas, aislamiento de
# celdas mutantes, secretos, referencias a scripts/targets inexistentes)
# viven en `clusterlog validate` (internal/clusterlog/runbooks.go) y corren
# siempre, con o sin Runme instalado. Este script agrega, por encima, un
# chequeo blando de que Runme puede parsear cada runbook de verdad.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${CLUSTERLOG_BIN:-$ROOT/bin/clusterlog}"

echo "== Reglas de clusterlog validate (obligatorio) =="
"$BIN" --root "$ROOT" --json validate --skip-zola

if ! command -v runme >/dev/null 2>&1; then
  echo "== runme no está instalado: se omite el chequeo de compatibilidad de Runme (make runme-install) =="
  exit 0
fi

echo "== Compatibilidad con la versión fijada de Runme (blando) =="
status=0
files=("$ROOT/RUNBOOK.md")
while IFS= read -r -d '' f; do
  files+=("$f")
done < <(find "$ROOT/runbooks" -name '*.md' -print0 2>/dev/null)

for f in "${files[@]}"; do
  rel="${f#"$ROOT"/}"
  if runme ls --filename "$f" >/dev/null 2>&1; then
    echo "  ok: $rel"
  else
    echo "  ADVERTENCIA: runme no pudo parsear $rel" >&2
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  echo "runme-check: advertencias de compatibilidad de Runme (no bloqueante)." >&2
fi
exit 0
