# ADR-0006: Embeber las plantillas Tera en el binario del motor

- **Estado:** aceptada
- **Fecha:** 2026-08-27

## Contexto

`bootstrap-instance` (ADR-0005) genera el esqueleto de una instancia nueva,
pero dejaba fuera `templates/`: una instancia necesitaba, aparte, descargar
`clusterlog-templates_vX.Y.Z.tar.gz` desde el release del motor (o clonar el
repo del motor directamente) antes de poder correr `zola build`. Eso son un
asset de release más, un checksum más que verificar, y una ruta real de
desincronización: nada impedía que el binario fuera de una versión y las
plantillas descargadas de otra.

## Decisión

Las plantillas Tera propias de clusterlog (`internal/clusterlog/scaffold/templates/`,
14 archivos `.html`) se embeben en el binario del motor con `go:embed`, igual
que ya se hacía con el contenido genérico de `bootstrap-instance`. Un
comando nuevo, `clusterlog templates extract [--output DIR]`, las escribe en
disco (por defecto `<root>/templates`) sin ninguna llamada de red.
`bootstrap-instance` lo invoca automáticamente al crear una instancia nueva.

El asset de release `clusterlog-templates_vX.Y.Z.tar.gz` deja de publicarse
a partir de esta versión.

## Consecuencias positivas

- Binario y plantillas son, por construcción, siempre de la misma versión —
  no hay forma de que se desincronicen.
- Una instancia nueva puede construir con Zola inmediatamente después de
  `bootstrap-instance`, sin un paso de red aparte para esto.
- Actualizar plantillas en una instancia existente es `clusterlog templates
  extract` con el binario nuevo ya descargado — no un fetch, checksum y
  extracción aparte.
- Un asset de release menos que mantener, firmar y documentar.

## Consecuencias negativas

- El binario crece un poco (14 archivos HTML, unos pocos KB comprimidos —
  no significativo, pero es un costo real, no cero).
- Ya no es posible usar unas plantillas de una versión del motor con un
  binario de otra versión sin recompilar — antes, en teoría, alguien podía
  mezclar y combinar tarballs de plantillas y binarios de distintas
  versiones (aunque nada lo garantizaba ni lo probaba).
- Esto no resuelve, ni pretende resolver, si una instancia puede
  personalizar una plantilla puntual sin forkear todo `templates/` — sigue
  siendo una decisión de gobierno abierta (ver ADR-0005, "Trabajo futuro").

## Alternativas descartadas

- **Seguir publicando el tarball de plantillas, pero que `bootstrap-instance`
  lo descargue automáticamente.** Resuelve la fricción de UX pero no la
  desincronización real entre binario y plantillas, y mantiene un asset de
  release y una llamada de red que el embed elimina de raíz.
- **Embeber también el tema (`zola.386`) en el binario.** Descartado: el
  tema es una dependencia de un repositorio de terceros
  (`github.com/lopes/zola.386`), con su propia licencia y ciclo de
  actualización, pinneada a un commit externo. Embeberlo mezclaría código
  ajeno dentro del binario de clusterlog y complicaría el pin (¿qué versión
  del motor corresponde a qué commit del tema?). `make theme` /
  `scripts/bootstrap-theme.sh` siguen siendo el mecanismo correcto para algo
  que no es código propio.

## Trabajo futuro

- Si en algún momento se decide permitir sobreescribir una plantilla puntual
  por instancia, este ADR es el punto de partida técnico: la extracción ya
  deja los archivos en disco como texto plano editable, así que un merge
  "vendored + overrides" no requeriría tocar el mecanismo de embed en sí,
  sólo agregar un paso de fusión antes de `zola build`.
