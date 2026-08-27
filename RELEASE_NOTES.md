# Notas de entrega — 0.1.0

**Fecha:** 2026-08-12

## Entrega

Esta versión convierte la idea inicial en un MVP ejecutable compuesto por:

- un sitio estático Zola de consulta;
- la CLI `clusterlog` como única interfaz de escritura;
- Markdown y eventos JSON anexables como datos fuente;
- Git y firmas SSH como capa de auditoría;
- una proyección determinista para la cola de revisión, tareas y proyectos.

## Capacidades principales

- Secciones de documentación, manuales, memorias, proyectos, tareas y revisión semanal.
- Identidad de administradores mediante huella de llave pública SSH.
- Registro generado de firmantes autorizados para Git.
- Memorias con contexto, cambios, validación, riesgo y reversión.
- Revisión por todos los administradores capturados al crear la memoria.
- Invalidación automática de revisiones tras cualquier cambio de contenido.
- Proyectos con objetivo, alcance, hitos y criterio de éxito.
- Tareas con estado, prioridad, responsables, estimación, vencimiento y tiempo efectivo.
- Eventos independientes para reducir conflictos de Git.
- Contrato JSON, entrada por `stdin`, `--dry-run`, salida estable y códigos de error para agentes de IA.
- Validación de contenido, referencias, proyección y secretos probables.
- Búsqueda local sin servicios externos.
- CI con pruebas, detector de carreras, validación Zola, construcción y control de firmas.

## Decisiones relevantes

- El navegador no modifica estado en el MVP; toda escritura pasa por la CLI y por Git.
- La huella pública atribuye una identidad registrada, pero sólo una firma Git verificada demuestra posesión autorizada de la llave.
- Cada revisión está ligada al SHA-256 de la versión exacta de la memoria.
- La lista de revisores se captura al crear la memoria para que altas futuras no reabran historia antigua.
- El tema `zola.386` aporta estilos y recursos, mientras las plantillas se mantienen localmente para aislar cambios de compatibilidad de Zola.

## Inicio recomendado

```bash
make theme
make build
./bin/clusterlog --root . sync
./bin/clusterlog --root . validate --skip-zola
zola serve
```

Antes de uso operativo, complete los pendientes de producción descritos en `HOJA_DE_PROYECTO.md`, `SECURITY.md` y `VALIDATION_REPORT.md`.
