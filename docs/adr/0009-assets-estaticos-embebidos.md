# ADR-0009: Embeber los assets estáticos en el binario del motor

- **Estado:** aceptada
- **Fecha:** 2026-09-22

## Contexto

`README.md` documenta desde siempre un directorio `static/` en la
"Estructura" del motor ("ajustes visuales y búsqueda"), y todas las
plantillas (`base.html`, `page.html`, etc.) referencian `css/ops.css`,
`js/ops-search.js` y `fonts/ShareTechMono-Regular.woff2`. Pero ese
directorio nunca se comiteó en este repositorio (`git log --all -- static/`
no devuelve nada): el diseño real del sitio se escribió, en cambio,
directamente dentro de la instancia `clusterlog-mcps` (bootstrap inicial +
dos commits de ajuste de diseño), y de ahí nunca volvió al motor.

Esto se detectó en producción real: `pangea1`, una instancia nueva
bootstrapeada con este motor, se veía sin ningún estilo (sólo el reset de
~1&nbsp;KB que deja `zola.386`) y con la búsqueda rota — exactamente lo que
predice la ausencia de `static/`. `css/ops.css` y `js/ops-search.js` son
genéricos (revisados campo a campo antes de portarlos: sin dominios,
nombres de host ni datos de ningún deployment real), así que no hay
conflicto con ADR-0005.

## Decisión

Se portan `css/ops.css`, `js/ops-search.js`, `fonts/ShareTechMono-Regular.woff2`
y `robots.txt` desde `clusterlog-mcps` a
`internal/clusterlog/scaffold/static/`, embebidos en el binario con
`go:embed` — mismo mecanismo que ya usa `templates/` desde ADR-0006. Un
comando nuevo, `clusterlog static extract [--output DIR]`, los escribe en
disco (por defecto `<root>/static`) sin ninguna llamada de red.
`bootstrap-instance` lo invoca automáticamente al crear una instancia
nueva, igual que ya hace con `templates/`.

Se mantiene además una copia en `static/` en la raíz de este repositorio
(idéntica a la embebida, sincronizada a mano), porque el motor construye su
propio sitio de documentación con `zola build`/`zola serve` — mismo patrón
que ya existe entre `templates/` (raíz) y
`internal/clusterlog/scaffold/templates/` (embebida).

## Consecuencias positivas

- Toda instancia nueva se ve como corresponde desde el primer
  `bootstrap-instance`, sin un paso manual de "andá a copiar `static/` de
  otra instancia que ya lo tenga" (que es exactamente lo que había que
  hacer hasta ahora, y lo que motivó este ADR).
- Binario y assets son, por construcción, siempre de la misma versión — el
  mismo argumento de ADR-0006 aplica igual acá.
- `README.md` deja de documentar una estructura que no existía: lo que dice
  "Estructura" ahora es cierto.

## Consecuencias negativas

- El binario crece un poco más (CSS + JS + una fuente woff2 de ~13&nbsp;KB
  — no es enorme, pero es más que las plantillas HTML de ADR-0006).
- El diseño visual (`ops.css`) queda fijado dentro del binario del motor:
  una instancia que quiera personalizar colores/tipografía sin forkear el
  motor no tiene hoy un mecanismo de override — mismo punto abierto que
  ADR-0006 dejó para plantillas, ahora también aplica a estilos.
- Dos copias a mantener sincronizadas a mano (`static/` raíz y
  `internal/clusterlog/scaffold/static/`), igual que ya pasa con
  `templates/` y `content/topologia/`. No hay una prueba automática que
  detecte que se desincronizaron.

## Alternativas descartadas

- **Dejar `static/` fuera del motor y que cada instancia lo escriba a
  mano**, documentando el patrón en vez de embeberlo. Descartado: es
  exactamente el estado que causó el problema en `pangea1` — cualquier
  instancia nueva nace rota hasta que alguien copia el directorio de otra a
  mano.
- **Reescribir `ops.css`/`ops-search.js` desde cero para el motor**, en vez
  de portar los de `mcps`. Descartado: ya existen, están probados en
  producción real (`mcps`) desde agosto, y son genéricos — no hay ninguna
  razón para no reusarlos.

## Trabajo futuro

- Si en algún momento se necesita personalizar el diseño por instancia sin
  forkear el motor, la extracción ya deja `ops.css` como texto plano
  editable en disco (mismo argumento que el "Trabajo futuro" de ADR-0006
  para plantillas) — un mecanismo de "vendored + overrides" podría cubrir
  ambos (plantillas y estilos) a la vez.
