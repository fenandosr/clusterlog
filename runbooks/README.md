# Runbooks ejecutables

Esta carpeta vive **fuera** de `content/`, así que Zola nunca la construye ni
la publica (confirmado en `zola.toml`: no hay `content_dir` alterno). Es la
razón por la que aquí sí se pueden usar atributos de celda de Runme sin
arriesgar el front matter TOML ni el resaltado de sintaxis que sí usa
`content/manuales/`.

## Cómo abrir un runbook

- **Como notebook (Runme, VS Code):** instale la extensión recomendada en
  [`.vscode/extensions.json`](../.vscode/extensions.json) (`stateful.runme`) y
  abra el archivo `.md` normalmente; VS Code lo mostrará como notebook.
- **Como Markdown normal:** clic derecho sobre la pestaña → "Reopen Editor
  With..." → "Text Editor". Ningún runbook depende de estar en modo notebook
  para ser legible o copiable.
- **Por CLI:** `runme run --filename runbooks/<archivo>.md <nombre-de-celda>`.
- **Sin Runme instalado:** copie el bloque de comando y ejecútelo en su
  terminal; ninguna celda usa sintaxis que no sea shell/`make`/`clusterlog`
  normal.

## Convención de nombres

`kebab-case`, prefijo de proyecto, único en **todo** el repositorio (no sólo
dentro de un archivo): `clusterlog-build`, `clusterlog-validate-local`,
`clusterlog-review-mine`. `clusterlog validate` (ver
`internal/clusterlog/runbooks.go`) falla si dos celdas repiten un nombre.

## Taxonomía de etiquetas

Toda celda debe declarar al menos una etiqueta de cada categoría:

| Categoría | Valores | Obligatoria |
|---|---|---|
| Audiencia | `role-developer`, `role-operator`, `role-reviewer`, `role-owner` | sí |
| Entorno | `env-local`, `env-ci`, `env-staging`, `env-production` | sí |
| Riesgo | `risk-read-only`, `risk-mutating`, `risk-destructive` | sí |
| Ejecución automatizada | `ci-safe`, `manual-only` | no (pero mutuamente excluyente con `risk-mutating`/`risk-destructive`) |

Una celda `risk-mutating` o `risk-destructive` **debe** llevar
`manual-only`, `"interactive":"true"` y `"excludeFromRunAll":true`, y
**no puede** llevar `ci-safe` al mismo tiempo. `clusterlog validate` hace
cumplir esto — no es sólo una convención de estilo.

## Qué no va en una celda

Ver la sección "Runme no contiene lógica de negocio" del ADR
[`docs/adr/0004-runme-sidecar.md`](../docs/adr/0004-runme-sidecar.md). En
resumen: una celda invoca `make`, `./bin/clusterlog` o un script existente; no
reimplementa validación, autorización, rollback ni gestión de secretos.
