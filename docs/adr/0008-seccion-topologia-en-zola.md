# ADR-0008: Sección "Topología" en el sitio Zola

- **Estado:** aceptada
- **Fecha:** 2026-09-15

## Contexto

ADR-0007 agregó `clusterlog topology import --from-mksrv`, que escribe
`data/topology.json`, pero dejó explícitamente pendiente cómo ese archivo se
refleja en el sitio publicado ("Este ADR no decide todavía... eso queda para
un ADR posterior una vez que haya una instancia real usándolo"). ADR-0007 ya
validó el mapeo campo a campo contra el fleet real de mksrv que administra
el autor (`mcps`, dos hosts AWS), así que la integración con Zola deja de
ser especulativa aunque ninguna instancia tenga todavía `data/topology.json`
comiteado — corresponde decidir el contrato de la sección ahora, antes de
que la primera instancia lo corra en serio.

`data/topology.json` es, por naturaleza, la misma clase de dato que
`data/generated/site_state.json` ya modela para la cola de revisión: una
vista computada sobre datos estructurados, no contenido autorado a mano en
Markdown. El precedente ya existente en el motor (`ReviewQueue` en
`SiteState`, sección "Revisión" con `template = "review.html"`,
`in_search_index = false`) es el molde correcto a copiar, en vez de forzar
la topología dentro del esquema de `content create` (pensado para
`documentacion`/`manuales`/`memorias`/`proyectos`/`tareas`, todo Markdown con
`extra.id` y front matter validado por `clusterlog validate`).

## Decisión

- `SiteState` (en `internal/clusterlog/types.go`) gana un campo opcional
  `Topology *Topology` que es literalmente el contrato de `topology.go` sin
  reproyectar — no hay un tipo `TopologyState` paralelo. `BuildState`
  (`state.go`) lee `data/topology.json` si existe (ausencia no es error:
  no toda instancia corre `topology import`) y lo agrega a las fuentes que
  entran al hash de `sync`, para que `sync --check` detecte cuando el JSON
  cambió y el sitio publicado quedó desactualizado respecto a él.
- Se sube `siteStateVersion` de 2 a 3: el campo nuevo hace que
  `site_state.json` generado por un binario viejo ya no calce con el
  esquema que las plantillas nuevas esperan, y `sync --check` debe marcarlo
  como obsoleto en vez de servirlo tal cual.
- Sección nueva `topologia` (`content/topologia/_index.md`,
  `template = "topology.html"`, `in_search_index = false`) agregada a
  `scaffoldSections` en `commands_bootstrap.go` — toda instancia nueva la
  trae de fábrica, igual que `revision`. `templates/topology.html` sigue el
  mismo patrón visual que `tasks.html` (tarjetas `ops-card` con `dl/dt/dd`
  para las direcciones de cada host) y `review.html` (encabezados de
  sección, estado vacío con el comando exacto a correr). Un enlace nuevo en
  la barra de navegación de `base.html`, entre "Tareas" y "Revisión".
- Cada host enlaza a `get_taxonomy_url(kind="systems", name=host.name)`
  cuando existe ese término (es decir, cuando alguna memoria o manual ya
  usó ese nombre de host en su `systems = [...]`) — la taxonomía `systems`
  ya existía en `zola.toml` desde antes de este ADR; no se agrega
  maquinaria de enlace nueva, sólo se consume la que ya había. Si el
  término no existe todavía, el nombre se muestra sin enlace en vez de
  producir un enlace roto.
- Como con `revision`, la página es puramente de lectura: no se agrega
  ningún subcomando de `content create` para `topologia`, y
  `validatePages` (`validate.go`) no la incluye en su lista de secciones
  autorables — sigue existiendo sólo vía `topology import` + `sync`.

## Consecuencias positivas

- Cero contenido nuevo que mantener a mano: la sección aparece con datos
  reales en cuanto una instancia corre `topology import` + `sync`, igual
  que "Revisión" aparece poblada sin que nadie escriba HTML.
- Reutiliza exactamente el patrón ya probado (fuente de datos JSON →
  `SiteState` → `load_data` en `base.html` → plantilla) en vez de inventar
  uno paralelo — menos superficie nueva que las pruebas y `AGENTS.md` deben
  cubrir.
- El enlace a la taxonomía `systems` por nombre de host cumple lo que
  ADR-0007 dejó como intención ("Trabajo futuro") sin agregar ningún campo
  ni comando nuevo: sólo consume `get_taxonomy_url`, una función ya
  provista por Zola.

## Consecuencias negativas

- `siteStateVersion = 3` invalida el `site_state.json` de toda instancia
  existente hasta que corra `sync` de nuevo con el binario actualizado —
  aceptable porque `sync --check` ya está diseñado exactamente para
  detectar y señalar esto, no para fallar en silencio.
- La plantilla asume el contrato `Topology` de `topology.go` tal cual
  (`hosts[].addresses.{private,management,mesh,public}`, `network`, `dns`);
  si ese contrato cambia de forma en el futuro (por ejemplo, un segundo
  adaptador `--from-<otra-cosa>` que no llene todos los campos), la
  plantilla ya tolera campos vacíos vía `{% if %}`, pero un cambio de forma
  (no sólo de valores) sí requeriría tocar `topology.html`.

## Alternativas descartadas

- **Modelar `topologia` como una sección de contenido autorable** (Markdown
  con `extra.id`, validado como `documentacion`/`manuales`). Descartado:
  el dato ya vive estructurado y actualizado en `data/topology.json`;
  convertirlo en Markdown a mano sería exactamente el problema que
  ADR-0007 evitó al agregar el import automático.
- **Un tipo `TopologyState` propio en `types.go`**, distinto de `Topology`
  (`topology.go`), para no acoplar el JSON publicado al contrato del
  importador. Descartado: `Topology` ya es un contrato genérico a
  propósito (no expone nada de mksrv, ver ADR-0007) y agregar una capa de
  reproyección solo para tener dos structs idénticos no compra nada hoy.
