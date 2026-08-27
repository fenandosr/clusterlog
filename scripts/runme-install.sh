#!/usr/bin/env bash
# Instala una versión fijada de Runme (nunca "latest") verificando su
# checksum SHA-256 contra los valores publicados en el release oficial.
# No sustituye ni toca bin/clusterlog. No usa curl|sh.
set -euo pipefail

RUNME_VERSION="3.17.4"
INSTALL_DIR="${RUNME_INSTALL_DIR:-$HOME/.local/bin}"
BASE_URL="https://github.com/runmedev/runme/releases/download/v${RUNME_VERSION}"

# SHA-256 tomados del checksums.txt publicado en el release v3.17.4 y
# verificados de forma independiente (sha256sum) contra el tarball real
# durante la implementación de esta integración.
CHECKSUM_LINUX_X86_64="78af7ed33a84ff91a617a018886730038f10af8f73027451d322c21c417fed2c"
CHECKSUM_LINUX_ARM64="fd016b92115d7f77806c70619ee113dc951c49398863e54bc83eadb9d48665e4"
CHECKSUM_DARWIN_X86_64="7c69077ad7331725fa5224fab95562089a42c95baad42b130e1b0b3774ebc480"
CHECKSUM_DARWIN_ARM64="cba556395ca24ccee641ee2faa20cc5a4150826205371de354aafc4a4d830381"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Linux)
    case "$arch" in
      x86_64) asset="runme_linux_x86_64.tar.gz"; checksum="$CHECKSUM_LINUX_X86_64" ;;
      aarch64|arm64) asset="runme_linux_arm64.tar.gz"; checksum="$CHECKSUM_LINUX_ARM64" ;;
      *) echo "Arquitectura Linux no soportada por este script: $arch" >&2; exit 1 ;;
    esac
    ;;
  Darwin)
    case "$arch" in
      x86_64) asset="runme_darwin_x86_64.tar.gz"; checksum="$CHECKSUM_DARWIN_X86_64" ;;
      arm64) asset="runme_darwin_arm64.tar.gz"; checksum="$CHECKSUM_DARWIN_ARM64" ;;
      *) echo "Arquitectura macOS no soportada por este script: $arch" >&2; exit 1 ;;
    esac
    ;;
  *)
    echo "Sistema operativo no soportado por este script: $os" >&2
    echo "En Windows, descargue manualmente runme_windows_{x86_64,arm64}.zip desde" >&2
    echo "  https://github.com/runmedev/runme/releases/tag/v${RUNME_VERSION}" >&2
    echo "y verifique su checksum contra checksums.txt del mismo release." >&2
    exit 1
    ;;
esac

if command -v runme >/dev/null 2>&1; then
  current="$(runme --version 2>/dev/null | awk '{print $3}')"
  if [ "$current" = "$RUNME_VERSION" ]; then
    echo "runme $RUNME_VERSION ya está instalado en $(command -v runme); nada que hacer."
    exit 0
  fi
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "Descargando $asset (runme v${RUNME_VERSION})..."
curl -fsSL -o "$tmp_dir/$asset" "$BASE_URL/$asset"

echo "Verificando checksum SHA-256..."
computed="$(sha256sum "$tmp_dir/$asset" | awk '{print $1}')"
if [ "$computed" != "$checksum" ]; then
  echo "ERROR: checksum no coincide para $asset" >&2
  echo "  esperado: $checksum" >&2
  echo "  obtenido: $computed" >&2
  exit 1
fi
echo "Checksum verificado."

mkdir -p "$INSTALL_DIR"
tar xzf "$tmp_dir/$asset" -C "$tmp_dir" runme
install -m 0755 "$tmp_dir/runme" "$INSTALL_DIR/runme"

if [ -e "$INSTALL_DIR/clusterlog" ] || [ -e "$INSTALL_DIR/bin/clusterlog" ]; then
  echo "ERROR: se detectó un binario 'clusterlog' en $INSTALL_DIR; abortando para no sobrescribirlo." >&2
  exit 1
fi

echo "runme instalado en $INSTALL_DIR/runme"
"$INSTALL_DIR/runme" --version
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Agregue $INSTALL_DIR a su PATH para usar 'runme' directamente." ;;
esac
