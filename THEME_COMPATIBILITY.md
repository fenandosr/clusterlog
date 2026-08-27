# Compatibilidad con `zola.386`

## Objetivo

Conservar la estética retro y responsiva de `lopes/zola.386` sin acoplar la lógica del producto a plantillas históricas del tema.

## Revisión fijada

El script `scripts/bootstrap-theme.sh` usa por defecto:

```text
d9c9726435d1347341f2171f9aa751291f1db93d
```

Puede cambiarse de forma explícita:

```bash
ZOLA386_REF=<commit> ./scripts/bootstrap-theme.sh
```

Para una copia local u offline:

```bash
ZOLA386_SOURCE=/ruta/a/zola.386 ./scripts/bootstrap-theme.sh
```

## Qué se importa

- `sass/`
- `static/`
- `theme.toml`
- licencia y README upstream, cuando existen

## Qué no se importa

Las plantillas upstream no se copian. El proyecto mantiene plantillas locales en `templates/` para:

- usar el contrato actual de Zola/Tera;
- cargar `data/generated/site_state.json`;
- presentar memorias, proyectos, tareas y revisión;
- controlar navegación, búsqueda y accesibilidad;
- evitar que una actualización visual cambie la semántica de datos.

Zola da precedencia a las plantillas locales sobre las del tema, por lo que se conserva el tema como fuente visual y no como capa de aplicación.

## Actualización del tema

1. Seleccione un commit, no una rama flotante.
2. Ejecute el script con `ZOLA386_REF`.
3. Revise visualmente todas las secciones.
4. Ejecute:

   ```bash
   go test ./...
   go vet ./...
   make sync
   clusterlog --root . validate --require-zola
   zola build
   ```

5. Compare HTML, búsqueda y responsive.
6. Registre la nueva revisión en un commit firmado.

## Búsqueda en español

El proyecto genera `search_index.es.json` con `elasticlunr_json`, pero no carga el motor Elasticlunr en el navegador. `static/js/ops-search.js` lee los documentos del índice y hace una coincidencia local normalizada, sin depender de stemmers externos ni de una CDN. Para el tamaño esperado del MVP, esto favorece portabilidad y funcionamiento offline.

## Zola fijado

La CI usa Zola `0.23.3`. Actualizar Zola requiere una pull request separada que incluya:

- revisión de notas de migración;
- `zola check`;
- construcción completa;
- prueba de búsqueda;
- prueba visual de portada, secciones, páginas, taxonomías y revisión.

## Licencias

La licencia del tema se conserva dentro de `themes/zola.386/LICENSE` cuando el script lo prepara. La licencia del proyecto no sustituye ni elimina las obligaciones del componente upstream.
