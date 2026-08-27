# ADR-0001: Git como fuente y Zola como proyección

- **Estado:** aceptada
- **Fecha:** 2026-08-12

## Contexto

El sitio debe mostrar estado compartido —revisiones, tiempo de tareas y estados— aunque Zola genere archivos estáticos. Guardar ese estado en el navegador no sería compartido ni auditable. Introducir una base de datos en el MVP aumentaría operación, permisos, respaldo y superficie de ataque.

## Decisión

La fuente de verdad será:

- Markdown para entidades legibles;
- un archivo JSON inmutable por evento;
- Git para historia y concurrencia;
- `clusterlog sync` para producir `data/generated/site_state.json`;
- Zola para consumir la proyección en modo sólo lectura.

La proyección es desechable y nunca se edita a mano.

## Consecuencias positivas

- El repositorio funciona sin un servicio permanente.
- Toda la historia es inspeccionable con herramientas estándar.
- Los conflictos se reducen al separar eventos.
- La publicación puede ser atómica y cacheable.
- Es posible reconstruir el sitio desde cero.

## Consecuencias negativas

- No hay edición web en tiempo real.
- Cada escritura necesita sincronización Git.
- Con volúmenes muy altos de eventos habrá que medir rendimiento y quizá archivar.
- La coherencia depende de CI, validación y disciplina de ramas.

## Alternativas descartadas

- `localStorage`: no compartido, fácil de perder y no auditable.
- Archivo JSON central mutable: conflictos frecuentes entre administradores.
- Base de datos desde el MVP: complejidad desproporcionada para el flujo y volumen iniciales.
