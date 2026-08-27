# Bitácora Operativa del Clúster

Sitio interno y CLI auditable para conservar en un solo lugar la documentación, los manuales, las memorias de cambio, los proyectos, las tareas y la agenda de revisión semanal del equipo que administra un clúster.

La solución separa responsabilidades:

```text
administrador o agente IA
          |
          v
     clusterlog CLI  ---- identifica por huella SSH
          |          ---- valida contratos y secretos probables
          |          ---- escribe Markdown/eventos JSON
          |          ---- opcionalmente crea commit SSH firmado
          v
     repositorio Git ---- fuente de verdad y auditoría
          |
          v
     clusterlog sync ---- genera data/generated/site_state.json
          |
          v
        Zola         ---- lectura, navegación, taxonomías y búsqueda
```

## Qué incluye el MVP

- Secciones de **Documentación**, **Manuales**, **Memorias**, **Proyectos**, **Tareas** y **Revisión**.
- Memorias operativas con autor, huella SSH, riesgo, sistemas, etiquetas, ticket, duración, validación y reversión.
- Cola de revisión por administrador. Una memoria desaparece cuando todos los revisores capturados para esa entrada la aprobaron.
- Invalidación automática de revisiones cuando cambia el contenido de una memoria.
- Tareas con estimación, fecha objetivo, responsables, estado y tiempo real mediante eventos `start`, `stop` y `adjust`.
- Proyectos con objetivo, responsables, alcance, hitos, criterio de éxito y estado.
- CLI no interactiva, salida JSON, entrada JSON por archivo o `stdin`, modo `--dry-run` y códigos de salida estables.
- Identidad por huella de llave pública SSH y commits Git firmados por SSH con `--commit`.
- Detección heurística de secretos antes de escribir contenido.
- Estado derivado y reproducible para que Zola siga siendo un sitio estático.

## Requisitos

- Go 1.23 o posterior.
- Git con soporte de firmas SSH.
- OpenSSH/`ssh-keygen` para firmar y verificar commits.
- Zola 0.23.3 para construir el sitio.
- Acceso al repositorio del tema durante `make theme`, o una copia local indicada con `ZOLA386_SOURCE`.

## Inicio rápido

```bash
make theme
make build
./bin/clusterlog --root . sync
./bin/clusterlog --root . validate --skip-zola
zola serve
```

El sitio queda disponible en la dirección que muestre Zola. Antes de publicarlo, cambie `base_url` en `zola.toml` y manténgalo detrás de VPN, red privada o autenticación corporativa.

## Crear una instancia nueva

Este repositorio hoy mezcla la herramienta con el contenido de un deployment. Para empezar un deployment **distinto** sin partir de este contenido, use:

```bash
clusterlog bootstrap-instance --output ./mi-deploy --product-name "Mi Bitácora"
```

Genera `zola.toml`, las seis secciones de `content/` vacías, `data/admins.json`/`config/allowed_signers` vacíos y `.clusterlog-version`. Ver la sección 17 de [`CLI.md`](CLI.md) para el detalle completo; sigue siendo un primer paso — todavía no incluye plantillas/tema empaquetados ni CI de instancia (el motor no publica releases binarios todavía).

## Registrar al primer administrador

La llave pública identifica el registro del administrador. Para una auditoría fuerte, las operaciones posteriores deben usar `--commit` y la rama protegida debe rechazar commits sin firma válida.

```bash
./bin/clusterlog \
  --root . \
  --ssh-key ~/.ssh/id_ed25519.pub \
  admin add \
  --id ana \
  --name "Ana Torres" \
  --email ana@example.org \
  --key ~/.ssh/id_ed25519.pub \
  --roles owner,admin
```

Configure Git para firmar el primer commit completo:

```bash
git init
git config user.name "Ana Torres"
git config user.email ana@example.org
git config gpg.format ssh
git config user.signingkey ~/.ssh/id_ed25519
git config commit.gpgsign true
git add .
git commit -S -m "bootstrap: iniciar bitácora operativa"
```

A partir de ahí:

```bash
./bin/clusterlog --root . --ssh-key ~/.ssh/id_ed25519.pub --commit identity
./bin/clusterlog --root . --ssh-key ~/.ssh/id_ed25519.pub --commit memory create --from-json examples/memory.json
```

`identity` no modifica el repositorio; `--commit` sólo tiene efecto en comandos que escriben archivos.

## Flujo para Claude Code u otro agente

1. El agente recopila y redacta los hechos de la sesión, sin incluir credenciales ni valores sensibles.
2. Consulta el contrato:

   ```bash
   ./bin/clusterlog --json schema memory
   ```

3. Prevalida sin escribir:

   ```bash
   cat session-memory.json | ./bin/clusterlog \
     --root . --json --dry-run \
     --ssh-key ~/.ssh/id_ed25519.pub \
     memory create --from-json -
   ```

4. Escribe y firma una unidad atómica:

   ```bash
   cat session-memory.json | ./bin/clusterlog \
     --root . --json --commit \
     --ssh-key ~/.ssh/id_ed25519.pub \
     memory create --from-json -
   ```

5. El agente devuelve al humano el `entry_id`, la ruta y el SHA del commit; nunca afirma que la firma fue válida sin comprobar la salida.

Las instrucciones completas para agentes están en [`AGENTS.md`](AGENTS.md) y [`CLAUDE.md`](CLAUDE.md).

## Flujo de revisión semanal

```bash
# Ver sólo las memorias que le faltan al administrador actual
./bin/clusterlog --root . --json --ssh-key ~/.ssh/id_ed25519.pub review list --mine

# Marcar la versión actual como revisada y crear un commit firmado
./bin/clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  review mark MEM-... --note "Revisado en reunión semanal"

# Confirmar la firma del commit actual
./bin/clusterlog --root . verify commit HEAD
```

La página `/revision/` contiene la cola pendiente y, como apoyo a la reunión, las tareas abiertas y los proyectos planeados, activos o bloqueados.

## Comandos de calidad

```bash
make test            # pruebas unitarias e integración de la CLI
make vet             # análisis estático de Go
make e2e             # flujo completo en una copia temporal
make smoke           # contratos, sincronización y validación local
make validate-local  # sin requerir Zola
make site            # tema + estado + validación completa + sitio
make ci              # equivalente local de la canalización
```

## Documentación ejecutable (Runme)

Los procedimientos frecuentes también existen como notebooks ejecutables,
sin reemplazar ningún comando canónico: [`RUNBOOK.md`](RUNBOOK.md) es el
punto de entrada. Requiere Runme `v3.17.4` (`make runme-install`) sólo si se
quiere abrir como notebook o correr por CLI; cada celda es, además, un
bloque de shell normal que sigue funcionando copiado a mano. Detalles,
modelo de identidad y limitaciones conocidas en
[`docs/RUNME.md`](docs/RUNME.md) y la decisión en
[`docs/adr/0004-runme-sidecar.md`](docs/adr/0004-runme-sidecar.md).

## Estructura

```text
cmd/clusterlog/          punto de entrada de la CLI
internal/clusterlog/     modelo, comandos, validación y proyección
content/                 Markdown fuente por sección
data/admins.json         registro de administradores y política
data/review-events/      un archivo por revisión
data/task-events/        un archivo por evento de tiempo/estado
data/generated/          proyección regenerable para Zola
templates/               plantillas locales compatibles con Zola actual
static/                  ajustes visuales y búsqueda
scripts/                 preparación del tema y smoke tests
docs/                    decisiones arquitectónicas y operación
```

## Documentación clave

- [`HOJA_DE_PROYECTO.md`](HOJA_DE_PROYECTO.md): alcance, requisitos, fases y criterios de aceptación.
- [`CLI.md`](CLI.md): contrato de comandos, JSON y códigos de salida.
- [`SECURITY.md`](SECURITY.md): modelo de confianza y despliegue recomendado.
- [`THEME_COMPATIBILITY.md`](THEME_COMPATIBILITY.md): estrategia para `zola.386`.
- [`docs/WEEKLY_REVIEW.md`](docs/WEEKLY_REVIEW.md): guion de la reunión semanal.
- [`docs/ADR.md`](docs/ADR.md): decisiones de diseño de **clusterlog** (la herramienta), no del clúster que administra; formato, campos y ejemplo. Registros en [`docs/adr/`](docs/adr/), sin equivalente en la UI del sitio ni en la CLI.
- [`docs/RUNME.md`](docs/RUNME.md): documentación ejecutable con Runme — instalación, modelo de identidad y limitaciones. Punto de entrada: [`RUNBOOK.md`](RUNBOOK.md).
- [`VALIDATION_REPORT.md`](VALIDATION_REPORT.md): pruebas ejecutadas, límites y controles pendientes.
- [`RELEASE_NOTES.md`](RELEASE_NOTES.md): alcance de la entrega 0.1.0.

## Estado del prototipo

La CLI compila con Go 1.23 y sus pruebas cubren creación por JSON, resolución de identidad, revisión por hash, proyección de tareas y eventos independientes. La construcción de Zola está configurada en la CI incluida. Cuando el registro ya contiene administradores activos, la CI también verifica cada commit del pull request contra `config/allowed_signers`; el scaffold vacío omite esa comprobación durante el bootstrap inicial.
