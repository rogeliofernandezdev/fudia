# Fudia Pay

Dispositivos del local (Android, proyecto hermano `../fudia-pay`) que detectan
las notificaciones de pagos recibidos por **Yape** y **Plin** y las reportan al
backend para que Caja las revise, las asocie a un pedido y confirme el cobro.

**Regla de oro:** una notificación detectada nunca marca una venta como pagada
por sí sola y nunca se presenta como confirmación bancaria. El cobro real se
registra con el flujo de pagos existente de `billing`.

## Flujos

### 1. Primera vez — vinculación (una por teléfono)

1. Configuración › Locales › Editar local › pestaña «Dispositivos de pago»
   › «Vincular dispositivo» (el teléfono queda ligado a ese local): código temporal de 10 minutos, un solo uso, mostrado como QR.
2. La app lo escanea y lo canjea, como WhatsApp Web (migración 000131): al
   leer el QR avisa el escaneo (`POST /v1/mobile/pair/scan`, con el modelo) y
   Caja, que consulta el estado del código cada 1,5 s, cambia el QR por
   «Vinculando «modelo»…»; luego lo canjea (`POST /v1/mobile/pair`): el backend
   resuelve empresa y local, liga el dispositivo al código y devuelve la
   credencial, que viaja **una sola vez** y se guarda cifrada. Al quedar
   vinculado, el modal de Caja se cierra solo y avisa «Dispositivo vinculado».
   El aviso de escaneo es de mejor esfuerzo: si la red falla, el canje igual
   valida el código.
3. El cajero activa el **acceso a notificaciones** desde la configuración de
   Android (permiso del sistema, no de la app).
4. Un QR vencido o ya canjeado responde 400/409 (también al escanear): Caja
   muestra «El código venció sin usarse» con «Generar otro código».

### 2. Operación diaria (automática)

1. Android entrega la notificación de Yape/Plin al `NotificationListenerService`.
2. Se verifica el paquete y el parser extrae remitente y monto; sin confianza
   **no se registra nada**.
3. El evento entra a la cola local (Room) con su clave de idempotencia.
4. El worker lo envía; el backend deduplica (`UNIQUE device_id, event_key`) y
   lo registra como `detected`.
5. Aviso flotante en cualquier pantalla del cajero + entrada en la campana +
   fila en Caja › Pagos detectados.

### 3. Revisión y cierre (web)

- Con un modal de cobro abierto: el detalle **sugiere** ese pedido (nunca
  automático); «Asociar pago» o «Buscar otro pedido».
- Sin modal: lista de pedidos con saldo pendiente, monto exacto primero;
  la fila completa abre la revisión del saldo y «Asociar pago».
- La asociación **registra el cobro real** (turno y movimientos) y actualiza
  saldo/estado; parciales y excedentes con sus reglas.
- Cerrar sin asociar deja el movimiento `detected` (lista y campana), sin
  afectar pedidos. Un movimiento se asocia **una sola vez** (409).

### 4. Resistencia

Envío: el servicio de escucha envía cada pago apenas lo encola, sin abrir la
app (WorkManager en segundo plano lo aplazaría hasta abrirla). Sin internet:
la cola persiste y WorkManager reintenta con backoff + barrido periódico. Reinicio del teléfono: la cola sobrevive. Notificación repetida:
misma clave → `duplicate`. Dispositivo revocado: 401 inmediato. Sin permiso o
cuenta pausada: no llegan notificaciones y la app lo muestra con sinceridad.
Congelador de HONOR (2026-11-20): congela el proceso minutos después de dejar
la app y los pagos esperan hasta abrirla; la app mantiene un servicio en
primer plano con el aviso fijo «Detectando pagos» para impedirlo.
Listener enlazado que Android deja de alimentar (HONOR/MagicOS, 2026-11-20):
el barrido lo detecta y fuerza la reconstrucción del enlace desactivando y
reactivando su propio componente; al reconectar procesa los pagos que siguen
en la bandeja. Detalle en el README de `fudia-pay`.

### 5. Revinculación

Revocar en Caja (o desvincular en el teléfono) invalida la credencial; un QR
nuevo crea una nueva. La credencial anterior no vuelve a servir.

## Módulo y datos (`internal/modules/paydetect`)

- `pay_devices`: dispositivo por empresa/local con `platform` (`ANDROID|IOS`),
  credencial propia guardada **solo como hash SHA-256** (`token_hash`) y
  `token_suffix` para reconocerla; `active=false` al revocar.
- `pay_pairing_codes`: códigos temporales de un solo uso (10 min) que Caja
  muestra como QR; nunca contienen credenciales.
- `pay_notifications`: evento con `UNIQUE (device_id, event_key)` — la
  idempotencia soporta reintentos y solicitudes concurrentes. Estados:
  `detected → associated (→ dismissed)`; `order_id` y `payment_id` referencian
  el pedido y el cobro real creado por la asociación.

## Contratos

Desde 2026-11-18 (migración `000135_pay_operation_number`), los eventos admiten
`operationNumber` opcional: el número bancario que el proveedor incluye
explícitamente en la notificación, como texto para conservar ceros iniciales
(máximo 64 caracteres alfanuméricos o guiones). Se persiste y se devuelve en
`PayNotification`. No se deriva de `eventKey`, del ID interno, del monto ni de
un código de seguridad. Los eventos anteriores o de proveedores que omiten el
número conservan el campo vacío; Caja muestra «No informado».
Android lee también `android.bigText`, extrae referencias con etiqueta explícita
de operación y conserva la clave idempotente construida con el resumen original.
La cola Room migra de versión 1 a 2 sin borrar pagos pendientes.

La asociación exige `method == provider` (`yape` o `plin`), validado en backend;
el catálogo operativo sigue validando que ese medio esté activo para ventas.
El selector web está preseleccionado y deshabilitado. Si el medio falta, la
asociación queda bloqueada con indicación de activarlo en Configuración.
La referencia del cobro usa el proveedor y el número de operación cuando está
disponible; nunca publica la clave técnica del evento como número bancario.
Desplegar la migración y API antes de actualizar el frontend y el APK. No se
puede reconstruir el número de notificaciones históricas que no lo guardaron.

- `billing/contracts.PayRegisterer`: **asociar registra el cobro real** con el
  `Create` de pagos (turno, movimientos de caja, comprobante y cierre). Nada de
  segunda lógica de caja; un importe superior al saldo lo rechaza el propio
  flujo de cobros con sus reglas.
- `orders/contracts.PayOrderLookup`: valida que el pedido exista en el alcance.
- La lista de Caja (`GET …/pay-notifications`) acepta `q`: remitente que
  contiene el texto (sin tildes ni mayúsculas, con `translate()`, sin extensión
  `unaccent`) o monto que empieza con la cifra escrita
  (`domain.NotificationAmountPrefix`). No se busca por número de operación:
  Yape no lo envía, y el número solo se muestra cuando la notificación lo trae.
- «Asociar pago» lista candidatos con `GET /v1/admin/pos/orders?paymentStatus=unpaid&ready=true`:
  `ready` aplica en SQL la misma regla que `billing/domain.PaymentReady`
  (`posPaymentReadySQL`, comparada en `payment_ready_sql_test.go`), así la lista
  no ofrece pedidos que el cobro rechazaría por cuenta abierta o pedido no listo.
- `notifications/contracts.Publisher`: deja el aviso en la campana
  (`kind=pay_detected`, audiencia cajero) por cada pago registrado. El detalle
  dice «S/ 1.00 · remitente»: el monto usa el símbolo de la moneda de
  `organizations/contracts.CurrencyByCode`, nunca el código ISO («PEN»).

## Endpoints

Cajeros (permisos `cash.read`/`cash.manage`, sesión del panel):

| Ruta | Uso |
| --- | --- |
| `POST /v1/admin/locations/{locationId}/pay-devices/pairing` | Código temporal para el QR (10 min, un solo uso); devuelve `id`, `code` y `expiresAt` |
| `GET /v1/admin/locations/{locationId}/pay-devices/pairing/{pairingId}` | Estado del código: `pending`, `scanned` (con `model`), `linked` (con `deviceName`) o `expired`; solo del local |
| `GET /v1/admin/locations/{locationId}/pay-devices?status=&page=&size=` | Dispositivos del local: `active`, `revoked` o todos; `{items,total}`, activos primero |
| `POST /v1/admin/locations/{locationId}/pay-devices/{deviceId}/revoke` | Revocar la credencial (401 inmediato) |

El local de la ruta se valida contra la empresa de la sesión (`ForLocation`);
uno ajeno responde 404 `pay_location_not_found`. Los dispositivos piden
`organizations.read` para listar y `organizations.manage` para vincular y
revocar.
| `GET /v1/admin/pay-notifications?status=&provider=` | Pagos detectados |
| `POST /v1/admin/pay-notifications/{id}/associate` | Asocia y registra el cobro (`{orderId, method, amount?}`); una sola vez (409) |
| `POST /v1/admin/pay-notifications/{id}/dismiss` | Descarta con traza |

Dispositivo (credencial `Bearer`, revocable):

| Ruta | Uso |
| --- | --- |
| `POST /v1/mobile/pair/scan` | Sin credencial: avisa que la app leyó el QR (`{code, model}`), 204; idempotente |
| `POST /v1/mobile/pair` | Canjea el código del QR; el token viaja **una sola vez** |
| `GET /v1/mobile/session` | Restaurante, local y moneda resueltos por el backend |
| `POST /v1/mobile/notifications` | Reporta lotes (≤20); responde `registered|duplicate` por evento |

La empresa y el local **siempre** los resuelve el backend desde la vinculación;
el dispositivo nunca los envía. Los reintentos con WorkManager y la cola local
de la app hacen el resto; el sweep horario purga códigos vencidos.

## Experiencia en `foods-admin-web`

Corrección 2026-11-18 (migración `000136_pay_staff_notices`): el tipo
`pay_detected` estaba declarado pero faltaba tanto en la normalización como
en la restricción SQL de avisos. Se admite en ambas y se recuperan, sin
duplicados, los avisos de pagos todavía detectados de las últimas 24 horas.
La campana los muestra a caja y abre `/cash?tab=pagos`. La tarjeta flotante
es independiente: X o 8 segundos la ocultan sin descartar ni asociar el pago;
el tiempo se pausa con hover, foco o pestaña oculta. Solo emergen pagos nuevos
después de cargar el historial, y el contexto se reinicia al cambiar de usuario,
empresa o local. El teléfono aparece plegado en «Origen del aviso».

La app no captura automáticamente la pantalla de la constancia dentro de
Yape/Plin: el acceso a notificaciones no proporciona esa imagen. El soporte
para adjuntar o compartir una constancia real sigue pendiente de definición.

- Aviso flotante global en cualquier pantalla del cajero (sondeo cada 10 s,
  tarjetas apilables por encima de los modales, clic → detalle).
- Modal de detalle: medio, remitente, monto, fecha, número de operación y
  estado; «Asociar pago a pedido» sugiere el pedido que el cajero está cobrando
  (contexto del POS), permite cambiarlo, revisar su detalle (saldo, pagos) y
  confirmar. Un pago superior al saldo se registra parcial del saldo y la
  diferencia queda explicada.
- Caja › Pagos detectados (pestaña de Caja y turnos, sin casilla de menú
  propia: la habilita el acceso a Caja): filtros y paginación en el servidor
  (`?page=&size=`, respuesta `{items,total}`). Consulta y revisión posterior;
  los movimientos sin asociar permanecen pendientes sin afectar pedidos.
- Configuración › Locales › Editar local › «Dispositivos de pago» (desde
  2026-11-19; antes vivía en Mi restaurante y usaba el local de la sesión sin
  decirlo): vincular y revocar los teléfonos de ese local, con filtro
  Activos/Revocados/Todos y paginación en el servidor. No hay renombrar: el
  nombre es el modelo que declara la app y volver a vincular crea un registro
  nuevo; los revocados quedan como historial. (La casilla de menú `pagos_detectados` de la
  página separada que existió el 2026-11-16 se retiró; la migración 000133 la
  limpia de los roles.)

## Preparación para iOS

Contratos independientes de la plataforma (`ANDROID|IOS`). iOS no ofrece un
equivalente general a `NotificationListenerService`: la futura versión iOS
requerirá otro mecanismo autorizado; no se desarrolla en esta etapa.
El panel ya distingue la plataforma (ícono Android o iPhone en la lista del
local) para que un futuro dispositivo `IOS` no requiera cambios de interfaz.

**Permisos restringidos (HONOR/Huawei/Xiaomi/Oppo):** al instalar el APK por
fuera de la tienda, MagicOS/HyperOS/ColorOS bloquea permisos delicados («Fudia
Pay · Controlado por ajustes restringidos»). En el teléfono: Ajustes › Apps ›
Gestionar apps › Fudia Pay › menú ⋮ › **«Permitir ajustes restringidos»** →
confirmar. Después el acceso a notificaciones se activa con normalidad; es un
paso único por teléfono, documentado también en el README de la app.

## Pruebas

- `internal/modules/paydetect/domain`: normalización de eventos, claves,
  montos y vigencia de códigos.
- `internal/transport/httpapi` `TestPayDetectPairingReportAndCashReview`: canje,
  reporte idempotente, aislamiento entre restaurantes, asociación con cobro
  real (1 pago en `payments`), duplicado 409 y revocación 401.
