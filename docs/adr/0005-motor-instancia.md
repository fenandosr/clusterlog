# ADR-0005: Separar el motor de cada instancia de deployment

- **Estado:** aceptada
- **Fecha:** 2026-08-27

## Contexto

El repositorio original de `clusterlog` (nombre de trabajo interno:
`bitacora-cluster-mvp`) mezclaba en un solo árbol la herramienta (CLI en Go,
plantillas Tera, tema fijado) con el contenido de un deployment concreto
(memorias, manuales, `data/admins.json`, `config/allowed_signers`, todos con
detalles reales de esa infraestructura). Clonar la herramienta para
administrar un clúster distinto obligaba a cargar con el historial y el
contenido del primero, o a mantener forks divergentes del código.

## Decisión

Este repositorio (`clusterlog`) contiene únicamente el motor: `cmd/`,
`internal/`, `templates/`, el mecanismo de tema fijado, scripts de calidad,
documentación de la herramienta y la CLI para generar instancias nuevas
(`clusterlog bootstrap-instance`). No contiene memorias, manuales ni datos de
administradores reales.

`content/`, `data/` y `config/` que sí existen en este commit son un
**fixture genérico**, generado por el propio `bootstrap-instance`, no una
instancia real: existen únicamente para que `make smoke`, `make e2e` y
`make site` tengan algo contra qué construir dentro de este mismo
repositorio. Ningún administrador, memoria o dato real de ningún deployment
vive aquí.

## Por qué no se preservó el historial de Git

El repositorio original tiene commits firmados por SSH que
`clusterlog verify commit` verifica contra `config/allowed_signers`.
Cualquier técnica de reescritura de historia (`git filter-repo`, un subtree
split que reescribe commits) invalida esas firmas SSH. Para no arriesgar esa
cadena de confianza, este repositorio motor **nace sin historia compartida**:
es un `git init` limpio con los archivos de motor copiados tal como estaban
en el commit de origen. El repositorio original conserva su historia intacta
y se convierte en una instancia agregando un commit nuevo, no reescribiendo
los existentes (ver el trabajo de fase 3 pendiente en el repo de instancia).

## Relación con `bootstrap-instance`

`bootstrap-instance` (agregado antes de esta separación, en el mismo
repositorio cuando todavía era uno solo) es el punto de unión entre motor e
instancia: genera exactamente el árbol mínimo que una instancia necesita
(`zola.toml`, `content/<sección>/_index.md`, `data/admins.json` y
`config/allowed_signers` vacíos, `.clusterlog-version`). El fixture de este
mismo repositorio se generó con ese comando, no a mano — es la primera
prueba real de que el comando produce algo que compila, valida y construye.

## Versionado

`Version` se fija en build (`-ldflags -X .../clusterlog.Version=...`, ver
`Makefile`). Este release es `v0.1.0`: primera versión con
`bootstrap-instance` y la integración de Runme. Una instancia fija la
versión del motor que usa en su propio `.clusterlog-version`.

## Consecuencias positivas

- El motor es clonable y probable de forma aislada, sin ningún dato de
  ningún deployment real.
- Una instancia nueva se crea con un comando, no copiando y editando a mano
  un scaffold.
- El fixture generado por `bootstrap-instance` sirve como prueba viva de que
  el comando produce un proyecto Zola+clusterlog funcional.

## Consecuencias negativas / limitaciones

- **`templates/` y el tema siguen requiriendo un checkout o un tarball del
  motor** (`clusterlog-templates_v0.1.0.tar.gz` en este release) —
  `bootstrap-instance` no los empaqueta dentro de la instancia todavía; una
  instancia real necesita `make theme`/`scripts/bootstrap-theme.sh` apuntando
  a este repositorio o al tarball de plantillas del release.
- No existe todavía un workflow de CI de referencia para una instancia real
  que descargue el binario del motor por versión fijada — se agrega cuando
  el motor efectivamente publique releases en un remoto (este release, por
  ahora, es sólo local).
- `RUNBOOK.md`/`runbooks/clusterlog-development.md` de este repositorio
  siguen operando contra el fixture genérico, no contra un deployment real;
  eso es intencional, pero quien lea `RUNBOOK.md` de un fork de este repo
  debe saber que "sync/validate/site" aquí son de prueba, no de producción.

## Trabajo futuro

- Publicar este repositorio y el release `v0.1.0` en un remoto (pendiente de
  decisión de dónde vive y su visibilidad).
- Convertir el repositorio de instancia original en una instancia real
  (fase 3 del plan motor/instancia): un solo commit nuevo que retira los
  archivos de motor y agrega `.clusterlog-version`, sin reescribir historia
  existente.
- Empaquetar `templates/` y el pin del tema dentro de lo que
  `bootstrap-instance` deja listo para usar, para que una instancia nueva no
  necesite un checkout separado del motor sólo para construir con Zola.
