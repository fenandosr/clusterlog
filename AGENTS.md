# Instrucciones para agentes de programación

## Propósito

Use `clusterlog` para registrar trabajo operativo de forma atribuible, validada y reproducible. No edite directamente archivos derivados ni simule una identidad.

## Reglas obligatorias

1. Lea `clusterlog --json schema <tipo>` antes de construir una entrada nueva.
2. No incluya secretos, tokens, contraseñas, llaves privadas ni datos personales innecesarios.
3. Registre sólo hechos observados. Distinga claramente entre cambio ejecutado, validación ejecutada y recomendación pendiente.
4. Ejecute primero con `--dry-run` cuando el contenido provenga de una sesión larga o incluya comandos.
5. Use siempre `--json` y compruebe el código de salida.
6. Para una escritura final, use `--commit` únicamente cuando el usuario haya autorizado la operación y su mecanismo de firma esté disponible.
7. Nunca exporte, lea ni copie la llave privada. Pase una ruta autorizada o permita que Git/ssh-agent la maneje.
8. No edite `data/generated/site_state.json`; ejecute `clusterlog sync`.
9. No agregue manualmente eventos a un arreglo central. Cada evento debe ocupar su archivo independiente y ser creado por la CLI.
10. Después de escribir, ejecute `clusterlog validate` y devuelva al usuario ID, ruta, advertencias y SHA del commit.

## Flujo para una memoria de sesión

```bash
clusterlog --json schema memory > /tmp/memory-schema-envelope.json

cat > /tmp/memory.json <<'JSON'
{
  "title": "...",
  "description": "...",
  "tags": [],
  "systems": [],
  "risk": "low",
  "review_required": true,
  "context": "...",
  "changes": [],
  "commands": [],
  "validation": [],
  "rollback": "...",
  "agent": "<nombre-del-agente>"
}
JSON

clusterlog --root . --json --dry-run --ssh-key "$CLUSTERLOG_SSH_KEY" \
  memory create --from-json /tmp/memory.json

clusterlog --root . --json --commit --ssh-key "$CLUSTERLOG_SSH_KEY" \
  memory create --from-json /tmp/memory.json

clusterlog --root . --json validate
clusterlog --root . --json verify commit HEAD
```

## Calidad de una memoria

Una memoria útil responde:

- ¿Cuál era el contexto o síntoma?
- ¿Qué se cambió exactamente?
- ¿Qué comandos son relevantes, con valores sensibles redactados?
- ¿Qué resultado se observó?
- ¿Cómo se validó?
- ¿Qué riesgo quedó abierto?
- ¿Cómo se revierte?
- ¿Qué manual o documento debe actualizarse?

No convierta salidas extensas de terminal en el cuerpo. Resuma y enlace evidencia aprobada.

## Comportamiento ante errores

- `identity_not_found`: deténgase; no elija otra identidad por aproximación.
- `possible_secret`: redacte y repita el dry-run; no fuerce la escritura.
- `validation_failed`: presente los códigos y rutas concretas.
- `git_commit_failed`: no afirme que la operación quedó auditada; los archivos pueden existir sin commit y deben revisarse.
- `task_has_running_timers`: identifique a los actores; no cierre o borre eventos.

## Runbooks ejecutables (Runme)

`RUNBOOK.md` y `runbooks/**/*.md` son documentación ejecutable, no una
segunda CLI. Reglas para un agente:

- Nunca active una celda `risk-mutating` o `risk-destructive` sin que el
  usuario haya autorizado explícitamente esa operación puntual — la
  etiqueta `manual-only`/`interactive`/`excludeFromRunAll` es una defensa
  técnica de Runme, no una autorización.
- Trate cada celda como el comando que envuelve: si el comando requiere
  `--dry-run` antes de `--commit`, la celda también.
- No agregue lógica nueva dentro de una celda; si hace falta lógica
  reutilizable, colóquela en `clusterlog`, un script en `scripts/` o un
  target de `Makefile`, y haga que la celda sólo lo invoque.
- No genere ni edite manualmente un identificador Lifecycle Identity de
  Runme.

## Cambios al código del proyecto

Antes de entregar:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
make build
./bin/clusterlog --root . sync
./bin/clusterlog --root . sync --check
./bin/clusterlog --root . validate --skip-zola
```

Cuando Zola esté disponible:

```bash
./scripts/bootstrap-theme.sh
./bin/clusterlog --root . validate --require-zola
zola build
```

Mantenga compatibilidad del JSON. Cualquier cambio de contrato debe actualizar `schema.go`, `CLI.md`, ejemplos y pruebas.
