# Arquitectura y reglas de FudIA Image Processor

## 1. Decisión y estado

Proyecto Go independiente, con módulo, Dockerfile y workflow propios. Es un
servicio privado de Cloud Run invocado por Eventarc, no un comando dentro del
API ni una función de `foods-backend`.

Responsabilidad única: **validar, transformar y publicar fotografías de productos
y registrar el resultado**. La conversión no debe consumir CPU, memoria ni ancho
de banda del API comercial ni bloquear el guardado del producto.

Implementado en repositorio; no desplegado ni certificado en GCP. Las pruebas
locales no certifican IAM, Eventarc, CORS o un bucket real. Crear los recursos
es una condición previa a esa validación, no una prueba pendiente ejecutable hoy.

## 2. Fronteras y responsabilidades

| Componente | Debe hacer | No debe hacer |
| --- | --- | --- |
| Frontend | Preview, validación inicial, upload directo, guardar y mostrar estado | Decidir permisos definitivos o esperar la conversión para guardar |
| `foods-backend` | Sesión, empresa/local, permisos, reserva, autorización, confirmación, vínculo y firmas de lectura | Recibir bytes o ejecutar libvips en modo GCS |
| Storage/Eventarc | Persistir objetos y entregar finalized con identidad de servicio | Garantizar entrega exactamente una vez o reemplazar la autorización comercial |
| Processor | Validar contenido, generar WebP, publicar, actualizar estado y habilitar limpieza exitosa | Crear productos, autorizar uploads, firmar URLs de lectura o modificar precios/stock |

**No migrar usuarios a JWT por introducir este servicio.** El API conserva la
autenticación existente cookie/token opaco. IAM autentica la invocación Eventarc;
los headers CloudEvent por sí solos no autentican a nadie.

## 3. Flujo y experiencia inmediata

```text
Frontend ─ JSON: autorización ─> API ─ política POST V4 ─> Frontend
                                                           │
                                                      archivo directo
                                                           v
Frontend ─ confirmar y guardar ─> API/SQL              Cloud Storage
   │                                                       │ finalized
   └─ detalle inmediato                                    v
                                                       Eventarc
                                                           │ IAM
                                                           v
                                                fudia-image-processor
                                                  │               │
                                                  v               v
                                            WebP en Storage    estado SQL
```

1. API crea reserva antes de autorizar. PostgreSQL genera el UUID mediante
   `gen_random_uuid()`; el procesador nunca genera identificadores persistidos.
2. Frontend/API validan JPEG/PNG/WebP, 1–5242880 bytes (5 MiB) y 1–4000 px por
   lado. La política POST V4 fija key/MIME, vence en 10 minutos y limita el
   tamaño realmente en Storage. El archivo no atraviesa el API.
3. Frontend confirma metadatos/generación y guarda con `imageId`, sin esperar
   `READY`. En PROCESSING usa preview local, o original con URL temporal.
4. **El evento puede llegar antes o después de confirmar/guardar.** Processor
   puede transformar una reserva sin producto. Confirmar/vincular no debe
   degradar READY a PROCESSING. En READY se usa la variante correspondiente.
5. FAILED informa fallo y permite otra imagen. El original solo sirve de
   fallback si es visualizable; un archivo inválido no es una imagen certificada.

`UPLOADING` solo existe en frontend. La reserva interna inicia PROCESSING,
pero no aparece como foto del producto hasta estar confirmada y vinculada.
Una reserva no prueba que el objeto exista ni que su contenido sea válido.

## 4. Capas y reglas de dependencias

```text
cmd/processor                    composición, configuración y servidor
internal/transport/events        recepción/validación HTTP del evento
internal/application             orquestación, estados y puertos
internal/domain                  registro, límites y claves de imagen
internal/infrastructure/postgres lease y finalización transaccional
internal/infrastructure/gcs      descarga, publicación y Custom-Time
internal/infrastructure/imaging  validación real y adaptación a libvips
```

- Dominio/aplicación no importan GCP, pgx ni ejecutan libvips. Usar los puertos
  `ImageProcessingRepository`, `ImageProcessingStorage`, `ImageTransformer`.
- `cmd/processor` conecta adaptadores. No importar paquetes internos del API.
- No agregar frontend, login, endpoints comerciales, pagos, facturación o KDS.
- No crear migraciones aquí: el esquema sigue siendo propiedad del backend.
- Procesar dentro de la petición antes del ACK; no lanzar trabajo después de
  responder ni depender de archivos locales entre eventos o instancias.

## 5. Contrato de eventos

Entrada `POST /`, CloudEvent binario con `Ce-Type` igual a
`google.cloud.storage.object.v1.finalized` y `Ce-Id` obligatorio. Cuerpo JSON
con `bucket`, `name` y `generation` como cadena de entero positivo, máximo
64 KiB. Bucket debe coincidir con `PRODUCT_IMAGES_BUCKET`.

| Respuesta | Regla |
| --- | --- |
| 204 | Terminado, fallo permanente registrado o evento ignorado de forma segura |
| 503 | Lease ocupado o fallo transitorio: necesita redelivery |
| 400 | Evento/bucket/generación inválidos, no se procesa |
| 405 | Método no permitido |

Ignorar claves fuera del patrón, reservas inexistentes, otras generaciones y
derivados. **Las variantes también emiten finalized: nunca reprocesarlas.**
`Ce-Id` identifica logs, pero la consistencia no depende solo de ese ID: se
protege por empresa, imagen, generación y lease.

## 6. Claves, validación y transformación

```text
restaurants/{organizationId}/products/{imageId}/
├── original.jpg       (o original.png / original.webp)
├── thumbnail.webp     lado mayor hasta 200 px
└── medium.webp        lado mayor hasta 600 px
```

`organizationId` representa la empresa del catálogo compartido entre locales.
Empresa, local y propietario provienen de la reserva autorizada; no confiar en
datos libres del evento/cliente. UUIDs y estructura de key deben ser válidos.
No usar nombres originales del usuario. Un reemplazo usa otro `imageId`.

Reglas obligatorias:

1. Descargar generación exacta; contrastar tamaño, MIME y creación dentro del
   plazo autorizado con la reserva. Lectura acotada a 5 MiB y longitud verificada.
2. Validar formato real, no solo extensión/MIME; dimensiones por lado hasta
   4000 antes de decodificar y decodificación completa del primer frame.
   Admitir dimensiones JPEG intercambiadas por orientación del navegador.
3. Solo WebP 200/600, proporción conservada, sin crop ni upscale. **No 1200 px.**
4. Libvips con autoorientación, exportación sRGB, calidad 82 y eliminación de
   EXIF/metadatos. No prometer conservar animaciones: el contrato es de fotos.
5. Ejecución sin shell y argumentos constantes. Directorio temporal exclusivo,
   eliminado al finalizar incluso con error; no ejecutar datos del usuario.
6. Comprobar resultados WebP válidos y dimensiones dentro de su variante.
7. Publicar `image/webp`, `private,max-age=1800,immutable` y metadata
   `source-generation`. Crear con `DoesNotExist`, nunca sobrescribir a ciegas.
8. Un objeto existente se reconoce como reintento propio únicamente al coincidir
   generación fuente, MIME y tamaño. Un conflicto no es éxito.
9. READY exige que **ambas variantes** hayan sido publicadas correctamente.

## 7. Contrato SQL, propiedad y tenencia

Misma base PostgreSQL, no una segunda base. Migración `000080_product_images`
del backend define el contrato compartido. Esta excepción SQL es explícita y
limitada: no permite acceder a tablas de negocio ajenas.

| Datos | Propietario / permisos del procesador |
| --- | --- |
| Reservas, metadatos declarados, confirmación y vínculo | API; processor no los crea ni reasigna |
| `product_images` | Lee; solo actualiza generation, lease_until, attempts, status, failure_code, processed_at |
| `products` | Lee id/organization_id/image_id; solo actualiza image_key/image_status/updated_at si sigue referenciando la imagen |
| Esquema y migraciones | Backend; processor no tiene DDL |
| Object keys | Persistentes en SQL; no guardar URLs firmadas |
| URLs de lectura | API: firmas y catálogo por lote; processor no firma |

- Usuario SQL propio con grants por columna, sin permisos generales/heredados.
- Filtrar toda adquisición/finalización por `organization_id` e `imageId`.
- La key original debe coincidir con la reserva. No basta encontrar un UUID.
- READY usa `medium.webp` como image_key; PROCESSING/FAILED usan original.
- No insertar productos ni cambiar image_id, nombre, precio, stock, identidad,
  sesiones, usuarios, permisos, pagos o cuentas.
- Separación de responsabilidad/despliegue no significa independencia total de
  persistencia. Cambiar el contrato requiere migraciones compatibles en ambos
  proyectos; una futura separación de datos exige contrato privado y reconciliación,
  no callbacks públicos ni secretos improvisados.

## 8. Estados, lease, reintentos y carreras

| Estado/transición | Condición |
| --- | --- |
| PROCESSING | Reservada/en proceso; fallo transitorio libera lease y conserva estado |
| PROCESSING → READY | Dos variantes publicadas y finalización SQL atómica |
| PROCESSING → FAILED | Contenido inválido, o fallo de procesamiento con intento adquirido >= 5 |
| READY repetido | Sin conversión; solo reintentar programación de limpieza si corresponde |
| FAILED repetido | Sin reprocesamiento automático; usuario selecciona otra foto |

Invariantes:

1. Claim solo en PROCESSING/generación compatible sin lease vigente: fija
   generación si falta, lease 180 s y aumenta attempts. Lease ocupado → 503.
2. Caída/timeout permite recuperar lease vencido. `attempts` es fencing token:
   Finish exige generación, intento y estado adquiridos; un proceso viejo no
   finaliza sobre un intento nuevo.
3. Estado de imagen y referencia del producto se finalizan en una transacción.
   Bloquear producto antes de imagen, igual que el guardado del API.
4. Evento de foto reemplazada puede finalizar su registro, nunca cambiar el
   producto que referencia otra imagen. Guardar antes/después no degrada READY.
5. Una sola variante no habilita READY. Reintentos reconocen los objetos propios
   ya publicados y completan faltantes. ACK solo después de persistir resultado.
6. `invalid_image` es permanente. Otros errores de conversión/Storage se
   reintentan conservando PROCESSING; al umbral se registra `processing_failed`.
7. Fallos de entrega o SQL que impiden adquirir/finalizar intento no están
   acotados por ese contador. Exigen supervisión y política de entrega externa.
   No existe una DLQ ni una recuperación manual automatizada en este proyecto.

## 9. Retención y limpieza

Después de READY durable, marcar Custom-Time del original de esa generación.
Lifecycle configurado elimina objetos marcados después de dos días. No aplicar
Age=2 indiscriminadamente al prefijo; no marcar variantes ni originales FAILED.

- Si marcar falla, conservar READY y devolver error para redelivery; el siguiente
  evento reintenta limpieza sin convertir de nuevo. Original ausente es idempotente.
- Lifecycle, no temporizadores en memoria, realiza el borrado. No se garantiza
  el minuto exacto de 48 h; soft delete/versionado puede retener bytes más tiempo.
- Originales FAILED, reservas abandonadas y variantes huérfanas **no se limpian
  automáticamente hoy**. Requieren política de mantenimiento separada; no declarar
  que todos los originales caducan a las 48 h.

## 10. Seguridad y límites operativos

- Cloud Run exige IAM, sin allUsers/allAuthenticatedUsers. Ingress actual es
  `all`: privado por autenticación de invocación, no por aislamiento de red.
- Identidades separadas: API, processor, Eventarc y migrador. Processor usa ADC,
  acceso acotado al bucket y solo su secreto DATABASE_URL; sin claves JSON.
- Eventarc invoca solo processor; no acceso a Storage, SQL ni sus secretos.
- Bucket privado dedicado. CORS y firmas de subida/lectura pertenecen al flujo
  API/frontend y a infraestructura, no al procesador.
- SQL por red privada con usuario mínimo; sin migraciones al arrancar, DDL ni
  credenciales de administrador. Contenedor no-root.
- No registrar archivos, EXIF, sesiones, credenciales, políticas ni URLs firmadas.

| Parámetro actual | Límite |
| --- | --- |
| Runtime | DATABASE_URL y PRODUCT_IMAGES_BUCKET obligatorios; PORT por defecto 8080 |
| Cloud Run | 1 CPU, 1 GiB, concurrency 1, mínimo 0 / máximo 3 instancias |
| Go / libvips | GOMEMLIMIT=512MiB; VIPS_CONCURRENCY=1; hasta 30 s por comando |
| Evento / HTTP / plataforma | 120 s / WriteTimeout 130 s / Cloud Run 150 s |
| Lease | 180 s, mayor que el procesamiento |
| SQL | Pool máximo 2 por instancia: hasta 6 conexiones, adicionales a API/migrador |
| Arranque | Ping SQL y startup probe GET /health |

GOMEMLIMIT no limita por sí solo memoria nativa de libvips. Pool 2 está fijado
en código; DATABASE_MAX_CONNS del despliegue no es ajuste dinámico del worker.
`/health` no certifica IAM, permisos de columnas, Storage/Eventarc ni readiness
continua. No hay endpoint de métricas propio.

## 11. Despliegue, observabilidad y puertas de calidad

Orden: recursos/secretos → migración backend → grants → processor/trigger →
comprobar entrega → API/frontend. Workflow `foods-image-processor-gcp.yml` y
script `foods-infrastructure/gcp/image-processor-deploy.mjs`; AWS no se reemplaza.
Runtime del processor no contiene API/migraciones. Pipelines API/worker comparten
grupo de concurrencia de esquema para serializar migraciones.

Logs JSON actuales: arranque y `image_processing_retry` con eventId/error.
Alertas para FAILED, PROCESSING >10 minutos, reentregas, IAM, memoria, SQL y
lifecycle se deben configurar antes de producción; no están creadas por el
código. No confundir logs con un sistema completo de métricas o recuperación.

- Go: `go test ./...`, `go vet ./...`, `go build ./cmd/processor`.
- Validación y estados: contenido falso, dimensiones/tamaño, ambas variantes,
  fallos, duplicados, leases vencidos, fencing y eventos de imagen reemplazada.
- Conversión real desde este proyecto:
  `docker build --target verification -t fudia-image-conversion-test .` y
  `docker run --rm fudia-image-conversion-test -test.v`.
- Integración SQL: solo FUDIA_IMAGE_PROCESSOR_TEST_DATABASE_URL de base temporal,
  nunca fixtures contra producción. Runner compartido: gcp/verify-images.ps1
  en foods-infrastructure.
- GCP **cuando existan recursos**: IAM/CORS/tamaño real, entrega autenticada,
  guardado inmediato, carreras, reintentos y limpieza posterior a READY.
  Simulaciones gcloud, CI o un build exitoso no prueban ese flujo real.

## 12. Documentos relacionados

- [Arquitectura transversal](../../docs/02_SOLUTION_ARCHITECTURE.md).
- [Contrato funcional del API](../../foods-backend/docs/11_PRODUCT_IMAGES.md).
- [OpenAPI](../../foods-backend/api/openapi.yaml): frontera oficial del frontend.
- [Recursos, IAM, grants y variables GCP](../../foods-infrastructure/docs/10_PRODUCT_IMAGES.md).
- [Cloud Run y corte de base de datos](../../foods-infrastructure/docs/09_GCP_CLOUD_RUN.md).

Este documento define responsabilidades e invariantes. El runbook conserva los
comandos/permisos concretos. Mantenerlos coherentes con código y migraciones,
sin presentar las limitaciones pendientes como capacidades implementadas.

