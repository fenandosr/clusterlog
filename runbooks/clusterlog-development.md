# Ciclo de desarrollo de clusterlog

Propósito: correr el mismo ciclo que `AGENTS.md` pide "antes de entregar",
como notebook en vez de copiar/pegar bloques sueltos. Todas las celdas de
este documento envuelven targets existentes de `Makefile`; ninguna reimplementa
lo que ya hace `go test`, `go vet` o `clusterlog`.

## Prerrequisitos

- Go 1.23+, Git con firmas SSH.
- Para las celdas con Zola: binario `zola` en `PATH` (opcional; las celdas de
  `validate`/`sync` no lo requieren).

## Formato

```sh {"name":"clusterlog-gofmt-check","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
test -z "$(gofmt -l cmd internal)"
```

## Pruebas y análisis estático

```sh {"name":"clusterlog-vet","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
go vet ./...
```

```sh {"name":"clusterlog-test-race","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
go test -race ./...
```

## Compilar y proyectar

```sh {"name":"clusterlog-dev-build","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make build
```

```sh {"name":"clusterlog-dev-sync","tag":"role-developer,env-local,risk-mutating,manual-only","interactive":"true","excludeFromRunAll":true}
# Escribe data/generated/site_state.json. Revise el diff antes de confirmar
# cualquier commit que incluya ese archivo regenerado.
make sync
git diff --stat -- data/generated/site_state.json
```

### Prerrequisitos de `clusterlog-dev-sync`

- Haber corrido `clusterlog-dev-build` en esta sesión (o tener `bin/clusterlog`
  ya compilado).

### Validación posterior

```sh {"name":"clusterlog-dev-sync-check","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make sync-check
```

### Rollback

`data/generated/site_state.json` es 100% regenerable desde el contenido
fuente (`content/`, `data/admins.json`, `data/*-events/`). Si `make sync`
produjo un diff no deseado, descarte el archivo con
`git checkout -- data/generated/site_state.json` y no lo incluya en el commit.

## Validar

```sh {"name":"clusterlog-dev-validate-local","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make validate-local
```

## Flujo completo local (equivalente a CI, sin Zola)

```sh {"name":"clusterlog-dev-smoke","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
make build
./bin/clusterlog --root . sync --check
./bin/clusterlog --root . validate --skip-zola
./scripts/smoke.sh
```

## Registro en Clusterlog

Ninguna celda de este runbook crea memorias, tareas ni commits. Si el
trabajo de la sesión amerita una memoria, hágalo con el flujo normal descrito
en `AGENTS.md` (`clusterlog memory create --from-json`, con `--dry-run` antes
de `--commit`) — fuera de Runme, con su propia llave.

## Limitaciones

- `clusterlog-dev-sync` está marcado `manual-only` a propósito: regenera un
  archivo versionado y su diff debe revisarse antes de confirmarlo.
- Este runbook no incluye `make site`/`zola build`: ver
  [`RUNBOOK.md`](../RUNBOOK.md#comprobaciones-rápidas) para por qué esa celda
  vive ahí y no se duplica aquí.
