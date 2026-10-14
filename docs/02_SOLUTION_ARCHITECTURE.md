# Arquitectura de solución

## Decisión

La solución tiene cinco proyectos coordinados por contrato. Se
conserva la base técnica de `bodegas`: monolito modular, arquitectura hexagonal,
vertical slices, PostgreSQL, REST/OpenAPI y mensajería solo donde aporte valor.

```text
foods-admin-web ─ HTTPS/OpenAPI ─ foods-backend ─ PostgreSQL
                                      │              ^
WhatsApp ─ fudia-concierge ─────────────┤              │ contrato de imágenes
                                      ├─ SUNAT/OSE   │
                                      ├─ impresión   │
                                      └─ pagos       │
                                                     │
foods-admin-web ─ upload directo ─ Cloud Storage      │
                                      │              │
                                      └─ Eventarc ─ fudia-image-processor
                                                       │
                                                       └─ WebP en Storage

foods-infrastructure: despliegue, secretos, red y observabilidad
```

## Un frontend operativo y administrativo

`foods-admin-web` concentra POS, mesas, comandas, caja, KDS y administración.
Rutas, layouts y permisos separan los modos de trabajo sin duplicar sesión,
contratos ni sistema visual. Se diseña mobile-first desde 390 px; no existe
un segundo frontend operativo como proyecto independiente.

## Backend

- Go.
- Monolito modular con puertos y adaptadores.
- PostgreSQL como fuente de verdad.
- REST con OpenAPI versionado.
- Outbox para eventos que deban publicarse con garantía.
- WebSocket o SSE para comandas y cocina en tiempo real.
- Redis únicamente para coordinación efímera demostrable; nunca como verdad.

## Frontend

- Next.js, TypeScript estricto y Tailwind CSS.
- TanStack Query para estado remoto.
- React Hook Form y Zod para formularios.
- BFF y sesión con cookies `HttpOnly` cuando corresponda.
- Componentes propios sobre tokens compartidos, sin copiar identidad de terceros.

## Tenencia y seguridad

Toda entidad operativa pertenece a `organization_id` y, cuando corresponda, a
`location_id`. El backend deriva ambos alcances de la sesión. Auditoría obligatoria
para anulaciones, descuentos, reaperturas, ajustes, cierres y comprobantes.

La empresa es la frontera de tenencia. Sus perfiles fiscales se separan por país;
cada local referencia uno de esos perfiles. Moneda, impuesto y tipo de cambio no
se duplican como preferencias del usuario ni como columnas independientes del
local. Los tipos de cambio son históricos, efectivos por fecha y auditables.


## Fudia Concierge

- Python/FastAPI + LangGraph.
- OpenAI se usa para interpretar conversación y seleccionar herramientas; no es
  fuente de verdad de catálogo, disponibilidad, precios ni pedidos.
- Redis conserva únicamente sesión y carrito conversacional efímero.
- La integración con `foods-backend` es server-to-server mediante contrato
  versionado y credencial propia.
- El QR es una capacidad de contexto: permite identificar empresa, local y mesa,
  pero no reemplaza la autenticación del servicio para crear pedidos.
- `foods-backend` recalcula precio, valida disponibilidad/stock y persiste el
  pedido antes de que entre a KDS.

## FudIA Image Processor

Proyecto independiente de procesamiento, no un módulo de negocio extraído del
monolito ni una función del API. Su servicio Cloud Run privado consume eventos
Storage finalized mediante Eventarc y publica WebP de hasta 200 y 600 px.
No genera 1200 px, no amplía fotos pequeñas y no hace esperar el guardado.

El API conserva autorización de usuario/empresa/local, reservas, confirmación,
vínculos a productos y firmas de lectura. El navegador sube directamente a
Storage mediante política POST V4 con límite real de 5 MiB. Processor valida
contenido y resolución hasta 4000 × 4000, transforma y registra el resultado.
La autenticación existente no cambia a JWT por introducir este servicio.

Reglas transversales obligatorias:

- Responsabilidad exclusiva de imágenes; no endpoints comerciales ni acceso
  a stock, precios, sesiones, usuarios o pagos.
- Identidad IAM y usuario SQL propios, contenedor no-root y bucket privado.
- PostgreSQL genera UUID; almacenar object keys, nunca URLs temporales.
- Persistencia compartida explícitamente limitada a `product_images` y las
  columnas de referencia/estado del producto, con grants por columna.
- `READY` solo después de ambas variantes; `FAILED` ante contenido inválido o
  agotamiento de intentos. `UPLOADING` pertenece únicamente al frontend.
- Generación inmutable, lease y fencing para duplicados, caídas y eventos
  tardíos; una foto reemplazada nunca vuelve a ser la referencia vigente.
- Original exitoso: Custom-Time después de READY y lifecycle de dos días.
  Limpieza de fallidos/huérfanos y recuperación operativa siguen pendientes.
- Migraciones del backend antes del worker; workflows GCP separados de AWS.
  Código y pruebas locales no certifican IAM/Eventarc sin recursos reales.

Las responsabilidades por capa, contrato de eventos, estados, límites y reglas
de reintentos están definidos en
[la arquitectura del procesador](../fudia-image-processor/docs/01_ARCHITECTURE.md).
La configuración concreta está en
[el runbook de imágenes](../foods-infrastructure/docs/10_PRODUCT_IMAGES.md).
