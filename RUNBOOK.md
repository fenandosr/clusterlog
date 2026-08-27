# RUNBOOK

Punto de entrada de la documentación ejecutable de `clusterlog`. Cada celda de
abajo envuelve un comando canónico existente (`make`, `./bin/clusterlog`, un
script en `scripts/`) — ninguna celda contiene lógica propia. Ver
[`docs/RUNME.md`](docs/RUNME.md) para el porqué y el modelo de identidad, y
[`runbooks/README.md`](runbooks/README.md) para la convención completa de
nombres y etiquetas.

Este archivo funciona igual con o sin Runme instalado: cada bloque es un
comando de shell normal que también puede copiarse y ejecutarse a mano.

## Runbooks disponibles

- [`runbooks/clusterlog-development.md`](runbooks/clusterlog-development.md) — ciclo de desarrollo (build, test, vet, sync, validate, sitio).
- [`runbooks/weekly-review.md`](runbooks/weekly-review.md) — versión ejecutable de la reunión semanal ([`docs/WEEKLY_REVIEW.md`](docs/WEEKLY_REVIEW.md) conserva la agenda).
- [`runbooks/_template.md`](runbooks/_template.md) — plantilla para runbooks nuevos.

## Comprobaciones rápidas

```sh {"name":"clusterlog-runme-preflight","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
set -euo pipefail
test -f go.mod && test -f Makefile || { echo "ejecute esto desde la raíz de clusterlog" >&2; exit 1; }
echo "go:      $(go version)"
echo "git:     $(git --version)"
if command -v zola >/dev/null 2>&1; then echo "zola:    $(zola --version)"; else echo "zola:    no instalado (validate --skip-zola sigue funcionando)"; fi
if command -v runme >/dev/null 2>&1; then echo "runme:   $(runme --version)"; else echo "runme:   no instalado (make runme-install)"; fi
```

```sh {"name":"clusterlog-build","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make build
```

```sh {"name":"clusterlog-test","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make test
```

```sh {"name":"clusterlog-sync-check","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make sync-check
```

```sh {"name":"clusterlog-validate-local","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make validate-local
```

```sh {"name":"clusterlog-site-build","tag":"role-developer,env-local,risk-read-only","interactive":"false"}
make site
```

`clusterlog-site-build` no lleva `ci-safe`: `make site` incluye `make theme`,
que clona el repositorio del tema por red. La CI principal
(`.gitea/workflows/ci.yml`) ya construye el sitio en su propio paso; repetirlo
aquí sería una segunda copia de la misma lógica, no una celda adicional útil.

```sh {"name":"clusterlog-review-mine","tag":"role-reviewer,env-local,risk-read-only","interactive":"false"}
: "${CLUSTERLOG_SSH_KEY:?defina CLUSTERLOG_SSH_KEY con la ruta a su llave pública}"
./bin/clusterlog --root . --json --ssh-key "$CLUSTERLOG_SSH_KEY" review list --mine
```

`clusterlog-review-mine` tampoco lleva `ci-safe`: `--mine` resuelve identidad
a partir de una llave SSH personal, y CI no tiene ni debe tener acceso a
ninguna llave privada real.

```sh {"name":"clusterlog-verify-head","tag":"role-reviewer,env-local,risk-read-only,ci-safe","interactive":"false"}
./bin/clusterlog --root . --json verify commit HEAD
```

## Ejecutar

Ejemplo (nota: éste es texto de referencia, no una celda ejecutable —
usa fence ` ```text ` a propósito para que Runme no lo liste como comando):

```text
runme run --filename RUNBOOK.md clusterlog-build
```

o, sin Runme instalado, copie y corra el comando dentro del bloque
directamente. `make runme-ci` ejecuta:

```text
runme run --filename RUNBOOK.md --tag ci-safe --all --skip-prompts
```

`runme run` sí acepta `--tag`/`--all` (verificado con el binario real de
`v3.17.4`: un tag inexistente devuelve `No tasks to execute with the tag
provided`, y `--tag ci-safe --all` corre exactamente las celdas marcadas
`ci-safe`, en orden, deteniéndose en el primer fallo). Como es un filtro
explícito por etiqueta de seguridad — no "correr todo" — este uso puntual sí
es seguro.

Nunca ejecute `runme run --all --skip-prompts` **sin** `--tag ci-safe` sobre
este archivo ni sobre `runbooks/weekly-review.md`: sin ese filtro, `--all`
también incluiría celdas que mutan estado real. Ésas deben correrse una por
una, a mano, después de revisar sus
prerrequisitos.
