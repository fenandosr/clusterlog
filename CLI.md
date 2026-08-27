# Referencia de `clusterlog`

## 1. Forma general

```text
clusterlog [opciones globales] <comando> [opciones del comando]
```

Las opciones globales deben colocarse **antes** del comando.

| Opción | Efecto |
|---|---|
| `--root DIR` | raíz del repositorio; si se omite, la CLI busca el proyecto desde el directorio actual |
| `--json` | salida estructurada y estable para agentes |
| `--ssh-key PATH` | llave pública o ruta base usada para resolver identidad y firma |
| `--commit` | crea un commit Git SSH firmado con sólo los archivos de la operación |
| `--dry-run` | valida y calcula el resultado sin escribir |

También puede definirse `CLUSTERLOG_SSH_KEY`.

## 2. Contrato JSON

Éxito:

```json
{
  "ok": true,
  "command": "memory.create",
  "data": {},
  "warnings": []
}
```

Error:

```json
{
  "ok": false,
  "error": {
    "kind": "identity_not_found",
    "message": "ninguna llave SSH local coincide con un administrador activo",
    "details": {}
  }
}
```

Un agente debe usar el código del proceso como decisión primaria y `error.kind` para tratamiento específico. No debe inferir éxito por texto parcial.

## 3. Códigos de salida

| Código | Significado |
|---:|---|
| `0` | éxito |
| `2` | uso, flags o entrada inválida |
| `3` | identidad no encontrada o no autorizada |
| `4` | validación, firma o contenido inválido |
| `5` | conflicto de estado |
| `6` | operación Git fallida |
| `10` | error interno no clasificado |

## 4. Identidad

```bash
clusterlog --root . --json --ssh-key ~/.ssh/id_ed25519.pub identity
```

La respuesta incluye el administrador, la huella, la ruta pública y la ruta que Git intentará usar para firmar. No prueba posesión hasta que exista una firma válida.

## 5. Administradores

### Agregar

```bash
clusterlog --root . --ssh-key ~/.ssh/id_ed25519.pub admin add \
  --id ana \
  --name "Ana Torres" \
  --email ana@example.org \
  --key ~/.ssh/id_ed25519.pub \
  --roles owner,admin
```

Reglas:

- El primer administrador obtiene `owner` aunque no se incluya explícitamente.
- Después del primero, sólo un `owner` puede agregar administradores.
- No se aceptan IDs, correos o huellas duplicados.
- `data/admins.json` y `config/allowed_signers` se actualizan juntos.

### Listar

```bash
clusterlog --root . --json admin list
clusterlog --root . --json admin list --active
```

## 6. Memorias

### Consultar esquema

```bash
clusterlog --json schema memory
```

### Crear por JSON

```bash
cat examples/memory.json | clusterlog \
  --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  memory create --from-json -
```

### Crear por flags

```bash
clusterlog --root . --commit --ssh-key ~/.ssh/id_ed25519.pub memory create \
  --title "Rotar certificado del API" \
  --description "Renovación programada y comprobación de clientes" \
  --tags tls,certificados \
  --systems kube-apiserver \
  --risk medium \
  --ticket CHG-1042 \
  --duration 45m \
  --agent claude-code \
  --body-file memoria.md
```

Opciones específicas:

- `--title`
- `--description`
- `--tags`
- `--systems`
- `--risk low|medium|high|critical`
- `--ticket`
- `--review` (por defecto `true`)
- `--review-policy all-active|all-active-except-author`
- `--duration` en minutos, `90m` o `2h`
- `--agent`
- `--body-file ARCHIVO|-`
- `--from-json ARCHIVO|-`

Cuando se combinan JSON y flags, los flags que se proporcionen explícitamente tienen precedencia.

## 7. Revisión

### Listar pendientes

```bash
clusterlog --root . --json review list
clusterlog --root . --json --ssh-key ~/.ssh/id_ed25519.pub review list --mine
```

`--mine` requiere identidad y devuelve sólo memorias cuyo conjunto pendiente contiene al administrador actual.

### Marcar la versión actual

```bash
clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  review mark MEM-20260812-... --note "Revisado; sin objeciones"
```

El evento guarda el SHA-256 del contenido actual. Repetir el comando sobre el mismo hash es idempotente; editar la memoria hace que el evento anterior deje de contar.

## 8. Tareas

### Crear

```bash
cat examples/task.json | clusterlog \
  --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  task create --from-json -
```

Flags disponibles:

- `--title`, `--description`
- `--project PRJ-...`
- `--priority low|medium|high|urgent`
- `--assignees ana,luis`
- `--estimate 120m`
- `--due YYYY-MM-DD`
- `--tags`, `--systems`
- `--objective`
- `--acceptance` separado por comas
- `--notes`
- `--body-file ARCHIVO|-`
- `--from-json ARCHIVO|-`

### Iniciar y detener tiempo

```bash
clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  task start TSK-... --note "Diagnóstico inicial"

clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  task stop TSK-... --note "Prueba completada"
```

Un administrador sólo puede mantener un temporizador activo. Los eventos se ordenan por fecha y tipo para producir un total reproducible.

### Ajustar tiempo

```bash
clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  task adjust TSK-... --minutes 20 --note "Trabajo previo no registrado"

clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  task adjust TSK-... --minutes -10 --note "Corregir doble conteo"
```

El total efectivo nunca puede quedar negativo.

### Cambiar estado

```bash
clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  task status TSK-... --status blocked --note "Esperando refacción"
```

Estados admitidos aquí: `backlog`, `todo`, `in_progress`, `blocked`, `cancelled`. Para finalizar use `task done`.

### Completar

```bash
clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  task done TSK-... --note "Criterios de aceptación verificados"
```

Si el administrador actual mantiene el temporizador, la CLI lo detiene. Rechaza completar si otra persona sigue registrando tiempo en la misma tarea.

### Listar

```bash
clusterlog --root . --json task list
clusterlog --root . --json task list --status blocked
clusterlog --root . --json --ssh-key ~/.ssh/id_ed25519.pub task list --mine
clusterlog --root . --json task list --project PRJ-...
```

## 9. Proyectos

### Crear

```bash
cat examples/project.json | clusterlog \
  --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  project create --from-json -
```

Flags:

- `--title`, `--description`, `--objective`
- `--owners`
- `--target-date YYYY-MM-DD`
- `--status planned|active|blocked|done|cancelled|archived`
- `--tags`, `--systems`
- `--scope`, `--out-of-scope`
- `--milestones`, `--success`
- `--body-file ARCHIVO|-`
- `--from-json ARCHIVO|-`

### Listar

```bash
clusterlog --root . --json project list
clusterlog --root . --json project list --status active
```

## 10. Documentación y manuales

```bash
cat examples/manual.json | clusterlog \
  --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  content create --from-json -
```

También acepta:

```bash
clusterlog --root . --commit --ssh-key ~/.ssh/id_ed25519.pub content create \
  --section manuales \
  --title "Reemplazar nodo de cómputo" \
  --description "Procedimiento seguro y validaciones" \
  --tags runbook,hardware \
  --systems compute \
  --body-file reemplazo.md \
  --weight 20
```

`section` sólo puede ser `documentacion` o `manuales`.

## 11. Proyección

```bash
clusterlog --root . sync
clusterlog --root . sync --check
```

`sync` lee Markdown, administradores y eventos; genera `data/generated/site_state.json`. `--check` no escribe y falla cuando el archivo derivado no coincide con la fuente.

Los agentes no deben editar `data/generated/site_state.json` a mano.

## 12. Validación

```bash
clusterlog --root . --json validate
clusterlog --root . --json validate --skip-zola
clusterlog --root . --json validate --require-zola
```

Se comprueban, entre otros:

- registro de administradores y `allowed_signers`;
- IDs, autores, revisores, owners y assignees;
- estados, fechas y referencias de proyectos;
- eventos huérfanos o inconsistentes;
- temporizadores concurrentes y totales negativos;
- secretos probables;
- vigencia de la proyección;
- plantillas y enlaces con `zola check`, cuando está disponible.

## 13. Firmas

```bash
clusterlog --root . verify commit HEAD
clusterlog --root . verify commit abc1234
```

La verificación usa:

```text
config/allowed_signers
namespace: git
```

En CI conviene verificar todos los commits nuevos, no sólo `HEAD`.

## 14. Esquemas para agentes

```bash
clusterlog --json schema
clusterlog --json schema memory
clusterlog --json schema task
clusterlog --json schema project
clusterlog --json schema content
```

Los esquemas siguen JSON Schema 2020-12 y rechazan propiedades adicionales. Esta restricción permite detectar errores ortográficos en nombres de campos.

## 15. Patrón recomendado para agentes

```bash
set -euo pipefail

SCHEMA="$(clusterlog --json schema memory)"
# Construir input.json con base en data del sobre.

clusterlog --root . --json --dry-run --ssh-key "$KEY" \
  memory create --from-json input.json

RESULT="$(clusterlog --root . --json --commit --ssh-key "$KEY" \
  memory create --from-json input.json)"

printf '%s\n' "$RESULT"
clusterlog --root . --json verify commit HEAD
```

El agente debe detenerse ante cualquier código distinto de cero y presentar el error al usuario sin ocultarlo.

## 16. Versión

```bash
clusterlog version
```

No requiere `--root` ni identidad. Devuelve `{"name":"clusterlog","version":"..."}`.
El valor de `version` se fija en tiempo de compilación:

```bash
go build -ldflags "-X example.org/bitacora-cluster/internal/clusterlog.Version=v0.2.0" ...
```

`make build` ya lo hace automáticamente a partir de `git describe --tags --always --dirty` (variable `CLUSTERLOG_VERSION` del `Makefile`); un build local sin tags produce algo como `abc1234-dirty`, no `dev` ni una cadena vacía.

## 17. Instancias nuevas

```bash
clusterlog bootstrap-instance \
  --output ./clusterlog-nuevo-deploy \
  --product-name "Bitácora de Ejemplo" \
  --base-url "https://clusterlog.ejemplo.org"
```

No requiere `--root`: `--output` es un directorio nuevo, normalmente fuera del repositorio actual. Tampoco usa `--commit` (no hay nada que firmar todavía). Genera:

- `zola.toml` con `base_url`/`title`/`[extra] product_name` sustituidos y el resto de los valores genéricos del motor;
- `content/<sección>/_index.md` para `documentacion`, `manuales`, `memorias`, `proyectos`, `tareas` y `revision`;
- `data/admins.json` y `config/allowed_signers` vacíos (mismo formato que produce `admin add`, sin ningún administrador todavía);
- `data/review-events/.gitkeep`, `data/task-events/.gitkeep`, `data/generated/`;
- `.clusterlog-version` con la versión del binario que generó el scaffold;
- `README.md` y `.gitignore` de arranque.

Falla con `output_not_empty` (código `5`) si `--output` ya existe y no está vacío — nunca sobreescribe una instancia existente. Con `--dry-run` no crea el directorio; sólo muestra qué haría.

Después de `bootstrap-instance`, el flujo es el mismo que "Registrar al primer administrador" en `README.md`, apuntando `--root` al directorio nuevo: `admin add` → `sync` → `validate --skip-zola`. `validate` antes de la primera `sync` falla con `generated_state_stale` a propósito: la proyección todavía no existe.

**No incluido todavía** (fases posteriores, ver el ADR de motor/instancia cuando exista): plantillas Tera ni tema empaquetados para descarga, ni un `.gitea/workflows/ci.yml` de instancia — el motor aún no publica releases binarios.
