#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI="${CLUSTERLOG_BIN:-$ROOT/bin/clusterlog}"

for command in jq mktemp cp find; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Falta el comando requerido para la prueba E2E: $command" >&2
    exit 1
  fi
done
if [[ ! -x "$CLI" ]]; then
  echo "No existe un binario ejecutable en $CLI; ejecute make build." >&2
  exit 1
fi

TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT
cp -a "$ROOT/." "$TMP/"
CLI="$TMP/bin/clusterlog"

for section in memorias tareas proyectos manuales documentacion; do
  find "$TMP/content/$section" -maxdepth 1 -type f -name '*.md' ! -name '_index.md' -delete
done
find "$TMP/data/review-events" -type f ! -name '.gitkeep' -delete
find "$TMP/data/task-events" -type f ! -name '.gitkeep' -delete
cat > "$TMP/data/admins.json" <<'JSON'
{
  "version": 1,
  "review_policy": {"default": "all-active", "include_author": true},
  "admins": []
}
JSON
: > "$TMP/config/allowed_signers"

# Material público ficticio: sólo ejercita resolución de huella, nunca firma.
cat > "$TMP/ana.pub" <<'KEY'
ssh-ed25519 ZmFrZS1rZXktbWF0ZXJpYWwtYW5hLTAwMDAwMDAwMDAwMDAwMA== ana@test
KEY
cat > "$TMP/bob.pub" <<'KEY'
ssh-ed25519 ZmFrZS1rZXktbWF0ZXJpYWwtYm9iLTAwMDAwMDAwMDAwMDAwMA== bob@test
KEY

run_json() {
  local output
  output=$("$@")
  jq -e '.ok == true' <<<"$output" >/dev/null
  printf '%s' "$output"
}

run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/ana.pub" \
  admin add --id ana --name 'Ana Admin' --email ana@example.org \
  --key "$TMP/ana.pub" --roles admin >/dev/null
run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/ana.pub" \
  admin add --id bob --name 'Bob Admin' --email bob@example.org \
  --key "$TMP/bob.pub" --roles admin >/dev/null

cat > "$TMP/memory.json" <<'JSON'
{
  "title": "Actualizar kubelet de nodos de prueba",
  "description": "Cambio controlado en dos nodos",
  "tags": ["kubernetes", "mantenimiento"],
  "systems": ["worker-test-01", "worker-test-02"],
  "risk": "medium",
  "review_required": true,
  "review_policy": "all-active",
  "context": "Validar la actualización antes de la ventana productiva.",
  "changes": ["Drenar nodos", "Actualizar paquete", "Reintegrar nodos"],
  "validation": ["Nodos Ready", "Pods reprogramados"],
  "rollback": "Reinstalar la versión anterior."
}
JSON
memory=$(run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/ana.pub" \
  memory create --from-json "$TMP/memory.json")
mem_id=$(jq -er '.data.entry_id' <<<"$memory")
mem_path=$(jq -er '.data.path' <<<"$memory")

queue=$(run_json "$CLI" --root "$TMP" --json review list)
[[ $(jq -er '.data.count' <<<"$queue") -eq 1 ]]
[[ $(jq -er '.data.items[0].required_count' <<<"$queue") -eq 2 ]]
[[ $(jq -er '.data.items[0].reviewed_count' <<<"$queue") -eq 0 ]]

run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/ana.pub" \
  review mark "$mem_id" --note 'Revisado por Ana' >/dev/null
queue=$(run_json "$CLI" --root "$TMP" --json review list)
[[ $(jq -er '.data.items[0].reviewed_count' <<<"$queue") -eq 1 ]]
run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/bob.pub" \
  review mark "$mem_id" --note 'Revisado por Bob' >/dev/null
queue=$(run_json "$CLI" --root "$TMP" --json review list)
[[ $(jq -er '.data.count' <<<"$queue") -eq 0 ]]

# Cambiar el contenido invalida las confirmaciones de la versión anterior.
printf '\n## Seguimiento\n\nSe agregó una observación posterior.\n' >> "$mem_path"
run_json "$CLI" --root "$TMP" --json sync >/dev/null
queue=$(run_json "$CLI" --root "$TMP" --json review list)
[[ $(jq -er '.data.count' <<<"$queue") -eq 1 ]]
[[ $(jq -er '.data.items[0].reviewed_count' <<<"$queue") -eq 0 ]]
run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/ana.pub" review mark "$mem_id" >/dev/null
run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/bob.pub" review mark "$mem_id" >/dev/null
queue=$(run_json "$CLI" --root "$TMP" --json review list)
[[ $(jq -er '.data.count' <<<"$queue") -eq 0 ]]

cat > "$TMP/project.json" <<'JSON'
{
  "title": "Renovar plataforma de monitoreo",
  "description": "Actualizar y normalizar la observabilidad",
  "objective": "Reducir puntos ciegos operativos.",
  "owners": ["ana"],
  "target_date": "2099-12-01",
  "status": "active",
  "tags": ["observabilidad"],
  "systems": ["prometheus"],
  "scope": ["Métricas y alertas"],
  "out_of_scope": ["Mesa de ayuda"],
  "milestones": ["Inventario", "Migración"],
  "success_criteria": ["Cobertura completa de nodos"]
}
JSON
project=$(run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/ana.pub" \
  project create --from-json "$TMP/project.json")
project_id=$(jq -er '.data.project_id' <<<"$project")

cat > "$TMP/task.json" <<JSON
{
  "title": "Migrar alertas de nodos",
  "description": "Mover reglas al repositorio estándar",
  "project_id": "$project_id",
  "priority": "high",
  "assignees": ["bob"],
  "estimate_minutes": 60,
  "due": "2099-11-01",
  "tags": ["alertas"],
  "systems": ["prometheus"],
  "objective": "Centralizar reglas.",
  "acceptance_criteria": ["Reglas validadas"],
  "notes": "Sin cambios productivos."
}
JSON
task=$(run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/bob.pub" \
  task create --from-json "$TMP/task.json")
task_id=$(jq -er '.data.task_id' <<<"$task")
run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/bob.pub" \
  task start "$task_id" --note 'Inicio' >/dev/null
run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/bob.pub" \
  task adjust "$task_id" --minutes 15 --note 'Trabajo previo' >/dev/null
done=$(run_json "$CLI" --root "$TMP" --json --ssh-key "$TMP/bob.pub" \
  task done "$task_id" --note 'Terminado')
[[ $(jq -er '.data.task.status' <<<"$done") == done ]]
[[ $(jq -er '.data.task.actual_minutes' <<<"$done") -ge 15 ]]
[[ $(jq -er '.data.task.running_by | length' <<<"$done") -eq 0 ]]

tasks=$(run_json "$CLI" --root "$TMP" --json task list --project "$project_id")
[[ $(jq -er '.data.count' <<<"$tasks") -eq 1 ]]
[[ $(jq -er '.data.items[0].status' <<<"$tasks") == done ]]
projects=$(run_json "$CLI" --root "$TMP" --json project list --status active)
[[ $(jq -er '.data.count' <<<"$projects") -eq 1 ]]
run_json "$CLI" --root "$TMP" --json sync --check >/dev/null
validation=$(run_json "$CLI" --root "$TMP" --json validate --skip-zola)
[[ $(jq -er '.data.valid' <<<"$validation") == true ]]
[[ $(find "$TMP/data/review-events" -type f -name '*.json' | wc -l) -eq 4 ]]
[[ $(find "$TMP/data/task-events" -type f -name '*.json' | wc -l) -ge 4 ]]

printf 'E2E OK: memoria %s, proyecto %s, tarea %s; revisión por hash y eventos independientes verificados.\n' \
  "$mem_id" "$project_id" "$task_id"
