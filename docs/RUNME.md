# Runme como sidecar ejecutable

Este documento explica cómo y por qué `clusterlog` usa Runme, qué garantiza y
qué no. La decisión formal está en
[`docs/adr/0004-runme-sidecar.md`](adr/0004-runme-sidecar.md); esto es la
guía operativa del día a día.

## Qué es "sidecar" aquí

No es un contenedor sidecar de Kubernetes ni un servicio adicional. Es una
capa opcional y adyacente: Runme abre `RUNBOOK.md` y `runbooks/**/*.md` como
notebooks y ejecuta las celdas, pero cada celda sólo invoca `make`,
`./bin/clusterlog` o un script existente en `scripts/`. Si Runme no está
instalado, todo sigue funcionando exactamente igual copiando el comando de la
celda a una terminal — no hay una segunda implementación de la lógica en
ningún lado.

## Instalación

Versión fijada: **`v3.17.4`** (nunca `latest`).

```bash
make runme-install
```

Esto corre [`scripts/runme-install.sh`](../scripts/runme-install.sh), que:

- descarga el binario de `github.com/runmedev/runme` para su combinación de
  SO/arquitectura (Linux x86_64/arm64, macOS x86_64/arm64);
- verifica su SHA-256 contra los valores publicados en el `checksums.txt` del
  release (verificados de forma independiente con `sha256sum` durante esta
  implementación, no copiados a ciegas del sitio de descargas);
- instala en `$HOME/.local/bin` (o `$RUNME_INSTALL_DIR`), nunca sobre
  `bin/clusterlog` — el script aborta si detecta un binario `clusterlog` ahí;
- no usa `curl | sh`.

**Windows**: el script no lo soporta. Descargue manualmente
`runme_windows_{x86_64,arm64}.zip` desde la página de releases y verifique su
checksum contra el mismo `checksums.txt` antes de usarlo.

`make runme-install` no es un paso obligatorio de ningún flujo existente:
`make build`, `make test`, `clusterlog sync`/`validate` y `zola build` no lo
requieren ni lo disparan.

## Cómo correrlo

```bash
make runme-list    # runme ls --filename RUNBOOK.md
make runme-check   # clusterlog validate --skip-zola + compatibilidad blanda con Runme
make runme-ci      # runme run --filename RUNBOOK.md --tag ci-safe --all --skip-prompts
```

`runme-check` nunca escribe en el repositorio. `runme-ci` sólo corre celdas
etiquetadas `ci-safe`.

## El flag `--tag` es real (corrección respecto al diseño inicial)

Durante la investigación previa a implementar, la documentación pública de
`docs.runme.dev` que se pudo consultar no mostraba un flag de filtrado por
etiqueta en `runme run`, sólo `--filter` (regex por nombre). El plan inicial
asumió por eso que la "seguridad de CI" tendría que vivir como una lista de
nombres explícita en el `Makefile`.

Al instalar el binario real de `v3.17.4` y correr `runme run --help`, apareció
`-t, --tag stringArray` ("Run from a specific tag"). Se verificó
empíricamente:

```console
$ runme run --tag totally-bogus-tag-xyz --filename RUNBOOK.md --dry-run
could not execute command: No tasks to execute with the tag provided

$ runme run --tag ci-safe --all --skip-prompts --filename RUNBOOK.md
 ►  Running task clusterlog-runme-preflight...
 ...
 ►  ✓ Task clusterlog-runme-preflight exited with code 0
 ►  Running task clusterlog-build...
 ...
```

Es decir: `--tag` sí filtra de verdad, y `--tag ci-safe --all` corre
exactamente y sólo las celdas marcadas `ci-safe`, en el orden en que aparecen,
deteniéndose en el primer fallo. `make runme-ci` usa esa forma. La lección
que sí se mantiene del plan inicial: **nunca** ejecute `runme run --all
--skip-prompts` sin `--tag ci-safe` — sin el filtro, `--all` corre también
las celdas que mutan estado.

`runme ls --json` **no** expone las etiquetas de una celda en su salida (se
verificó con `--json` contra `RUNBOOK.md`: el JSON trae `name`, `file`,
`first_command`, `description`, `named`, `run_all`, pero no `tags`). Por eso
la fuente de verdad para "qué es `ci-safe`" es el atributo `tag` en el propio
Markdown, exigido y verificado por `clusterlog validate`
(`internal/clusterlog/runbooks.go`), no algo que se pueda inspeccionar desde
`runme ls`.

## `excludeFromRunAll` es la defensa real, no la etiqueta

`runme ls --json` también reveló que **toda** celda es elegible para
`--all` (`"run_all": true`) salvo que declare explícitamente
`"excludeFromRunAll": true`. Una celda `risk-mutating`/`risk-destructive` sin
ese atributo seguiría corriendo si alguien ejecuta `runme run --all` sin
filtro. Por eso `clusterlog validate` exige `excludeFromRunAll:true` +
`interactive:true` + la etiqueta `manual-only` en toda celda
`risk-mutating`/`risk-destructive` — la etiqueta sola no protege nada; el
atributo que Runme sí respeta es `excludeFromRunAll`.

## Capas de identidad (no confundirlas)

### Capa 1 — Identidad de contenido (Runme)

El atributo `name` de una celda y cualquier Lifecycle Identity que Runme
asigne identifican **el documento y la celda**, para poder seguir sus
cambios. No identifican quién la ejecutó ni prueban que se ejecutó. No edite
a mano un identificador Lifecycle Identity generado por Runme.

### Capa 2 — Evidencia local de sesión (Session Outputs)

Runme puede, si se habilita, guardar una copia del documento con la salida de
cada celda (hora, duración, código de salida, hostname, salida). Esta
integración **no habilita esa función**. Si en el futuro alguien la activa
localmente:

- debe quedar fuera de Git, del sitio Zola y de cualquier Gist;
- es evidencia auxiliar de que algo se ejecutó en una máquina, nunca prueba
  de autoría ni sustituto de una memoria en `clusterlog`.

**Limitación documentada**: la documentación pública de Runme
(`docs.runme.dev/usage/auto-save/`) no fija un nombre de archivo exacto para
estos Session Outputs en `v3.17.4` — sólo dice "genera un archivo, agréguelo
a `.gitignore`". Por eso este repositorio **no** agrega un patrón amplio de
`.gitignore` que podría ocultar Markdown legítimo sin darse cuenta.
`clusterlog validate` sí revisa (best-effort, no exhaustivo):

- que no exista un directorio `.runme/` versionado en el árbol de trabajo;
- que ningún archivo dentro de `runbooks/` tenga "output" en el nombre sin
  ser `_template.md` (señal, no prueba, de que podría ser una captura de
  sesión y no contenido fuente).

La defensa real sigue siendo procedimental: revise el diff antes de hacer
`git add`, igual que con cualquier otro archivo generado.

### Capa 3 — Auditoría canónica de Clusterlog (sin cambios)

La atribución fuerte sigue siendo exactamente la de siempre:

```text
huella de llave pública SSH
        +
identidad registrada en data/admins.json
        +
commit Git firmado por SSH
        +
verificación en CI (clusterlog verify commit)
```

Runme **no** está enlazado criptográficamente a esto de ninguna forma. Un
runbook crítico que termine en una escritura real sigue el patrón de
`AGENTS.md`: `clusterlog identity` → operación con la herramienta canónica →
validación observada → `--dry-run` → `--commit` en una celda manual,
interactiva y excluida de "Run All" → `clusterlog verify commit`.

Esta primera iteración **no** agrega un modelo nuevo de eventos de ejecución
(por ejemplo, un futuro `clusterlog runbook record`). Si se necesita, se
describe primero en un ADR con su propio modelo de amenazas y contrato JSON.

## Abrir y cerrar el modo notebook en VS Code

1. Instale la extensión recomendada en
   [`.vscode/extensions.json`](../.vscode/extensions.json)
   (`stateful.runme`).
2. Abra `RUNBOOK.md` o cualquier archivo en `runbooks/`; VS Code lo mostrará
   como notebook.
3. Para volver a verlo como texto: clic derecho en la pestaña → "Reopen
   Editor With..." → "Text Editor".

Ningún archivo `.md` fuera de `RUNBOOK.md`/`runbooks/` se fuerza a abrir como
notebook.

## Limitaciones conocidas de este entorno

- **Zola local es 0.22.1**, no la `0.23.3` que documenta `README.md` y que
  instala la CI. `make validate-local` / `validate --skip-zola` no dependen
  de la versión exacta; `make site`/`zola build` sí podrían comportarse
  distinto localmente que en CI.
- El escaneo de `clusterlog validate` sobre runbooks es un analizador de
  líneas con estado, no un parser completo de CommonMark: reconoce fences de
  una sola línea de atributos (`` ```sh {"name":...} ``). Fences de apertura
  multilínea no están soportados.
- La detección de "target de make inexistente" busca el patrón textual
  `make <nombre>` dentro del cuerpo de la celda; puede marcar un falso
  positivo si ese texto aparece dentro de un comentario o un `echo` (no
  ocurre en los runbooks incluidos, pero es una limitación real del
  analizador, no una garantía de que nunca puede pasar).
- La detección de Session Outputs es best-effort (ver arriba); no hay una
  garantía de que capture todo lo que Runme pudiera llegar a escribir en el
  futuro.
- `runme ls` lista **cualquier** fence de código, incluso uno sin atributos
  y con lenguaje `text` usado a propósito como texto de referencia (se
  confirmó con el binario real: aparecen como filas "No" en la columna
  `NAMED`). No son ejecutables por nombre y `--tag ci-safe --all` los
  ignora correctamente, pero ensucian la salida de `runme ls`/`make
  runme-list`. No se encontró una forma de excluirlos sin dejar de mostrar
  también snippets ilustrativos legítimos.

## Fuera de alcance de esta iteración

Terminal web, ejecución desde el sitio Zola, acceso directo al clúster desde
el navegador, un servicio Runme permanente, contenedor sidecar de
Kubernetes, Runme Cloud, publicación de Gists, autenticación propia de
Runme, reemplazo de Git/SSH, ejecución productiva en CI, conversión masiva de
manuales existentes, y runbooks para infraestructura que no existe en este
repositorio (bases de datos, escalamiento, rotación de secretos, despliegue a
producción).
