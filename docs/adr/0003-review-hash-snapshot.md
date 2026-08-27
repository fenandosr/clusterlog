# ADR-0003: Revisiones por hash y conjunto capturado

- **Estado:** aceptada
- **Fecha:** 2026-08-12

## Contexto

Un booleano `revisada` no indica quién revisó ni qué versión vio. Además, añadir un administrador meses después podría reabrir todo el histórico si la política se recalculara siempre contra el equipo actual.

## Decisión

Al crear una memoria:

- se captura el conjunto de revisores requerido;
- se calcula un hash SHA-256 sobre la versión de contenido;
- cada revisión registra administrador, huella, fecha y hash;
- sólo cuentan revisiones cuyo hash coincide con el actual;
- la cola se completa cuando todos los IDs capturados tienen un evento válido.

## Consecuencias positivas

- Editar una memoria invalida automáticamente el consenso anterior.
- La cola histórica no cambia al incorporar administradores futuros.
- Es posible explicar exactamente por qué una entrada está pendiente.
- Los eventos permanecen inmutables y auditables.

## Consecuencias negativas

- Corregir incluso un detalle menor puede reabrir la revisión.
- El equipo debe decidir cuándo una corrección editorial merece conservar o repetir consenso; el MVP prioriza seguridad y vuelve a abrir siempre.
- Bajas de personal requieren una política de excepción explícita, todavía no automatizada.
