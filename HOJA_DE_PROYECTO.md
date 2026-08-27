# Hoja de proyecto: Bitácora Operativa del Clúster

**Versión:** 0.1.0  
**Fecha base:** 12 de agosto de 2026  
**Nombre de trabajo:** Bitácora Operativa del Clúster  
**CLI:** `clusterlog`  
**Tipo de producto:** sitio interno estático + CLI de escritura + repositorio Git auditable

## 1. Resumen ejecutivo

El equipo de soporte necesita un lugar único para comprender **cómo está construido el clúster, cómo se opera, qué cambió, qué falta por hacer y qué debe revisar el resto del equipo**. La propuesta convierte un repositorio Git en la fuente de verdad y publica una proyección navegable mediante Zola, con la apariencia del tema `zola.386`.

La escritura no ocurre en el navegador. Administradores y agentes de programación usan una CLI no interactiva que:

- identifica al actor por la huella de su llave pública SSH;
- valida el contenido contra contratos predecibles;
- crea Markdown y eventos JSON pequeños;
- actualiza la proyección que consume Zola;
- y, cuando se solicita, deja todo en un commit Git firmado por SSH.

La página **Revisión** será la agenda inicial de las reuniones semanales. Muestra las memorias aún no revisadas por todos los administradores requeridos y deja de mostrarlas en cuanto la versión actual obtiene todas las confirmaciones.

## 2. Problema que resuelve

Hoy, el conocimiento operativo suele quedar fragmentado entre terminales, chats, tickets, wikis, memoria personal y commits técnicos. Esto produce cinco fallas:

1. No hay una narrativa cronológica de las intervenciones.
2. Los manuales y la arquitectura se desactualizan sin que exista una señal clara.
3. El equipo no sabe qué cambios ya fueron comprendidos por todos.
4. Las tareas y proyectos pierden contexto técnico y tiempo real invertido.
5. Los agentes de IA pueden ayudar a documentar, pero no disponen de un contrato seguro, determinista y atribuible.

## 3. Visión del producto

> Después de cada sesión de administración, el equipo debe poder reconstruir qué se intentó, qué se cambió, cómo se validó, cómo se revierte, quién lo ejecutó y quién ya lo revisó, sin depender de la memoria individual.

## 4. Objetivos

### Objetivos del MVP

- Centralizar documentación técnica, runbooks, memorias, proyectos y tareas.
- Hacer que registrar una memoria tome menos fricción que escribir una nota libre en otro sistema.
- Proporcionar a Claude Code y agentes similares un contrato JSON estable y no interactivo.
- Atribuir las escrituras a una identidad registrada por llave SSH.
- Hacer auditable la aprobación mediante commits SSH firmados.
- Construir una cola semanal de revisión reproducible y sin estado oculto en el navegador.
- Medir tiempo real de tareas sin introducir una base de datos.
- Mantener la solución portable: archivos legibles, Git y un generador estático.

### Objetivos posteriores al MVP

- Integración con CI corporativa, SSO, tickets y notificaciones.
- Importación de inventario o CMDB.
- Reportes de SLO, incidentes, cambios por sistema y deuda operativa.
- API remota controlada, sólo cuando el trabajo sin conexión y el modelo Git resulten insuficientes.

### Fuera de alcance del MVP

- Edición web multiusuario.
- Chat interno, comentarios en tiempo real o presencia.
- Almacenamiento de secretos, credenciales o volcados sensibles.
- Sustitución de monitoreo, ticketing, CMDB o gestión de incidentes.
- Identidad basada únicamente en el comentario textual de una llave.
- Firmar en nombre del usuario sin acceso autorizado a su llave privada o agente SSH.

## 5. Usuarios y responsabilidades

### Administrador del clúster

Crea memorias, documentación, manuales, tareas y proyectos; registra tiempo; revisa cambios ajenos; conserva su llave privada bajo su control.

### Owner de la bitácora

Da de alta administradores, mantiene la política de revisión, protege la rama principal y supervisa la validez de firmas.

### Agente de IA

Prepara contenido estructurado en nombre del administrador que inició la sesión. Usa la CLI y su salida JSON; no modifica archivos derivados a mano; no inventa resultados; no expone secretos.

### Lector técnico

Consulta el sitio, busca por texto, sistema o etiqueta y usa manuales y memorias como contexto operativo.

## 6. Arquitectura de información

| Sección | Propósito | Unidad principal | Cadencia esperada |
|---|---|---|---|
| Documentación | Explicar estado y diseño vigentes | documento técnico | cuando cambia la arquitectura |
| Manuales | Describir procedimientos repetibles | runbook/manual | cuando cambia un procedimiento |
| Memorias | Narrar una intervención concreta | entrada cronológica | después de cada sesión o cambio |
| Proyectos | Coordinar un resultado de varias tareas | proyecto | semanal/mensual |
| Tareas | Registrar trabajo accionable y tiempo | tarea | diaria |
| Revisión | Alinear al equipo sobre cambios recientes | proyección de pendientes | semanal |

### Taxonomías comunes

- `tags`: tema transversal, por ejemplo `firmware`, `backup`, `seguridad`.
- `systems`: componente real afectado, por ejemplo `etcd`, `spine-01`, `ceph`.
- `categories`: sección funcional.

### Convención de identificadores

- Documentación: `DOC-...`
- Manuales: `RUN-...`
- Memorias: `MEM-...`
- Proyectos: `PRJ-...`
- Tareas: `TSK-...`
- Eventos: identificadores únicos propios, almacenados en archivos independientes.

## 7. Modelo de datos

### Memoria operativa

Campos mínimos o recomendados:

- ID, título, resumen y fecha.
- autor registrado y huella SSH empleada;
- agente que ayudó a redactar, cuando aplique;
- sistemas, etiquetas, ticket y nivel de riesgo;
- contexto, cambios, comandos redactados, validaciones y reversión;
- inicio, fin y duración;
- requisito y política de revisión;
- conjunto capturado de revisores;
- hash SHA-256 de la versión actual, calculado por la CLI.

### Evento de revisión

Cada confirmación vive en `data/review-events/<evento>.json` e incluye:

- memoria;
- administrador y huella;
- fecha;
- hash exacto del contenido revisado;
- nota opcional.

No existe un booleano mutable `reviewed=true`. La revisión se deriva de eventos válidos para el hash actual.

### Tarea

La definición Markdown contiene título, objetivo, prioridad, responsables, proyecto, estimación y fecha. Los cambios de ejecución viven como eventos independientes:

- `start`;
- `stop`;
- `adjust`;
- `status`.

El tiempo real y el estado efectivo se proyectan al sincronizar.

### Proyecto

Incluye objetivo medible, responsables, estado, fecha objetivo, alcance, fuera de alcance, hitos y criterios de éxito.

## 8. Semántica de revisión

La revisión debe responder: **“¿qué administradores confirmaron haber comprendido esta versión exacta?”**

Reglas:

1. Al crear una memoria se captura el conjunto de revisores aplicable en ese momento.
2. La política puede requerir a todos los activos o a todos excepto el autor.
3. Cada evento registra el hash SHA-256 actual de la memoria.
4. Sólo cuentan eventos cuyo hash coincide con la versión actual.
5. Una edición posterior cambia el hash y vuelve a abrir la memoria automáticamente.
6. Dar de alta a un administrador en el futuro no reabre memorias históricas, porque cada memoria conserva su conjunto capturado.
7. Una memoria sale de `/revision/` cuando no falta ningún revisor.
8. La firma del commit, no la mera presencia de la huella en JSON, aporta la prueba criptográfica de autoría.

## 9. Identidad y confianza

### Resolución de identidad

La CLI busca una llave local en este orden general:

1. `--ssh-key`;
2. `CLUSTERLOG_SSH_KEY`;
3. `git config user.signingkey` cuando es una ruta;
4. llaves públicas comunes dentro de `~/.ssh`.

Calcula la huella OpenSSH SHA-256 de la llave pública y la compara con `data/admins.json`.

### Límite importante

Conocer una llave pública sólo identifica un registro; no demuestra posesión de la privada. Por ello, el flujo de producción debe exigir:

- `--commit` en operaciones de escritura;
- commits SSH firmados;
- `config/allowed_signers` generado desde administradores activos;
- verificación en CI;
- protección de rama y revisión de cambios en archivos de identidad.

## 10. Requisitos funcionales

### RF-01. Navegación central

El sitio debe exponer las seis secciones principales y búsqueda global.

**Aceptación:** desde cualquier página se llega a cada sección en un paso.

### RF-02. Creación de memorias por CLI

La CLI debe aceptar flags humanos o JSON por archivo/`stdin`.

**Aceptación:** `memory create --from-json -` produce una página válida, un ID y una proyección actualizada.

### RF-03. Atribución

Toda escritura debe guardar el ID del administrador y la huella resuelta.

**Aceptación:** una llave no registrada falla con código de identidad y no escribe archivos.

### RF-04. Revisión por todos

La página de revisión debe mostrar faltantes y desaparecer la memoria al completar el conjunto requerido.

**Aceptación:** con dos revisores, la entrada sigue visible tras una confirmación y desaparece tras la segunda.

### RF-05. Invalidación por edición

Una revisión anterior no debe contar si cambia la memoria.

**Aceptación:** modificar el cuerpo y ejecutar `sync` vuelve a mostrar la entrada.

### RF-06. Tareas y tiempo

Debe ser posible crear, iniciar, detener, ajustar y completar una tarea.

**Aceptación:** el tiempo real es la suma reproducible de eventos y no puede ser negativo.

### RF-07. Proyectos

Debe ser posible crear y listar proyectos con objetivo y estado.

**Aceptación:** las tareas pueden enlazar un proyecto existente y la validación rechaza referencias inexistentes.

### RF-08. Operación por agentes

Debe existir salida JSON estable, esquemas JSON y `--dry-run`.

**Aceptación:** un agente puede descubrir el esquema, validar y escribir sin prompts ni parsing de texto humano.

### RF-09. Validación

La CLI debe revisar estructura, referencias, eventos, firmas autorizables, secretos probables, proyección y Zola.

**Aceptación:** `validate --require-zola` falla ante cualquier error bloqueante.

### RF-10. Auditoría Git

Una operación con `--commit` debe añadir sólo los archivos que creó o modificó y producir un commit SSH firmado.

**Aceptación:** `verify commit <sha>` confirma la firma contra `config/allowed_signers`.

## 11. Requisitos no funcionales

- **Portabilidad:** Markdown, JSON y Git deben seguir siendo legibles sin la CLI.
- **Determinismo:** la misma fuente genera la misma proyección, salvo la marca informativa `generated_at` controlada por la sincronización.
- **Concurrencia Git:** cada revisión y evento de tarea ocupa un archivo distinto para minimizar conflictos.
- **Seguridad:** no almacenar secretos; sitio no indexable y desplegado en red privada.
- **Accesibilidad:** navegación semántica, uso por teclado y contenido legible sin JavaScript salvo la búsqueda.
- **Mantenibilidad:** plantillas locales separadas del tema visual fijado.
- **Observabilidad:** errores con `kind`, detalles y código de salida estable.
- **Recuperación:** toda proyección puede eliminarse y regenerarse desde la fuente.

## 12. Diseño de la CLI

Forma general:

```text
clusterlog [--root DIR] [--json] [--ssh-key PATH] [--commit] [--dry-run] COMANDO
```

Principios:

- flags globales antes del comando;
- sin prompts interactivos;
- operaciones de creación atómicas;
- `stdin` con `--from-json -` o `--body-file -`;
- respuesta JSON envuelta en `{ok, command, data, warnings}`;
- errores JSON con `kind`, `message`, `details` y código de proceso;
- no sobrescribir eventos existentes;
- comandos idempotentes cuando existe un estado equivalente, por ejemplo revisar dos veces el mismo hash.

La referencia exhaustiva está en `CLI.md`.

## 13. Arquitectura técnica

```text
┌───────────────────────────────┐
│ Humano / Claude Code / agente │
└───────────────┬───────────────┘
                │ JSON o flags
                v
┌───────────────────────────────┐
│ clusterlog (Go)               │
│ identidad · validación · Git  │
└───────┬──────────────┬────────┘
        │              │
        │ Markdown     │ eventos JSON inmutables
        v              v
┌───────────────────────────────────────────────┐
│ Repositorio Git                               │
│ content/ · data/ · config/allowed_signers     │
└───────────────────┬───────────────────────────┘
                    │ sync
                    v
┌───────────────────────────────────────────────┐
│ data/generated/site_state.json                │
│ cola · métricas · tareas · proyectos · memoria│
└───────────────────┬───────────────────────────┘
                    │ load_data
                    v
┌───────────────────────────────────────────────┐
│ Zola + recursos visuales de zola.386          │
└───────────────────────────────────────────────┘
```

### Decisión sobre estado compartido

No se usa `localStorage`, una cookie ni un formulario estático para las revisiones. Esos mecanismos no representan un consenso compartido ni son auditables. El estado se conserva en Git y Zola sólo lo presenta.

### Decisión sobre el tema

Se fijan Sass y archivos estáticos de `zola.386` a una revisión concreta. Las plantillas son locales, porque el tema original precede a cambios mayores de Zola/Tera y el proyecto necesita estructuras específicas para revisión, tareas y proyectos.

## 14. Alcance del MVP entregado

### Incluido

- estructura Zola y navegación;
- tema visual preparado mediante script;
- CLI Go funcional;
- contratos JSON y ejemplos;
- registro de administradores;
- memoria/revisión por hash;
- tareas y eventos de tiempo;
- proyectos;
- proyección para el sitio;
- validación y escaneo de secretos probables;
- commits y verificación SSH;
- pruebas Go;
- CI para Zola 0.23.3;
- documentación de seguridad y operación.

### Pendiente para producción

- reemplazar dominio de ejemplo;
- decidir hosting interno y autenticación;
- configurar protección de rama y verificación obligatoria de firmas;
- poblar administradores reales;
- revisar taxonomía de sistemas;
- probar la canalización en el repositorio corporativo;
- definir retención y clasificación de información;
- integrar respaldos del repositorio.

## 15. Fases recomendadas

### Fase 0 — Alineación y seguridad

- Aprobar nombres, secciones y política de revisión.
- Definir qué información está prohibida.
- Seleccionar repositorio y hosting interno.
- Registrar owners iniciales.

**Salida:** decisiones aprobadas y threat model aceptado.

### Fase 1 — Piloto técnico

- Desplegar el prototipo.
- Incorporar de dos a cuatro administradores.
- Importar cinco manuales y diez memorias recientes.
- Ejecutar dos reuniones semanales usando `/revision/`.

**Salida:** retroalimentación sobre fricción, campos y cola.

### Fase 2 — Adopción operativa

- Añadir plantilla a las instrucciones de Claude Code.
- Exigir `validate` y firmas en pull requests.
- Migrar proyectos y tareas activas.
- Establecer responsables de calidad documental.

**Salida:** bitácora como flujo normal, no como actividad extraordinaria.

### Fase 3 — Integraciones

- Enlaces automáticos a tickets.
- Notificaciones de revisión.
- Métricas y reportes de mantenimiento.
- Evaluar una API sólo con casos concretos que Git no resuelva.

## 16. Riesgos y mitigaciones

| Riesgo | Consecuencia | Mitigación |
|---|---|---|
| El agente incluye un secreto | exposición en Git y sitio | contrato explícito, escáner heurístico, revisión humana y herramientas corporativas de secret scanning |
| Se suplanta una huella en un JSON | falsa atribución aparente | exigir commit SSH firmado y verificarlo en CI |
| Dos personas editan el mismo estado | conflictos de merge | un archivo por evento y Markdown por entidad |
| Una memoria cambia después de revisarse | consenso obsoleto | hash de contenido en cada revisión |
| Un nuevo admin reabre todo el histórico | cola inmanejable | captura de revisores por memoria |
| El tema deja de compilar | bloqueo de publicación | revisión fijada, plantillas locales y build en CI |
| El sitio se expone a Internet | fuga de conocimiento operativo | VPN/SSO, `robots.txt`, `noindex`, mínimo privilegio |
| La documentación se vuelve burocrática | baja adopción | JSON preparado por agente, campos mínimos y plantillas breves |
| Git crece con muchos eventos | mantenimiento | eventos pequeños, política de archivo posterior y métricas reales antes de rediseñar |

## 17. Métricas de éxito del piloto

- Porcentaje de cambios relevantes con memoria dentro de 24 horas.
- Tiempo mediano para registrar una memoria.
- Porcentaje de memorias revisadas antes de la siguiente reunión.
- Antigüedad máxima de un pendiente de revisión.
- Número de manuales actualizados a partir de hallazgos en memorias.
- Diferencia entre tiempo estimado y real por categoría de tareas.
- Errores de validación o secretos bloqueados antes de llegar a la rama principal.

Las métricas deben usarse para mejorar el sistema, no para evaluar individualmente el desempeño sin contexto.

## 18. Criterios de aceptación del MVP

El MVP se considera aceptado cuando:

1. La CLI compila y todas las pruebas automatizadas pasan.
2. Zola construye el sitio con el tema preparado y las plantillas locales.
3. Un owner puede registrar a un segundo administrador.
4. Una memoria creada por JSON aparece en Memorias y Revisión.
5. Cada administrador puede revisarla desde su propia llave.
6. La memoria desaparece al completar el conjunto de revisores.
7. Editar la memoria vuelve a abrir su revisión.
8. Una tarea registra tiempo y puede completarse sin producir minutos negativos.
9. Una tarea puede enlazarse a un proyecto existente.
10. La validación detecta referencias inválidas, temporizadores inconsistentes y secretos probables.
11. Los commits creados con `--commit` se verifican contra `allowed_signers`.
12. La rama principal exige CI y firmas según la política del repositorio.
13. El sitio está accesible sólo en el entorno interno acordado.

## 19. Definición de terminado para cada cambio

Un cambio está terminado cuando:

- tiene pruebas o una justificación explícita de por qué no aplican;
- no introduce secretos ni datos personales innecesarios;
- actualiza esquema, documentación y ejemplo cuando cambia un contrato;
- ejecuta `go test ./...`, `go vet ./...`, `clusterlog sync --check` y `clusterlog validate`;
- construye con Zola en CI;
- conserva compatibilidad de salida JSON o incrementa versión con migración;
- queda en un commit firmado y revisable.

## 20. Backlog inmediato priorizado

### P0 — antes del piloto

- Configurar dominio interno, hosting y acceso.
- Ejecutar CI en el repositorio real.
- Registrar owners y política de firmas.
- Revisar patrones del escáner de secretos con ejemplos del entorno.
- Crear los primeros manuales críticos: acceso, respaldo, restauración y escalamiento.

### P1 — durante el piloto

- Añadir edición controlada de tareas y proyectos mediante eventos o comandos específicos.
- Añadir desactivación/rotación de llaves de administradores.
- Añadir verificación de firma para un rango de commits en CI.
- Generar una vista “mis pendientes”.
- Incorporar relaciones entre memorias y documentación que debe actualizarse.

### P2 — después de validar adopción

- Notificaciones de revisión pendientes.
- Importación de tickets.
- Reportes de actividad por sistema y riesgo.
- Archivado por periodo y política de retención.
- API o servicio de escritura sólo si los flujos Git resultan insuficientes.

## 21. Preguntas de gobierno que el equipo debe cerrar

- ¿El autor debe revisar su propia memoria o se excluye por defecto?
- ¿Qué riesgo exige revisión síncrona antes del cambio y cuál permite revisión posterior?
- ¿Cuánto tiempo puede permanecer una memoria pendiente?
- ¿Quién puede registrar, desactivar o rotar llaves?
- ¿Qué datos de infraestructura están permitidos en el sitio?
- ¿Qué sistema conserva el ticket canónico y cuál sólo enlaza?
- ¿Cuándo una memoria obliga a actualizar un manual o documento de arquitectura?
- ¿Qué estados de proyecto y tarea necesita realmente el equipo?

Estas decisiones son configuración y proceso; no requieren convertir el MVP en una aplicación con base de datos.
