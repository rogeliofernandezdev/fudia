# Gateway WhatsApp: proceso continuo

Código y plantilla de despliegue preparados; no se han creado recursos ni desplegado
el gateway. Meta continúa en el release actual de Concierge. La selección de proveedor
se hace manualmente en el único registro por país, sin failover.

## Host y almacenamiento

Usar una VM Linux existente/autorizada con Docker Compose, o un servicio equivalente
con CPU y red continuas. No desplegar como HTTP con escalado a cero. Plantilla:
`whatsapp-gateway/compose.yaml`; una réplica, reinicio automático, filesystem de solo
lectura, usuario sin root y puerto local únicamente. El coste del host y SQL es continuo.

Construir `fudia-whatsapp-gateway/Dockerfile`, publicar imagen por digest y establecer
`GATEWAY_IMAGE` al digest aprobado. En el host, su identidad administrada recupera
secretos de versiones numéricas de Secret Manager y genera `/run/fudia-whatsapp-gateway/runtime.env`
sobre tmpfs, propietario root y modo 0600. No crear archivos permanentes de claves,
no usar `set -x`, no mostrar `docker inspect` ni valores en logs. Revisar quién tiene
acceso al daemon de Docker: equivale a acceso a las credenciales de runtime.

PostgreSQL dedicado con TLS verificado, cifrado administrado en reposo y backups,
restauración ensayada, rol exclusivo y red restringida. Montar la CA en el host/contenedor
si no está en el almacén de confianza del sistema y referenciarla en `sslrootcert`.
`SESSION_STORAGE_PROTECTED=true` confirma esas condiciones; no activa cifrado por sí solo.
No usar un proxy SQL con pooling por transacción: el propietario necesita conexión fija.

La plantilla no publica acceso remoto al puerto 8020. Configurar delante un proxy/LB
HTTPS autorizado con certificado confiable y acceso desde Concierge; impedir acceso
público al backend del gateway. Mantener autenticación entre servicios incluso en VPC.
HTTP remoto, QR y métricas públicos no son un despliegue válido.

## Integración con Concierge y backend

En Concierge, tanto HTTP como worker requieren referencias privadas:

- `WHATSAPP_GATEWAY_KEY`: mismo valor que `GATEWAY_API_KEY`.
- `WHATSAPP_GATEWAY_INGRESS_KEY`: mismo valor que `CONCIERGE_INGRESS_KEY`.
- `WHATSAPP_GATEWAY_URL`: origen HTTPS autorizado, sin credenciales en URL.
- `WHATSAPP_COUNTRY_CODE`: país del activo Meta heredado (PE por defecto).
- `WHATSAPP_META_ACCOUNTS`: opcional, JSON secreto de activos Meta autorizados
  adicionales/retenidos para conversaciones históricas. No son canales operativos.

El release GCP existente admite estas tres referencias opcionales en
`GCP_CONCIERGE_SECRETS` y transmite la URL/país desde sus variables. Conserva los
secretos Meta requeridos por el despliegue actual y su preflight. No elimina los
activos anteriores mientras existan operaciones pendientes. El gateway se despliega
por separado: no se ha añadido un segundo número ni una conmutación automática.

El backend requiere `FUDIA_CONCIERGE_URL` HTTPS y su clave de integración existente
para verificar números y consultar QR privados. Publicar primero las migraciones
120–122 y backend/Concierge compatibles; después configurar el gateway y vincularlo
desde el formulario. El frontend también conserva su URL de Concierge para los QR.
Las tres aplicaciones deben escribir en el mismo entorno autorizado de Fudia.

## Cambio y verificación

La migración conserva duplicados archivados y auditoría. Los registros activos
heredados quedan pendientes hasta verificar el activo original: Concierge solo
restaura automáticamente un único activo importado si el número real coincide.
Varios activos previos para un país requieren revisión manual; no se elige uno.
Si Meta no está disponible durante el arranque, corregir configuración y reiniciar
Concierge, o verificar/activar desde el formulario. No se activa otro proveedor solo.

Los QR impresos permanecen iguales; cada scan consulta el canal actual del país.
Los avisos y respuestas conservan la referencia histórica de origen. No retirar una
sesión antigua o sus secretos hasta cerrar su bandeja pendiente. Ante desconexión
los mensajes quedan pendientes/error explícito; nunca salen por otro número.

Comprobar `/ready`, sesiones conectadas, pendientes y fallos de relay/reconexión.
Alertar por pérdida de conexión sostenida, crecimiento del outbox, envíos inciertos
y fallo de backups. Acceder a `/metrics` con credencial interna y sin logs de payload.
No lanzar dos versiones a la vez: detener la anterior, comprobar desconexión y luego
iniciar la nueva; el advisory lock es una segunda barrera, no un mecanismo de failover.

Ensayo de staging pendiente con número autorizado: vincular QR, reiniciar, recibir
texto/ubicación, respuesta Gemini con espera de 2,5–5 segundos, imágenes, recibos,
mesa finalizada → selector, cambio de número manteniendo QR y aviso pendiente por
canal original. Las pruebas locales simuladas no certifican WhatsApp ni IAM/red reales.
