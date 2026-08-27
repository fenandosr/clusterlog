#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REF="${ZOLA386_REF:-d9c9726435d1347341f2171f9aa751291f1db93d}"
DEST="$ROOT/themes/zola.386"
SOURCE="${ZOLA386_SOURCE:-}"
TMP=""

cleanup() {
  if [[ -n "$TMP" && -d "$TMP" ]]; then
    rm -rf "$TMP"
  fi
}
trap cleanup EXIT

if [[ -z "$SOURCE" ]]; then
  TMP="$(mktemp -d)"
  SOURCE="$TMP/source"
  git init -q "$SOURCE"
  git -C "$SOURCE" remote add origin https://github.com/lopes/zola.386.git
  git -C "$SOURCE" fetch -q --depth 1 origin "$REF"
  git -C "$SOURCE" checkout -q --detach FETCH_HEAD
else
  SOURCE="$(cd "$SOURCE" && pwd)"
fi

for required in sass static theme.toml; do
  if [[ ! -e "$SOURCE/$required" ]]; then
    echo "Falta $required en el tema fuente: $SOURCE" >&2
    exit 1
  fi
done

rm -rf "$DEST"
mkdir -p "$DEST"
mkdir -p "$DEST/templates"
cp -R "$SOURCE/sass" "$DEST/sass"
cp -R "$SOURCE/static" "$DEST/static"
cp "$SOURCE/theme.toml" "$DEST/theme.toml"
[[ -f "$SOURCE/LICENSE" ]] && cp "$SOURCE/LICENSE" "$DEST/LICENSE"
[[ -f "$SOURCE/README.md" ]] && cp "$SOURCE/README.md" "$DEST/UPSTREAM_README.md"
printf '%s\n' "$REF" > "$DEST/.upstream-revision"

# Ninguna plantilla local usa una sola clase de Bootstrap (auditado: cero
# coincidencias de .btn/.navbar/.row/.span*/.modal/etc.), así que el
# site.scss original del tema (que importa el bundle de Bootstrap 2 de
# ~103 KB) se reemplaza por un reset mínimo propio que cubre lo que
# ops.css sí necesita: el fondo negro de <code>/<pre> y unos resets base.
# static/ tampoco lo referencia nada local, así que se descarta entero.
cat > "$DEST/sass/site.scss" <<'SCSS'
// Reset propio: reemplaza el bundle de Bootstrap 2 del tema, que ninguna
// plantilla local usaba (auditado). Cubre solo lo que ops.css necesita.
*, *::before, *::after { box-sizing: border-box; }
html { -webkit-text-size-adjust: 100%; }
body { margin: 0; }
h1, h2, h3, h4, h5, h6, p, figure, blockquote, dl, dd { margin: 0 0 1em; }
ul, ol { margin: 0 0 1em; padding: 0; list-style: none; }
img, svg, video { max-width: 100%; height: auto; display: block; }
table { border-collapse: collapse; width: 100%; }
hr { border: 0; border-top: 1px solid currentColor; }
a { text-decoration: underline; }
:focus-visible { outline: 2px solid currentColor; outline-offset: 2px; }

code, pre {
  padding: 0;
  color: #ffffff;
  background-color: #000000;
  border-radius: 0;
}
pre {
  padding: 8.5px;
  white-space: pre-wrap;
}
pre code {
  padding: 0;
  color: inherit;
  background-color: transparent;
}
SCSS
rm -rf "$DEST/static"
mkdir -p "$DEST/static"

cat <<MSG
Tema zola.386 preparado en themes/zola.386
Revisión fijada: $REF
Se conservaron Sass y archivos estáticos; las plantillas locales modernas tienen precedencia.
site.scss reemplazado por un reset propio (~1 KB) en vez del bundle de Bootstrap 2 (~103 KB); static/ podado a vacío.
MSG
