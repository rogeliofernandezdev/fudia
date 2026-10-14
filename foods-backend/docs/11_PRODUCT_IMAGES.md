# Contrato de fotografías de producto

Sesiones y permisos existentes, sin migración a JWT. Menu administra reserva,
confirmación y vínculo; adapters PostgreSQL/GCS se componen en cmd/api. API GCP
recibe JSON, no bytes. Multipart legacy permanece solo en local/AWS sin provider
GCS; en GCS se rechaza antes de leer el body. No se migran ni borran URLs legacy.

Migración 000080: product_images y products.image_id/image_key/image_status.
UUIDs de gen_random_uuid(). Autorización ligada a empresa, usuario, local y
producto opcional; máximo 20 pendientes y 100 por hora. Mismo usuario/local
confirma/adjunta. Rechazar referencias ajenas, sin confirmar o de otro producto.
removeImage limpia la referencia. La asociación es transaccional con el producto.

OpenAPI publica authorize/complete/read/attach, imageId de entrada y campos
de lectura temporales. Política POST V4 con límite real de tamaño. Nunca guardar
URL firmada: frontend envía ID, backend rechaza firmas en imageUrl. Edición
comercial sin imageId/removeImage conserva referencia nativa.

fudia-image-processor es proyecto independiente; consume solo contrato SQL de
imágenes con credencial propia. Catálogo, disponibilidad/comanda, combos y
Concierge resuelven imágenes por lote, con caché de firmas acotada. Thumbnail
200 y medium 600 px, sin ampliar. No existe versión 1200 px por decisión posterior.

PROCESSING/READY/FAILED; original temporal de fallback. Guardar no espera READY;
conservar estado si evento termina primero. Error de firma posterior al commit
no debe comunicar falso fallo de escritura. Worker no cambia catálogo comercial,
stock, caja, permisos ni sesiones.

IAM, CORS, SQL, eventos y lifecycle:
[runbook](../../foods-infrastructure/docs/10_PRODUCT_IMAGES.md).
