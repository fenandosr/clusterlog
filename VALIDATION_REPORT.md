# Informe de validación del MVP

**Producto:** Bitácora Operativa del Clúster  
**Versión:** 0.1.0  
**Fecha de corte:** 2026-08-12, zona `America/Mexico_City`  
**Plataforma local:** Linux `x86_64`, Go 1.23.2, Git 2.47.3, Node.js 22.16.0

## Resultado ejecutivo

El prototipo de la CLI quedó compilado y todas las comprobaciones ejecutables en el entorno local terminaron correctamente. El flujo integral validó el ciclo administrador → memoria → revisión por dos personas → invalidación por edición → nueva revisión → proyecto → tarea → tiempo → cierre.

La construcción real del sitio con Zola y una firma SSH válida no pudieron ejecutarse en este entorno porque no estaban instalados `zola` ni `ssh-keygen`. Ambos controles están definidos en la canalización de GitHub Actions y deben ejecutarse antes de promover el repositorio a piloto.

## Comprobaciones ejecutadas

| Comprobación | Comando | Resultado |
|---|---|---|
| Formato Go | `test -z "$(gofmt -l cmd internal)"` | Correcto |
| Pruebas Go | `go test ./...` | Correcto |
| Detector de carreras | `go test -race ./...` | Correcto |
| Cobertura | `go test -cover ./...` | 32.9 % en `internal/clusterlog` |
| Análisis estático | `go vet ./...` | Correcto |
| Compilación | `make build` | Correcto |
| Flujo integral | `make e2e` | Correcto |
| Contratos rápidos | `make smoke` | Correcto |
| JavaScript | `node --check static/js/ops-search.js` | Correcto |
| Scripts shell | `bash -n scripts/*.sh` | Correcto |
| Proyección | `clusterlog sync` | Sin diferencias |
| Reproducibilidad | `clusterlog sync --check` | Correcto |
| Validación local | `clusterlog validate --skip-zola` | Válido, sin errores ni advertencias |
| Firma no autorizada | prueba negativa de `verify-signatures-ci.sh` | Rechazada correctamente |
| Bootstrap sin administradores | `verify-signatures-ci.sh` | Omisión controlada correcta |

## Escenario E2E validado

1. Registra el primer administrador como `owner` y un segundo administrador.
2. Crea una memoria con política `all-active`.
3. Comprueba que ambos administradores aparecen como revisores requeridos.
4. Registra cada revisión en un archivo de evento independiente.
5. Comprueba que la memoria desaparece de la cola al completar las revisiones.
6. Modifica la memoria, recalcula su hash y comprueba que las revisiones anteriores dejan de aplicar.
7. Registra nuevamente las dos revisiones.
8. Crea un proyecto activo y una tarea vinculada.
9. Registra inicio de tiempo, ajuste manual y cierre de la tarea.
10. Comprueba la proyección final, la independencia de eventos y la regeneración determinista.

## Estado inicial de demostración

La proyección incluida contiene:

- `0` administradores activos;
- `0` memorias pendientes de revisión;
- `1` tarea abierta de demostración;
- `1` proyecto planeado de demostración.

La memoria de ejemplo declara `review_required = false` para no contaminar la primera reunión real.

## Binario entregado

- Plataforma: Linux `amd64`.
- Tipo: ejecutable ELF de 64 bits, enlazado estáticamente.
- Ruta: `bin/clusterlog`.
- SHA-256 al cierre de esta validación: se publica en `SHA256SUMS`.

## Comprobaciones delegadas a CI

La canalización `.github/workflows/ci.yml` fija Zola 0.23.3 y ejecuta:

- descarga del tema fijado a una revisión concreta;
- validación con `--require-zola`;
- construcción completa con `zola build`;
- verificación de todos los commits de un pull request contra `config/allowed_signers` cuando ya existen administradores activos.

Estas comprobaciones están configuradas, pero este informe no afirma que GitHub Actions ya haya ejecutado una corrida. Debe conservarse como requisito de aceptación del primer pull request.

## Comprobaciones pendientes antes del piloto

1. Ejecutar la CI en el repositorio remoto y conservar una corrida verde.
2. Crear y verificar al menos un commit firmado con una llave SSH real autorizada.
3. Revisar el sitio generado en escritorio y móvil detrás del mecanismo de acceso interno elegido.
4. Probar el flujo con al menos dos administradores reales y una memoria no demostrativa.
5. Subir la cobertura de lógica crítica, especialmente errores de Git, validación de contenido y colisiones de archivos.
6. Ejecutar revisión de amenazas y acordar protección de rama, propietarios de código y retención.

## Repetición local

```bash
make build
make test
make vet
make e2e
make smoke
node --check static/js/ops-search.js
for f in scripts/*.sh; do bash -n "$f"; done
./bin/clusterlog --root . --json sync --check
./bin/clusterlog --root . --json validate --skip-zola
```

En un equipo con Zola, OpenSSH y acceso al tema:

```bash
make theme
make site
```
