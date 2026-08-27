# Reunión semanal de revisión operativa

> Versión ejecutable de los comandos de esta página:
> [`runbooks/weekly-review.md`](../runbooks/weekly-review.md) (abrible como
> notebook con Runme, o copiable a mano). Este documento conserva la agenda
> y el criterio; los bloques de comando canónicos viven sólo ahí.

## Objetivo

Asegurar que el conocimiento de los cambios técnicos se distribuya, que los riesgos pendientes tengan dueño y que manuales, tareas y proyectos reflejen la realidad.

## Preparación individual

Antes de la reunión, cada administrador ejecuta:

```bash
clusterlog --root . --json --ssh-key ~/.ssh/id_ed25519.pub review list --mine
clusterlog --root . --json --ssh-key ~/.ssh/id_ed25519.pub task list --mine
```

Debe leer las memorias, anotar preguntas y revisar evidencias. Marcar “revisada” significa comprender el cambio y sus implicaciones; no necesariamente aprobar retrospectivamente una decisión.

## Agenda sugerida

### 1. Riesgo alto o crítico

Abrir primero memorias `critical` y `high`:

- resultado observado;
- validaciones;
- rollback;
- riesgo residual;
- seguimiento requerido.

### 2. Resto de memorias pendientes

Para cada entrada:

- confirmar sistemas afectados;
- aclarar términos y comandos;
- detectar conocimiento reusable;
- decidir si debe actualizarse un manual o documento;
- crear una tarea cuando exista una acción concreta.

Cada administrador registra su propia confirmación:

```bash
clusterlog --root . --json --commit --ssh-key ~/.ssh/id_ed25519.pub \
  review mark MEM-... --note "Revisado en reunión semanal"
```

No debe existir una sola persona marcando por todo el equipo.

### 3. Tareas abiertas

Revisar especialmente:

- bloqueadas;
- vencidas o próximas a vencer;
- con tiempo real muy superior a la estimación;
- con temporizador activo desde una sesión anterior;
- sin responsable.

### 4. Proyectos

Para proyectos planeados, activos o bloqueados:

- progreso contra objetivo;
- siguiente hito;
- dependencia o riesgo;
- tareas faltantes;
- fecha objetivo.

### 5. Cierre

- La cola de memorias debe quedar vacía o con una causa y fecha de seguimiento.
- Toda acción debe convertirse en tarea con responsable.
- Toda mejora repetible debe actualizar un manual.
- Todo cambio de arquitectura debe actualizar Documentación.

## Evidencia de cierre

```bash
clusterlog --root . sync --check
clusterlog --root . validate
clusterlog --root . verify commit HEAD
```

El secretario de la reunión puede enlazar los IDs discutidos en la minuta corporativa, sin duplicar contenido sensible.
