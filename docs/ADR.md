# Decisiones de arquitectura (ADR)

## Alcance

Un ADR (*Architecture Decision Record*) documenta una decisión de diseño sobre **clusterlog**: la herramienta, su modelo de datos, su CLI, su proyección estática y su cadena de confianza. No es el lugar para decisiones operativas del clúster que administra el equipo (topología de nodos, políticas de red, capacidad, etc.); esas pertenecen a **Manuales** o **Memorias** dentro del sitio.

Un ADR se justifica cuando la decisión:

- es costosa o difícil de revertir;
- afecta el modelo de datos, la identidad/firma, o el contrato de la CLI;
- descarta una alternativa razonable por un motivo que no es obvio leyendo el código;
- va a ser preguntada de nuevo en el futuro ("¿por qué no usamos X?").

No se justifica para elección de librerías internas reversibles, estilo de código, o correcciones de bugs — eso va en el mensaje de commit o, si amerita contexto, en una memoria.

## Dónde viven y cómo se gestionan

- Los archivos están en [`docs/adr/`](adr/), uno por decisión.
- **No se ven en la UI del sitio ni se gestionan con la CLI de `clusterlog`.** `docs/adr/` no forma parte de `content/` (lo que Zola proyecta) y no existe ningún comando `clusterlog adr *`. Son Markdown plano, editado a mano, versionado y revisado como el resto del repositorio.
- Se crean, discuten y aprueban por revisión de código normal (pull request), igual que cualquier otro cambio al repositorio.

## Nombre de archivo

```text
docs/adr/NNNN-titulo-en-kebab-case.md
```

- `NNNN`: número secuencial de 4 dígitos, sin reutilizar ni renumerar al superar un ADR.
- El slug es una versión corta del título en minúsculas, con guiones.

## Campos y secciones

Cada ADR sigue esta estructura, en este orden:

```markdown
# ADR-NNNN: Título de la decisión

- **Estado:** propuesta | aceptada | rechazada | reemplazada por ADR-MMMM
- **Fecha:** AAAA-MM-DD

## Contexto

## Decisión

## Consecuencias positivas

## Consecuencias negativas

## Alternativas descartadas   (opcional)

## Regla de interpretación    (opcional)
```

| Campo/Sección | Obligatorio | Contenido |
|---|---|---|
| Título | sí | Frase corta que nombra la decisión, no el problema. |
| Estado | sí | Uno de: `propuesta`, `aceptada`, `rechazada`, `reemplazada por ADR-MMMM`. Un ADR aceptado no se borra ni se edita para invertir la decisión; se agrega uno nuevo que lo reemplaza y se actualiza el estado del anterior. |
| Fecha | sí | Fecha en que el estado actual entró en vigor. |
| Contexto | sí | La fuerza o restricción que exige decidir: qué problema, qué tensión, qué no puede seguir igual. Sin esto, la decisión no se puede evaluar más adelante. |
| Decisión | sí | Qué se decidió, en términos concretos y verificables (no aspiracionales). |
| Consecuencias positivas | sí | Efectos deseables que se aceptan a cambio. |
| Consecuencias negativas | sí | Costos, riesgos o deuda que la decisión introduce a sabiendas. Un ADR sin consecuencias negativas es sospechoso. |
| Alternativas descartadas | no | Opciones consideradas y por qué se descartaron. Útil cuando la pregunta "¿por qué no X?" es previsible. |
| Regla de interpretación | no | Aclaración normativa para evitar que la decisión se lea de forma laxa (ver ADR-0002 como ejemplo). |

## Ejemplo

```markdown
# ADR-0004: Límite de tamaño para eventos JSON individuales

- **Estado:** aceptada
- **Fecha:** 2026-08-18

## Contexto

Los eventos de `data/review-events/` y `data/task-events/` son archivos JSON
inmutables, uno por evento. Nada impide hoy que un evento incluya un adjunto
grande o un volcado de log completo, lo que degradaría `clusterlog sync` y
ensuciaría el historial Git con blobs pesados.

## Decisión

- Cada archivo de evento se limita a 64 KB.
- La CLI rechaza la escritura si el JSON serializado excede el límite,
  con un mensaje que indica el tamaño real y el máximo permitido.
- El límite se valida antes de firmar o comitear, nunca después.

## Consecuencias positivas

- El historial Git permanece liviano y rápido de clonar.
- `clusterlog sync` tiene un tiempo de ejecución más predecible.
- Fuerza a resumir en vez de adjuntar evidencia extensa.

## Consecuencias negativas

- Evidencia grande (logs completos, capturas) debe vivir fuera del evento
  y referenciarse por enlace o ticket.
- El límite es arbitrario y puede requerir ajuste con el uso real.

## Alternativas descartadas

- Sin límite: expone el repositorio a crecer sin control y a bloquear CI.
- Comprimir el evento: complica la lectura directa con herramientas
  estándar de Git, que es una propiedad central de ADR-0001.
```

## Relación con otros tipos de contenido

- **ADR**: por qué el software está diseñado así. Vive en `docs/adr/`, no se sincroniza al sitio.
- **Manual** (`content/manuales/`): cómo ejecutar un procedimiento repetible sobre el clúster o la bitácora. Sí se publica en el sitio.
- **Memoria**: qué se hizo, cuándo, por quién, con qué validación y reversión, en una sesión concreta. Sí se publica en el sitio.

Si al escribir una memoria o un manual surge una decisión de diseño reutilizable que no está documentada, es señal de que falta un ADR — no de que deba explicarse de nuevo cada vez.
