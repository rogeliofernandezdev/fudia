# Fotografías: subida directa y procesador independiente

Implementado en código, no desplegado ni verificado contra un bucket real. Se
conservan datos PostgreSQL, sesiones cookie/token opaco y workflows AWS. No hay
migración a JWT ni traslado de fotos antiguas.

## Flujo

1. Preview blob inmediato; JPEG/PNG/WebP, máximo 5242880 bytes (5 MiB), cada
   dimensión entre 1 y 4000 px. Validar antes de pedir autorización.
2. POST `/v1/admin/product-images/uploads`: solo JSON. Sesión y menu.manage
   vigentes, empresa/local y producto opcional verificados. UUID de PostgreSQL.
3. Política POST V4: bucket/key/MIME exactos, content-length-range real de 1 a
   5242880 bytes y vencimiento en 10 minutos. API no recibe el archivo.
4. Navegador sube FormData directamente a storage.googleapis.com, sin cookies de
   FudIA; file es el último campo. CORS del bucket permite el origen exacto.
5. POST `/product-images/{imageId}/complete` verifica metadatos en Storage y
   fija generación. Idempotente. POST/PATCH de producto recibe imageId y asocia
   dentro de su transacción, sin esperar la conversión. image/attach cubre los
   productos ya creados desde compras.
6. Eventarc finalized invoca **fudia-image-processor**, proyecto Go independiente,
   privado en Cloud Run. Valida contenido real, decodifica, corrige orientación,
   convierte a sRGB, elimina EXIF/metadata, comprime WebP (calidad 82).

```text
restaurants/{organizationId}/products/{imageId}/
├── original.jpg     (o .png/.webp)
├── thumbnail.webp   (lado mayor hasta 200 px)
└── medium.webp      (lado mayor hasta 600 px)
```

Decisión posterior del usuario: **sin versión 1200 px**. Proporción conservada,
sin crop ni upscale. restaurantId representa la empresa del catálogo compartido
entre locales; la operación incluye autorización de local activo.

products.image_id/image_key/image_status guardan referencias, nunca URL temporal.
product_images guarda reserva, propietario, metadatos y procesamiento. Reservar
no asocia todavía una foto al producto; no se persiste estado UPLOADING.

PROCESSING usa preview local si existe; de lo contrario original con generación
fijada. READY exige ambas variantes. FAILED conserva original como fallback y
permite seleccionar otra foto. El procesador es autoridad sobre el contenido:
metadata del cliente no prueba que el archivo sea realmente una imagen.

Lecturas de catálogo hacen una consulta SQL por lote y devuelven thumbnail;
detalle usa medium. GET firmados duran una hora; caché por instancia de hasta
4096 firmas con singleflight, concurrencia acotada y renovación un minuto antes.
No hay una petición de autorización del navegador por cada foto. Variantes:
Cache-Control privado de 30 minutos; original no-store. Detalle consulta cada
2 s mientras PROCESSING y renueva al enfocar o cada 30 minutos. Blob preview
acotado a 8 fotos, por empresa/local y reutilizable durante 5 minutos.

## Seguridad, eventos y carreras

- API firmante: Creator + Viewer, sin Delete/Update ni roles superiores
  heredados. Una política no puede sobrescribir el objeto original existente.
- Worker descarga generación exacta; variantes con DoesNotExist y metadato
  source-generation reconocen reintentos propios. Ignora eventos de derivados.
- Lease SQL 180 s; timeout de proceso 120 s; fencing por intentos. Duplicados
  ocupados/transitorios devuelven 503 para retry Eventarc; errores permanentes
  o cinco intentos fallidos terminan FAILED.
- Estado actualizado atómicamente. Guardar antes/después del evento no degrada
  READY; eventos antiguos solo afectan productos que aún referencien esa foto.
- Fallar al firmar después de commit no devuelve falso error de escritura.

## Qué crear/configurar en GCP

Además de [Cloud Run](09_GCP_CLOUD_RUN.md):

1. APIs Eventarc y Pub/Sub; Storage, IAM Credentials y Cloud Run.
2. Bucket regional dedicado, misma región/proyecto del trigger, uniform access
   y public access prevention. Sin backups/documentos dentro del bucket de fotos.
3. API SA: roles/storage.objectCreator + roles/storage.objectViewer solo en
   bucket. **Retirar objectUser de FUSE si se aplicó** y permisos Delete/Update
   heredados. iam.serviceAccounts.signBlob sobre sí misma (custom role mínimo;
   alternativa roles/iam.serviceAccountTokenCreator solo sobre esa SA).
4. Processor SA independiente: roles/storage.objectUser solo en bucket y Secret
   Accessor solo en su DATABASE_URL; roles/cloudsql.client para conexión por socket.
   Custom role mínimo opcional: objects.get,
   objects.create, objects.update. Sin acceso a secretos/sesiones del API.
5. Eventarc SA: roles/eventarc.eventReceiver en proyecto y roles/run.invoker
   únicamente en processor. Sin Storage, SQL ni Secret Manager.
6. Storage service agent: roles/pubsub.publisher para emitir eventos. Conservar
   roles administrados Eventarc/PubSub; proyectos antiguos pueden requerir Token
   Creator al agente Pub/Sub para entrega autenticada.
7. Usuario PostgreSQL image_processor y Secret Manager dedicado. Sin DDL ni
   roles heredados de acceso general. Grants después de migración 000080:

   Excepción aprobada para el MVP gcp-dev: se permite compartir usuario SQL y
   secreto con API/migrador. IAM sigue segregado; no afirmar mínimo privilegio SQL.
   Aplicar la separación siguiente antes de producción, no sobre el usuario
   compartido actual sin preparar la propiedad/permisos.

```sql
GRANT USAGE ON SCHEMA public TO image_processor;
GRANT SELECT ON product_images TO image_processor;
GRANT UPDATE (generation, lease_until, attempts, status, failure_code, processed_at)
  ON product_images TO image_processor;
GRANT SELECT (id, organization_id, image_id) ON products TO image_processor;
GRANT UPDATE (image_key, image_status, updated_at) ON products TO image_processor;
```

Configurar CONNECT y contraseña fuera de Git. El worker comparte solo este
contrato SQL versionado; las migraciones siguen siendo de foods-backend.
Deploy SA necesita Eventarc Admin y Service Account User sobre nuevas SAs,
además de sus permisos previos. No Owner/Editor ni claves JSON.

Variables nuevas en GitHub Environment:

| Variable | Ejemplo |
| --- | --- |
| GCP_IMAGE_PROCESSOR_SERVICE | fudia-image-processor-dev |
| GCP_IMAGE_PROCESSOR_SERVICE_ACCOUNT | image-processor@PROYECTO.iam.gserviceaccount.com |
| GCP_IMAGE_PROCESSOR_DATABASE_SECRET | foods-dev-image-processor-database-url:1 |
| GCP_EVENTARC_SERVICE_ACCOUNT | image-events@PROYECTO.iam.gserviceaccount.com |
| GCP_IMAGE_TRIGGER | product-images-dev |

API runtime: PRODUCT_IMAGE_STORAGE=gcs, PRODUCT_IMAGES_BUCKET,
PRODUCT_IMAGES_SIGNER (su propio email). Worker: bucket y su DATABASE_URL;
no credencial de usuario ni firmante del API.

## CORS, lifecycle y orden de despliegue

Adaptar gcp/product-images-cors.example.json a orígenes exactos (puerto local
solo en desarrollo), nunca wildcard. Desde Cloud Shell autorizado:

```bash
gcloud storage buckets update gs://TU_BUCKET --cors-file=cors.json
gcloud storage buckets update gs://TU_BUCKET --lifecycle-file=product-images-lifecycle.json
```

Lifecycle usa daysSinceCustomTime=2. Worker marca Custom-Time solo en original
y después de READY durable; no poner Age=2 a todo el prefijo. No se borran
variantes ni originales FAILED por esta regla. Lifecycle es asíncrono: no garantiza
el minuto exacto de borrado a las 48 h. Soft delete/versionado retiene bytes más
tiempo según política. FAILED, reservas abandonadas y variantes huérfanas requieren
mantenimiento separado; no se eliminan automáticamente en este cambio.

1. Restore controlado de datos y backup, recursos/permisos/secretos, CORS/lifecycle.
2. Workflow **foods-image-processor-gcp**: verifica ambos proyectos; construye
   API/migrador y worker antes de OIDC; aplica migraciones con el Job del backend,
   despliega worker privado, concede run.invoker a Eventarc solo sobre ese servicio
   y configura el trigger. API/worker comparten concurrency
   de esquema para no migrar juntos. Worker image no incluye API ni migraciones.
   El coordinador `foods-gcp` lo ejecuta primero por push a `feat/ui-standard`
   en `gcp-dev`; backend y frontend esperan el éxito de sus dependencias.
3. Completar grants después de migración 000080 antes de subir fotos. /health no
   certifica esos permisos. Confirmar trigger activo y ajustar ack deadline de
   su suscripción Pub/Sub a 600 s. Revisar reintentos y retención.
4. Deploy backend GCP y frontend. API exige una revisión ready de processor;
   comprobar además el trigger antes del corte. Se retira montaje FUSE.
5. Probar en GCP: preview, subida, guardado inmediato, reabrir PROCESSING, READY,
   WebP 200/600; archivo falso, >5 MiB directo a política firmada, >4000px,
   usuario sin permiso y otra empresa/local; evento duplicado, reemplazo y FAILED.
6. Confirmar limpieza de original solo tras READY; monitorear FAILED, PROCESSING
   >10 minutos, IAM/CORS/Eventarc, cuota signBlob, memoria, SQL y consumo Storage.

Un pipeline exitoso no certifica el flujo real. Eventarc mal configurado puede
dejar PROCESSING; configurar alertas y recuperación. AWS queda sin cambios.
gcp/verify-images.ps1 ejecuta ambos módulos con PostgreSQL Docker temporal;
tests gcloud son simulados. Conversión real de libvips se verifica en contenedor.

Fuentes: [POST policy](https://docs.cloud.google.com/storage/docs/xml-api/post-object),
[Eventarc](https://docs.cloud.google.com/run/docs/triggering/storage-triggers),
[lifecycle](https://docs.cloud.google.com/storage/docs/lifecycle),
[libvips](https://www.libvips.org/API/8.17/using-vipsthumbnail.html).
