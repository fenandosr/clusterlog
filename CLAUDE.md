# Claude Code: operación de la bitácora

Sigue primero `AGENTS.md`. Estas reglas son específicas para sesiones de administración asistidas.

## Al iniciar

- Confirma la raíz con `clusterlog --root . --json identity` usando la llave proporcionada por el entorno.
- Consulta tareas asignadas con `task list --mine` cuando la sesión parte de trabajo planeado.
- No inicies un temporizador si ya existe otro activo para el administrador.

## Durante la sesión

- Mantén notas temporales fuera del repositorio hasta redactarlas y eliminar secretos.
- Separa observación, hipótesis, acción y validación.
- No registres como ejecutado un comando sólo porque fue sugerido.
- Usa `task start` y `task stop` para trabajo rastreado; usa `adjust` sólo con justificación.

## Al cerrar

- Prepara una memoria cuando hubo un cambio, corrección, diagnóstico significativo o aprendizaje reutilizable.
- Incluye un rollback accionable para riesgo `high` o `critical`.
- Actualiza o crea un manual cuando el procedimiento deba repetirse.
- Ejecuta `--dry-run`, luego la escritura con `--commit` si está autorizada.
- Devuelve un resumen con IDs, archivos, tiempo, validación, advertencias y SHA firmado.

## Runbooks ejecutables (Runme)

- No actives una celda `risk-mutating`/`risk-destructive` de `RUNBOOK.md` o `runbooks/` sin autorización explícita del usuario para esa operación puntual.
- Una celda `ci-safe` sigue siendo el mismo comando que documenta; aplícale las mismas reglas de `--dry-run`/`--commit` que usarías si lo tipearas tú mismo.
- No agregues lógica nueva dentro de una celda ni edites un identificador Lifecycle Identity generado por Runme.

## Formato de respuesta al usuario

```text
Resultado: <éxito o error verificado>
Tarea: <TSK-ID y tiempo, cuando aplique>
Memoria: <MEM-ID>
Archivos: <rutas>
Validación: <comandos realmente ejecutados>
Commit: <SHA o “no creado”>
Pendientes: <riesgos o acciones concretas>
```

No ocultes un fallo de firma o validación detrás de una descripción optimista.
