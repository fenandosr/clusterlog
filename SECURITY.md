# Seguridad y modelo de confianza

## Alcance

Este repositorio contiene conocimiento operativo potencialmente sensible. Debe tratarse como un sistema interno aun cuando el resultado sea un sitio estático.

## Garantías que sí busca el diseño

- Atribución de contenido a un administrador registrado por huella de llave pública SSH.
- Prueba criptográfica de autoría cuando la operación queda en un commit SSH firmado y autorizado.
- Historial de cambios, revisiones y tiempo dentro de Git.
- Revisión ligada al hash exacto de una memoria.
- Separación entre fuente y proyección derivada.
- Validación de referencias, estados y secuencias de eventos.
- Bloqueo heurístico de patrones frecuentes de secretos.

## Lo que una huella pública no demuestra

Un archivo JSON puede copiar una huella conocida. La huella por sí sola sirve para resolver una identidad, pero **no demuestra posesión de la llave privada**. La confianza fuerte exige verificar la firma del commit que introdujo ese archivo.

Por ello, el despliegue de producción debe:

1. exigir commits firmados en la rama protegida;
2. ejecutar `clusterlog verify commit` o una verificación equivalente en CI;
3. limitar quién puede modificar `data/admins.json` y `config/allowed_signers`;
4. requerir revisión de owner para altas, bajas y rotaciones de llaves;
5. conservar logs del proveedor Git.

## Llaves

- Sólo se guardan llaves **públicas**.
- Nunca copie una llave privada al repositorio ni a la entrada JSON de un agente.
- `--ssh-key` puede apuntar a la `.pub` para identidad; para `--commit`, la CLI intenta localizar la contraparte privada cuando está disponible.
- En estaciones administradas, prefiera `ssh-agent` o un dispositivo de hardware según la política local.
- Rote una llave comprometida de inmediato y revoque su acceso en el proveedor Git.

El MVP no incluye todavía un comando de rotación/desactivación. Hágalo mediante un cambio revisado por owners en `data/admins.json`, regenere `config/allowed_signers` con la CLI o una migración controlada y documente el incidente.

## Secretos

El escáner integrado es una defensa adicional, no una garantía. Antes de guardar una memoria:

- elimine tokens, contraseñas, cookies, llaves privadas y cadenas de conexión;
- redacte valores en comandos y salidas;
- no copie volcados completos de configuración;
- use identificadores de un gestor de secretos, nunca el valor;
- ejecute el secret scanner corporativo en pre-commit y CI.

Una vez que un secreto entra a Git, borrarlo del archivo actual no basta: debe revocarse y, si la política lo exige, reescribirse el historial.

## Sitio interno

El proyecto incluye `robots.txt` y `noindex`, pero estos controles **no son autenticación**. Despliegue detrás de al menos uno de los siguientes:

- VPN o red de administración;
- proxy con SSO y MFA;
- hosting privado con control de acceso;
- segmentación y listas de acceso.

No publique el directorio `public/` en un bucket o servicio accesible anónimamente.

## Contenido prohibido o restringido

No almacene:

- credenciales o material de autenticación;
- datos personales no necesarios;
- claves privadas, kubeconfigs con tokens o archivos de credenciales cloud;
- dumps de bases de datos;
- inventarios que revelen información clasificada más allá del acceso autorizado;
- instrucciones destructivas sin contexto, validación y reversión.

Defina una clasificación interna y márquela en la documentación si el entorno lo requiere.

## Agentes de IA

Un agente debe operar con los permisos mínimos del usuario y cumplir estas reglas:

- nunca afirmar que ejecutó o validó algo que no observó;
- no inventar comandos, salidas, tickets, horas ni revisores;
- presentar un `--dry-run` antes de escrituras sensibles;
- usar `--commit` sólo con autorización y acceso legítimo a la firma del usuario;
- no intentar exportar la llave privada;
- redactar secretos antes de llamar la CLI;
- detenerse ante errores de identidad, validación o firma;
- devolver el ID y SHA del commit como evidencia.

## Protección recomendada de rama

- Pull request obligatorio.
- CI obligatoria: tests, vet, sync check, validate, Zola build y secret scanning.
- Firmas verificadas.
- Revisión de CODEOWNERS para `data/admins.json`, `config/allowed_signers`, `.github/` y `SECURITY.md`.
- Prohibir force-push y borrado de rama principal.
- Respaldo del repositorio y prueba periódica de restauración.

Ejemplo de `CODEOWNERS` a adaptar:

```text
/data/admins.json       @equipo/owners-bitacora
/config/allowed_signers @equipo/owners-bitacora
/.github/               @equipo/owners-bitacora
/SECURITY.md            @equipo/seguridad @equipo/owners-bitacora
```

## Revisión de commits en CI

`scripts/verify-signatures-ci.sh` recorre el rango del pull request y ejecuta la verificación de cada commit contra `config/allowed_signers`. El scaffold sin administradores activos omite la regla para permitir el bootstrap; desde el primer alta, un commit sin firma SSH autorizada hace fallar la comprobación.

La protección de rama debe marcar la CI como obligatoria. Para conservar las firmas originales, prefiera una estrategia de merge que no sustituya todos los commits por un commit nuevo sin una identidad autorizada, o configure un bot de merge con una llave explícitamente aprobada. Los cambios de gobierno siguen necesitando revisión de owners además de una firma válida.

## Respuesta a incidentes

Ante contenido sensible o firma cuestionable:

1. retire temporalmente el sitio o restrinja acceso;
2. preserve evidencia y determine el alcance;
3. revoque credenciales expuestas;
4. suspenda la llave o usuario afectado;
5. corrija la rama mediante el procedimiento aprobado;
6. documente el incidente en el sistema canónico, enlazando una memoria redactada cuando sea seguro;
7. actualice manuales y controles preventivos.

## Documentación ejecutable (Runme)

`RUNBOOK.md` y `runbooks/**/*.md` permiten abrir procedimientos como
notebooks ejecutables (ver `docs/RUNME.md` y ADR-0004). Esto no cambia el
modelo de confianza: se agregan las siguientes reglas.

- Ejecutar una celda de Runme equivale exactamente a ejecutar el comando
  local con los permisos del usuario que la corre. No hay una elevación ni
  un sandbox adicional.
- El usuario debe leer una celda antes de ejecutarla, igual que revisaría
  cualquier comando pegado desde un chat o un documento.
- Un agente de IA no puede activar una celda `risk-mutating`/`risk-destructive`
  sin autorización explícita del usuario para esa operación puntual — igual
  que con cualquier otro comando mutante.
- Una etiqueta (`ci-safe`, `role-*`, `risk-*`) es metadato de clasificación,
  **no** un control de acceso. `interactive` y `excludeFromRunAll` son
  defensas adicionales que Runme sí respeta técnicamente, pero tampoco son
  autorización: son mecanismos para reducir el riesgo de un "Run All"
  accidental, no una barrera de permisos.
- Runme no debe leer, exportar ni copiar ninguna llave privada. Ninguna
  celda de este repositorio lo hace ni debe hacerlo.
- No se guardan outputs de celdas con secretos. Los Session Outputs de
  Runme están desactivados por defecto en esta integración; si se habilitan
  localmente, no deben versionarse ni compartirse.
- Runme Cloud, Gists y cualquier servicio remoto de Runme quedan fuera de
  alcance. `runme open` (servidor web local) no debe exponerse en `0.0.0.0`.
- La ejecución de celdas desde el sitio Zola publicado está fuera de
  alcance: Zola no construye `runbooks/`, y no existe ningún mecanismo para
  correr Runme desde el navegador.
- Las operaciones productivas reales (RBAC, `sudo`, gestores de secretos,
  kubeconfig, proveedor cloud, controles corporativos) siguen siendo
  responsabilidad de esas herramientas. Runme nunca las reemplaza ni las
  reimplementa.
- Un agente debe detenerse ante errores de identidad, firma o validación
  también dentro de un runbook, igual que en cualquier otro flujo de
  `clusterlog`.

## Reporte de vulnerabilidades

No abra un issue público con detalles sensibles. Use el canal privado de seguridad de la organización y proporcione versión, impacto, pasos de reproducción y una prueba sin secretos reales.
