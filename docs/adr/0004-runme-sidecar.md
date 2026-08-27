# ADR-0004: Runme como sidecar de documentación ejecutable

- **Estado:** aceptada
- **Fecha:** 2026-08-26

## Contexto

Los procedimientos frecuentes de `clusterlog` (compilar, probar, sincronizar,
validar, la revisión semanal) viven como bloques de comando dentro de
Markdown que hay que copiar y pegar a mano. Se evaluó integrar Runme
(`v3.17.4`, fijado) para poder abrir esos mismos procedimientos como
notebooks ejecutables desde VS Code o su CLI, sin introducir un segundo lugar
donde vivan los mismos comandos ni debilitar el modelo de identidad/auditoría
existente (huella SSH + commit firmado + `clusterlog verify commit`).

## Decisión

Se integra Runme como una capa opcional y adyacente:

- `RUNBOOK.md` (raíz) y `runbooks/**/*.md` son los únicos documentos que
  llevan atributos de celda Runme.
- Cada celda invoca exclusivamente `make`, `./bin/clusterlog` o un script ya
  existente en `scripts/`. Ninguna celda reimplementa validación,
  autorización, resolución de identidad, rollback o gestión de secretos.
- `clusterlog validate` (no un script aparte) hace cumplir las reglas de
  nombres, etiquetas y aislamiento de celdas mutantes — ver
  `internal/clusterlog/runbooks.go`.
- La instalación de Runme es opcional y reproducible
  (`scripts/runme-install.sh`, checksum verificado); ningún flujo existente
  (`make build`, `make test`, `clusterlog sync`/`validate`, `zola build`)
  depende de que Runme esté instalado.

## Significado de "sidecar"

No es un sidecar de Kubernetes ni un servicio adicional. Es un envoltorio: el
humano o agente interactúa con Markdown+Runme, que a su vez llama `make`,
`clusterlog` o un script — nunca al revés, y nunca con lógica propia.

## Ubicación de los runbooks

`RUNBOOK.md` en la raíz (índice) y `runbooks/` como carpeta dedicada, **fuera
de `content/`**. Se confirmó en `zola.toml` que Zola no tiene un
`content_dir` alterno: sólo construye desde `content/`. Poner los runbooks
fuera de esa carpeta significa que Zola nunca los toca — cero riesgo de que
un atributo de celda `{"name":...}` rompa el front matter TOML o el
resaltado de sintaxis de una página publicada. `clusterlog validate` además
hace cumplir esto activamente: si una celda de Runme aparece dentro de
`content/`, es un error (`runme_cell_in_zola_content`), no sólo una
recomendación.

## Relación con `content/manuales`

`content/manuales/` no se toca. Sigue siendo prosa + comandos de ejemplo para
lectura humana en el sitio Zola, sin atributos de celda ni ninguna
dependencia de Runme. Los runbooks nuevos no son una migración de los
manuales existentes: son procedimientos de desarrollo/operación de
`clusterlog` mismo que hoy no existían como documento ejecutable.

## Fuente de verdad

Opción elegida de las tres consideradas (sección 4.2 del pedido original):
**un archivo raíz delgado que enlaza procedimientos canónicos**, aplicada
así:

- `docs/WEEKLY_REVIEW.md` conserva la agenda y el criterio de la reunión
  (prosa) y enlaza a `runbooks/weekly-review.md` para los comandos
  ejecutables, en vez de mantener los mismos bloques de shell en dos
  archivos.
- `README.md` enlaza `RUNBOOK.md` y `docs/RUNME.md` en vez de duplicar
  ejemplos de comandos.

No se copian bloques de comando manualmente entre dos documentos en ningún
punto de esta integración.

## Convención de metadatos

Atributos de celda como objeto JSON de una sola línea después del lenguaje
del fence: `` ```sh {"name":"...","tag":"...","interactive":"...",
"excludeFromRunAll":true} ``. `clusterlog validate` decodifica ese objeto con
`encoding/json` (no confía en una regex para interpretar el contenido; la
regex sólo reconoce la *forma* de la línea de fence).

## Taxonomía de etiquetas

Cuatro categorías, tres obligatorias por celda:

| Categoría | Valores | Obligatoria |
|---|---|---|
| Audiencia | `role-developer`, `role-operator`, `role-reviewer`, `role-owner` | sí |
| Entorno | `env-local`, `env-ci`, `env-staging`, `env-production` | sí |
| Riesgo | `risk-read-only`, `risk-mutating`, `risk-destructive` | sí |
| Ejecución automatizada | `ci-safe`, `manual-only` | no, mutuamente excluyente con riesgo mutante |

## Modelo de identidad

Tres capas, documentadas en detalle en `docs/RUNME.md`:

1. **Identidad de contenido** (Lifecycle Identity / `name` de Runme):
   identifica documento y celda, no a quien ejecuta.
2. **Evidencia local de sesión** (Session Outputs, no habilitados aquí):
   auxiliar, nunca auditoría canónica.
3. **Auditoría canónica de Clusterlog**: huella SSH + `data/admins.json` +
   commit firmado + `clusterlog verify commit`, sin cambios.

No se afirma ni se implementa ningún enlace criptográfico entre Runme y un
commit. Un futuro `clusterlog runbook record` queda explícitamente fuera de
esta iteración (ver "Trabajo futuro").

## Modelo de evidencia

Session Outputs permanecen desactivados por defecto en esta integración. Si
se habilitan más adelante, deben quedar fuera de Git, del sitio Zola y de
Gists, y tratarse como evidencia auxiliar redactable, nunca como prueba de
autoría.

**Limitación reconocida**: la documentación pública de Runme para `v3.17.4`
no fija el nombre exacto de archivo de un Session Output
(`docs.runme.dev/usage/auto-save/` sólo recomienda "agréguelo a
`.gitignore`" sin dar el patrón). Se optó explícitamente por **no** agregar
un glob amplio e inseguro a `.gitignore` que pudiera ocultar Markdown
legítimo. En su lugar, `clusterlog validate` aplica dos señales concretas
(directorio `.runme/` versionado; archivos en `runbooks/` con "output" en el
nombre) como defensa best-effort, documentada como tal.

## CI/CD

`.gitea/workflows/ci.yml` (Gitea Actions, no GitHub Actions — este repo no
tiene `.github/`) gana un job nuevo y separado que no reemplaza
`test-and-build`: instala Runme fijado en `v3.17.4` vía
`scripts/runme-install.sh` (checksum verificado), corre `make runme-check`
(reglas de `clusterlog validate`, obligatorias, más un chequeo blando de
compatibilidad de Runme) y `make runme-ci`
(`runme run --filename RUNBOOK.md --tag ci-safe --all --skip-prompts`).
Permisos mínimos (`contents: read`), `DO_NOT_TRACK=true`,
`SCARF_NO_ANALYTICS=true`, sin llaves ni acceso a producción.

La CI original de Zola instala herramientas fijadas por `curl` + tar en vez
de una Action de marketplace (ver el paso "Instalar Zola fijado" ya
existente); Runme sigue el mismo patrón por consistencia, no por elección
nueva.

## Seguridad

Ver el nuevo apartado en `SECURITY.md`. En resumen: una celda Runme equivale
a ejecutar comandos locales con los permisos de quien la corre; una etiqueta
no es control de acceso, `excludeFromRunAll`/`interactive` son defensas
adicionales; Runme no lee llaves privadas; Runme Cloud, Gists y `runme open`
expuesto a `0.0.0.0` quedan fuera de alcance.

## Alternativas consideradas

1. **Celdas dentro de `content/manuales/`.** Descartada: arriesga el front
   matter TOML y el resaltado de Zola, y mezcla contenido para lectores con
   contenido para ejecución; no encaja con la opción 1 de fuente única
   (Markdown único compatible con ambos) porque el pedido de Zola y el de
   Runme divergen (TOML `+++` vs. atributos JSON de celda) sin un beneficio
   real, dado que `runbooks/` fuera de `content/` ya resuelve el conflicto
   sin ese riesgo.
2. **Validación en un script Bash aparte en vez de `clusterlog validate`.**
   Descartada para las reglas estructurales: encajan limpio en el patrón
   existente de `ValidationReport`/`report.add(...)` y ya tenían pruebas
   unitarias equivalentes para páginas/eventos. Sí se mantiene un script
   (`scripts/runme-check.sh`) para el chequeo blando que sólo tiene sentido
   con el binario real de Runme instalado.
3. **Selección de celdas CI-safe con una lista de nombres en el Makefile en
   vez del flag `--tag`.** Fue el plan inicial, basado en documentación
   pública de Runme que no mostraba un flag de filtrado por etiqueta. Al
   instalar el binario real de `v3.17.4` se confirmó que `runme run --tag`
   sí existe y filtra correctamente (ver `docs/RUNME.md`). Se adoptó
   `--tag ci-safe --all --skip-prompts` porque es la forma más simple que
   sigue siendo una allowlist explícita por etiqueta de seguridad, no un
   "correr todo" sin filtro.

## Consecuencias

Positivas:

- Los procedimientos de desarrollo se pueden ejecutar como notebook sin
  perder la posibilidad de copiar/pegar el comando a mano.
- `clusterlog validate` ahora también protege contra celdas mutantes mal
  etiquetadas, nombres duplicados y fugas de Runme hacia `content/`.
- La instalación reproducible con checksum evita depender de `latest` o de
  `curl | sh`.

Negativas / riesgos aceptados:

- El analizador de runbooks es un escáner de líneas, no un parser CommonMark
  completo; hay casos borde documentados en `docs/RUNME.md`.
- La detección de Session Outputs es best-effort, no una garantía.
- Dos documentos nuevos (`RUNBOOK.md`, `docs/RUNME.md`) que hay que mantener
  sincronizados con la realidad del `Makefile`.

## Limitaciones

- Zola local (`0.22.1`) no coincide con la versión que exige `README.md` y
  que instala la CI (`0.23.3`); no se intentó resolver esto en esta
  iteración.
- No se ejecutó ninguna celda `risk-mutating` de verdad durante la
  implementación (por diseño: requieren identidad real y quedan fuera del
  alcance de una demostración automatizada).
- No existe hoy un enlace criptográfico entre una ejecución de Runme y un
  commit; quien necesite esa garantía debe seguir el flujo completo de
  `AGENTS.md` fuera de Runme.

## Trabajo futuro

- Evaluar un comando `clusterlog runbook record` que capture, con su propio
  contrato JSON y modelo de amenazas, evidencia de que una celda `ci-safe`
  corrió — sólo si el equipo decide que Session Outputs no bastan como
  evidencia auxiliar.
- Migrar manuales existentes a runbooks ejecutables uno por uno, sólo cuando
  exista infraestructura real que respalde cada procedimiento (no antes).
- Revisar si vale la pena que `runme ls --json` exponga etiquetas en una
  versión futura de Runme, para poder validar la relación entre "celdas
  etiquetadas ci-safe" y "celdas que `runme-ci` realmente ejecutó" de forma
  automatizada en vez de manual.
