#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI="${CLUSTERLOG_BIN:-$ROOT/bin/clusterlog}"
BASE_SHA="${BASE_SHA:-}"
HEAD_SHA="${HEAD_SHA:-HEAD}"

for command in git jq; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Falta el comando requerido para verificar firmas: $command" >&2
    exit 1
  fi
done
if [[ ! -x "$CLI" ]]; then
  echo "No existe un binario ejecutable en $CLI; ejecute make build." >&2
  exit 1
fi

active_admins=$(jq '[.admins[] | select(.active == true)] | length' "$ROOT/data/admins.json")
if [[ "$active_admins" -eq 0 ]]; then
  echo "Firmas: bootstrap sin administradores activos; comprobación omitida."
  exit 0
fi

if [[ -z "$BASE_SHA" ]]; then
  echo "BASE_SHA es obligatorio cuando existen administradores activos." >&2
  exit 1
fi
if ! git -C "$ROOT" cat-file -e "$BASE_SHA^{commit}" 2>/dev/null; then
  echo "El commit base no está disponible: $BASE_SHA" >&2
  exit 1
fi
if ! git -C "$ROOT" cat-file -e "$HEAD_SHA^{commit}" 2>/dev/null; then
  echo "El commit de cabeza no está disponible: $HEAD_SHA" >&2
  exit 1
fi

mapfile -t commits < <(git -C "$ROOT" rev-list --reverse "$BASE_SHA..$HEAD_SHA")
if [[ ${#commits[@]} -eq 0 ]]; then
  echo "Firmas: no hay commits nuevos en el rango."
  exit 0
fi

for commit in "${commits[@]}"; do
  "$CLI" --root "$ROOT" verify commit "$commit" >/dev/null
  echo "Firma SSH autorizada: $commit"
done
