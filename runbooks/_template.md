# <Título del runbook>

> Copie este archivo, renómbrelo en `kebab-case` y borre los comentarios
> `> así`. No borre las secciones: si alguna no aplica, escriba "No aplica" y
> explique por qué en una línea.

## Título

`<slug-del-runbook>`

## Propósito

> Qué logra este runbook en una o dos frases. Si no puede explicarlo en dos
> frases, probablemente son dos runbooks.

## Propietario

> Rol o persona responsable de mantenerlo actualizado.

## Audiencia

> Uno o más de: `role-developer`, `role-operator`, `role-reviewer`, `role-owner`.

## Entornos permitidos

> Uno o más de: `env-local`, `env-ci`, `env-staging`, `env-production`.

## Clasificación de riesgo

> `risk-read-only`, `risk-mutating` o `risk-destructive`, por celda — no
> necesariamente una sola clasificación para todo el documento.

## Prerrequisitos

> Herramientas, variables de entorno, accesos y estado previo necesarios.
> Distinga:
> - **Observación**: qué debe ser cierto antes de empezar.
> - **Hipótesis**: qué se espera que pase, si aplica (para runbooks de
>   diagnóstico).

## Variables requeridas

> Nombre, propósito y si es secreta (nunca ponga el valor aquí; use
> `.env.example` como referencia de nombres).

## Preflight

> Celda de sólo lectura que confirma que el entorno está listo, sin instalar
> ni mutar nada.

```sh {"name":"template-preflight-example","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
echo "reemplace esto por una comprobación real de prerrequisitos"
```

## Procedimiento

> Distinga explícitamente:
> - **Acción**: qué comando se ejecuta.
> - **Resultado esperado**: qué debería observarse si funcionó.

Ejemplo de celda de sólo lectura (segura para CI):

```sh {"name":"template-safe-example","tag":"role-developer,env-local,risk-read-only,ci-safe","interactive":"false"}
echo "una celda de sólo lectura va aquí"
```

Ejemplo conceptual de celda que muta estado (nunca copie literalmente sin
adaptar prerrequisitos y validación):

```sh {"name":"template-mutating-example","tag":"role-operator,env-production,risk-mutating,manual-only","interactive":"true","excludeFromRunAll":true}
echo "una celda que muta estado real va aquí; nunca risk-mutating + ci-safe"
```

## Validación

> Cómo se confirma que el resultado observado es el esperado. Comandos
> concretos, no "revisar que todo esté bien".

## Rollback

> Cómo revertir, si aplica. Si no hay reversión directa, dígalo explícitamente
> y describa el procedimiento de corrección manual.

## Registro en Clusterlog

> Qué memoria, tarea o commit debe generarse fuera de Runme como evidencia
> canónica de que esto se ejecutó.

## Evidencia

> Qué guardar como prueba (salida JSON, SHA de commit, ID de memoria).

## Referencias

> Enlaces a `CLI.md`, `SECURITY.md`, ADRs u otros runbooks relacionados.

## Limitaciones

> Qué este runbook explícitamente no cubre, y por qué.
