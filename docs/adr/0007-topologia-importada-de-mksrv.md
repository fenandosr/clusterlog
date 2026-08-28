# ADR-0007: Importar topología de infraestructura desde mksrv

- **Estado:** aceptada
- **Fecha:** 2026-08-28

## Contexto

Documentar a mano qué host es cuál, qué IP tiene y qué rol cumple es
exactamente el tipo de conocimiento que se desactualiza sin avisar (la falla
1 de `HOJA_DE_PROYECTO.md`, sección 2). Cuando la infraestructura la crea
Terraform a través de `mksrv` (proyecto hermano, mismo autor, mismo patrón
motor/instancia que este repositorio — ver `~/repos/mksrv`), esos datos ya
existen en disco, estructurados y actualizados por la propia herramienta:
`deployment.yaml` (estado declarado: qué host lleva qué stack) y
`.mksrv/infra/outputs.json` (estado real, volcado de `terraform output`:
IPs, ID de instancia, zona de disponibilidad, red, DNS). Escribir eso a mano
en una página de Documentación es transcribir un archivo que ya existe, con
el riesgo real de que se desactualice en cuanto alguien corra `terraform
apply` de nuevo.

Esta decisión es sobre el **contrato de datos y el comando de la CLI** de
clusterlog, no sobre la topología de ningún clúster administrado en
particular — por eso corresponde un ADR (afecta el modelo de datos y el
contrato de la CLI, ver `docs/ADR.md`) y no una página de Documentación.

## Decisión

Se agrega `clusterlog topology import --from-mksrv DIR [--output PATH]
[--include-public-ip]`. Lee `deployment.yaml` y `.mksrv/infra/outputs.json`
(y, si existe, `.mksrv/mesh.json`) de un workspace de mksrv y escribe
`data/topology.json` en un **contrato genérico** (`Topology` en
`internal/clusterlog/topology.go`): lista de hosts con rol, proveedor,
stacks, direcciones (privada/gestión/malla/pública) e identificadores de
nube, más red y DNS a nivel de deployment. El contrato no expone ningún tipo
de mksrv fuera de `topology.go` — un adaptador futuro de otra herramienta
(`--from-<otra-cosa>`) produciría la misma forma sin tocar nada río abajo.

El rol de cada host (`edge`/`data`) se deriva igual que ya lo hace el propio
`mksrv` (`internal/cli/hosts.go`, `renderContext`): presencia del stack
`base` en `deployment.yaml`. Un host en `outputs.json` sin apply reciente
queda con sus direcciones vacías y una advertencia (`ExitOK` con
`warnings`, no un error) en vez de fallar todo el comando. La IP pública es
**opt-in** (`--include-public-ip`): igual que el motor nunca embebe datos
reales, este comando nunca decide por el operador si esa IP entra al
contenido publicado.

El comando no firma ni comitea nada — es un paso de generación de datos,
igual que `templates extract`, no de contenido auditado.

Este ADR **no** decide todavía cómo `data/topology.json` se refleja en el
sitio Zola (nueva sección "Topología", plantilla, integración con `sync` y
con la taxonomía `systems`) — eso queda para un ADR posterior una vez que
haya una instancia real usándolo; ver "Trabajo futuro".

## Consecuencias positivas

- La topología deja de transcribirse a mano: se deriva de los mismos
  archivos que ya produce y mantiene `mksrv apply --infra-only`.
- El contrato de salida es agnóstico de mksrv por diseño: adoptar otra
  herramienta de aprovisionamiento en el futuro no exige rehacer nada río
  abajo, sólo un adaptador nuevo con la misma forma de salida.
- Reutiliza la taxonomía `systems` que ya existe (memorias/manuales
  etiquetados por sistema afectado) como mecanismo de cruce, sin agregar
  ninguna maquinaria de enlace nueva — pendiente de que la fase de
  integración con Zola use los mismos nombres de host como valores de
  `systems`.
- Validado end-to-end antes de mergear: se corrió contra un workspace mksrv
  real (dos hosts AWS) y el mapeo de rol, direcciones, instance_id, az, red
  y DNS coincidió campo a campo con `outputs.json`.

## Consecuencias negativas

- Primera dependencia externa de este repositorio (`gopkg.in/yaml.v3`, para
  decodificar `deployment.yaml`). El motor no tenía ninguna hasta ahora. Se
  consideró y descartó escribir un parser YAML propio acotado (ver
  Alternativas descartadas) — para un formato con anidación, listas en
  flujo (`[a, b, c]`) y comentarios, un parser de mano es más riesgo que una
  librería madura de un solo propósito.
- El adaptador decodifica sólo el subconjunto de campos que usa (no todo
  `internal/model/model.go` de mksrv): si mksrv cambia la forma de esos
  campos concretos sin avisar, el import falla o produce datos incompletos.
  No hay ninguna prueba de contrato cruzada entre los dos repositorios hoy.
- `data/topology.json` no se integra todavía con `sync`, `validate` ni
  ninguna plantilla Zola — hoy es un archivo generado que nadie consume
  todavía dentro de clusterlog.

## Alternativas descartadas

- **Un script de shell/jq en cada instancia**, en vez de un comando del
  motor. Descartado: el mapeo de rol (`stacks` contiene `base` → `edge`) es
  lógica real, no un simple `jq` de campos; tenerla en el motor, versionada
  y con pruebas, evita que cada instancia la reimplemente distinto o la
  deje desactualizada respecto a cómo `mksrv` mismo deriva el rol.
- **Parser YAML propio, acotado a lo que usa `deployment.yaml`**, para no
  romper la ausencia de dependencias externas del motor. Descartado: YAML
  es fácil de parsear mal (indentación, listas en flujo, comentarios) y el
  costo de un bug silencioso en un parser casero supera el costo de una
  dependencia bien establecida y de un solo propósito.
- **Modelar el contrato completo de mksrv** (`internal/model.Deployment`
  entero) en vez de un subconjunto mínimo. Descartado: acopla este
  adaptador a campos que topología nunca usa (correo, backend de Terraform,
  telemetría) y lo vuelve frágil ante cualquier cambio de mksrv que no
  toque topología en absoluto.
- **Incluir la IP pública por defecto.** Descartado: es una decisión de
  contenido (qué tan expuesto queda un dato en el sitio publicado), no una
  decisión técnica; el motor no la toma por el operador (mismo principio
  que llevó a que el motor nunca embeba datos reales de ningún deployment).

## Trabajo futuro

- Integrar `data/topology.json` con `clusterlog sync` (plegarlo a
  `site_state.json`) y con una plantilla Zola nueva para una sección
  "Topología", usando los nombres de host como valores de la taxonomía
  `systems` para el cruce automático con memorias/manuales.
- Decidir si `clusterlog validate` debe advertir cuando `data/topology.json`
  existe pero está más viejo que `outputs.json` del workspace mksrv
  referenciado (haría falta guardar esa ruta en algún lado — hoy no se
  persiste).
- Si aparece una segunda herramienta de aprovisionamiento real en uso,
  formalizar el punto de extensión (`--from-<adaptador>`) en vez de dejarlo
  implícito en el switch de `runTopologyImport`.
