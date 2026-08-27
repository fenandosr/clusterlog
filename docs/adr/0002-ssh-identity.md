# ADR-0002: Identidad por llave SSH y prueba mediante commit firmado

- **Estado:** aceptada
- **Fecha:** 2026-08-12

## Contexto

Los administradores ya usan llaves SSH y los agentes operan en su sesión local. Se busca evitar un sistema adicional de usuarios y contraseñas, pero una llave pública almacenada no basta para probar que una operación la realizó quien posee la privada.

## Decisión

- El registro `data/admins.json` asocia una o más huellas públicas con cada administrador.
- La CLI resuelve la identidad a partir de una llave local.
- El contenido conserva `admin_id` y huella para trazabilidad.
- Las escrituras de producción usan `--commit` para crear un commit Git firmado por SSH.
- `config/allowed_signers` se genera desde administradores activos.
- CI y protección de rama verifican firmas y autorización.

## Consecuencias positivas

- Reutiliza la práctica operativa existente.
- Evita secretos de aplicación dentro del repositorio.
- La evidencia se integra con el historial Git.
- Los agentes pueden usar el mismo contrato sin conocer la llave privada.

## Consecuencias negativas

- La configuración inicial de Git/OpenSSH requiere cuidado.
- Rotación y revocación deben gobernarse explícitamente.
- Una operación sin `--commit` queda atribuida sólo de manera declarativa.
- El proveedor Git debe aplicar políticas de firma y rama.

## Regla de interpretación

La huella es identidad declarada; una firma válida y autorizada es evidencia criptográfica. Nunca deben tratarse como equivalentes.
