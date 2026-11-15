# Centrales de delivery y cobertura de Fudia

## Propiedad y selección

Decisión vigente del 2026-11-14: todas las centrales activas reciben a la vez
los pedidos de los locales peruanos de empresas con Delivery Fudia habilitado.
La primera que acepta se queda con el pedido; las demás ofertas se retiran.
El orden global solo organiza el listado, sin prioridad de recepción.

Solo Plataforma Fudia registra, edita, ordena, activa o desactiva centrales y
habilita Delivery Fudia por empresa. El formulario requiere nombre y teléfono;
no solicita distritos. Los datos históricos de `delivery_partner_coverage` y
los campos de respuesta `coverageCodes/coverageLabels` se conservan por
compatibilidad, pero no filtran ofertas, mensajes, botones ni paneles.
`coverageCodes` de entrada está obsoleto y se ignora, sin modificar el histórico.
Una central nueva sin cobertura registrada recibe ofertas normalmente.

El distrito del local continúa siendo parte de su dirección y del catálogo
INEI, independiente del despacho. Un local sin distrito también puede recibir
Delivery Fudia. La tarifa y la distancia siguen calculándose entre coordenadas
del local y ubicación del cliente según sus rangos de envío; no usan distritos.
Los pedidos aceptados conservan su central al desactivarla; sus ofertas
pendientes se retiran. Se conserva la restricción de país soportado (Perú).

## Fuente y migración

La migración `000118_delivery_geography` agrega el catálogo `geographic_areas`,
el distrito del local y `delivery_partner_coverage`. Todos los IDs persistidos
son UUID generados por PostgreSQL; los códigos INEI son claves naturales.

Instantánea consultada el 2026-11-07: 25 departamentos, 196 provincias y 1890
distritos publicados por el [GeoServer INEI](https://geoespacial.inei.gob.pe/geoserver/Interoperabilidad/ows),
capas `ig_departamento`, `ig_provincia` e `ig_distrito`. La migración conserva las
filas y el hash de la descarga. El servicio no consulta INEI durante la operación;
una actualización del catálogo se incorpora mediante otra migración revisada.

Desplegar la migración antes del API y frontend. Después completar los distritos
de los locales y la cobertura de las centrales desde sus formularios.

## Consistencia y botones

Las mutaciones bloquean pedido, despacho y central en ese orden. Aceptación,
cesión del costo de envío y su auditoría se confirman en una transacción.
Delivery llama los puertos transaccionales de Orders mediante composición; no
se abren transacciones anidadas. Volver a delivery propio restaura el envío en la
misma transacción. La recogida valida y actualiza pedido y despacho juntos.

Los botones aceptados exigen la vuelta y central vigentes. Un botón de otra
vuelta o de una asignación retirada devuelve `taken`, incluso si esa oferta
había sido aceptada. El doble toque de la asignación vigente sigue siendo
idempotente.

Si caja registró pagos antes de la aceptación, el pedido pasa a delivery propio,
conserva todos sus importes y no envía instrucciones de cobro a otra central.
Aceptar devuelve `payment_conflict`; Concierge indica no recoger ni cobrar y
coordinar con el local. Reintentar ese pedido responde 409
`delivery_payment_conflict`. Esto se valida bajo el mismo bloqueo que los cobros.
Después de la aceptación, caja sigue registrando el pago recibido de la central.
Si ya se cobró tras aceptar, cambiar a reparto propio exige devolver antes esos
pagos: no se altera el total de una cuenta cobrada ni su comprobante.

## Pruebas

`TestDeliveryFudiaPartnerCascade` recorre botones antiguos tras reasignación,
pagos completos/parciales anteriores a aceptación, desactivación con oferta
pendiente, ofertas a centrales con distritos históricos distintos o vacíos,
local sin distrito y aceptación concurrente con un único ganador.
`TestDeliveryGeographyCatalogAndPlatformOwnership` usa una sesión real de empresa:
permite leer el catálogo y rechaza administrar centrales, orden y activación de
Delivery Fudia. Verifica también distrito válido, desconocido y provincia
enviada como distrito, sin modificar el local ante el rechazo.

## Panel web de la central y seguimiento (migración 000125)

Desde 2026-11-14 la central de reparto atiende sus pedidos en un panel web
propio, sin WhatsApp: `/delivery-panel` en foods-admin-web. Desde 2026-11-15
(migración 000128) el panel exige iniciar sesión; el enlace privado de la central
(`panel_token`) dejó de existir (ver «Acceso de la central» abajo).

- `GET /v1/delivery-partner/panel`: ofertas vigentes de la central de la sesión,
  repartos en curso y entregados de las últimas 12 horas. Mostrar una oferta
  por primera vez inicia sus 3 minutos (`expires_at`), igual que enviarla por
  WhatsApp.
- `POST …/actions` (`accept`, `reject`, `pickup`, `deliver`): comparte con los
  botones de WhatsApp los mismos bloqueos y resultados (`Dispatcher.act`). La
  autorización compara la central de la sesión con la de la oferta; la de otra
  central responde `not_found`. `deliver` exige el despacho `picked_up` y el
  pedido `en_camino`; lo pasa a `entregado` y el despacho a `delivered`
  (`not_picked_up` si aún no se recoge). Es idempotente.
- Recoger exige el pedido listo y el pago completo registrado por caja al local,
  neto de devoluciones (`orders.paid_amount >= orders.total` y cobrado positivo).
  `paymentConfirmed` publica esa confirmación en cada pedido del panel. Sin ella,
  «Recogí el pedido» está deshabilitado y explica que debe esperar a caja. Un abono
  parcial o una devolución que deja saldo vuelven a bloquearlo. El backend devuelve
  `payment_pending` sin modificar pedido ni despacho, también desde WhatsApp y si
  el local ya lo puso en camino. La validación usa el bloqueo del pedido compartido
  con cobros y devoluciones. Repetir una recogida ya aplicada conserva su resultado.
- `POST …/location`: guarda la última ubicación del repartidor
  (`delivery_dispatches.courier_*`) en los repartos `accepted` y `picked_up` de
  la central. El punto viaja en el cuerpo, nunca en la URL.

`delivered` conserva el envío con la central (`WithPartner`): el barrido no lo
devuelve al pedido. La entrega confirmada por la central no tiene usuario del
restaurante: `DeliverServiceItems` admite un alcance sin `UserID`.

El seguimiento del cliente (`delivery/contracts.TrackingReader`) se agrega a
`ConciergeOrderStatus.delivery`: local de origen y punto de entrega con sus
coordenadas, estado del despacho, central y, solo mientras el reparto sigue en
curso y la ubicación tiene menos de 15 minutos, la posición del repartidor.
Concierge lo publica al chat web; el navegador nunca elige el pedido.

## Motorizado, liberación, alertas y entrega fallida (migración 000126)

Desde 2026-11-14. No hay estados nuevos: el despacho sigue en `searching`,
`accepted`, `picked_up`, `delivered`, `own`, `unassigned` o `cancelled`, y el
pedido en sus estados de siempre. Los hechos del reparto son columnas de
`delivery_dispatches`. El delivery propio no cambia.

- **Oferta simultánea.** Cuando cocina inicia la preparación (`Orders` avisa al
  despacho también al pasar a `preparando`), el pedido se ofrece a todas las
  centrales activas. La primera que acepta se lo queda; la
  aceptación bloquea pedido → despacho → central, así que dos clics o dos
  centrales a la vez no pueden tomarlo dos veces.
- **Motorizado.** `POST /v1/delivery-partner/panel/courier` exige el nombre
  (2 a 60 caracteres) antes de entregar el enlace `/delivery-courier/{courierToken}`
  (32 hex generados por PostgreSQL). El mismo nombre conserva el enlace; otro
  nombre lo reemplaza, el anterior responde 404 y la etapa no cambia. El enlace
  vale hasta la entrega, la liberación, la entrega fallida o un cambio de nombre.
  Desde él, el motorizado recoge (pago completo confirmado por caja), entrega,
  marca «No pude entregar» y comparte su ubicación
  (`/v1/public/delivery-courier/{token}`, `…/actions`, `…/location`).
- **Liberar.** `release` (solo antes de recoger) marca la oferta `withdrawn` con
  `released_at`, invalida el enlace, abre una vuelta nueva, devuelve el envío al
  pedido y ofrece el pedido a las demás centrales. La central que liberó no lo
  vuelve a recibir automáticamente; «Reintentar» de caja sí la incluye. Con pagos
  registrados responde `payment_conflict`; recogido, `already_picked_up`.
- **Alertas de recogida.** Con el pedido `listo` y el despacho `accepted`, la
  espera cuenta desde `ready_notified_at` o desde el último cambio de motorizado
  (`courier_assigned_at`): 3 min `warning`, 5 min `urgent` y 8 min `critical`. A
  los 8 min el barrido publica una sola vez por motorizado `delivery_pickup_delayed`
  a caja (`pickup_alerted_at`). La central decide: `keep` (guarda
  `pickup_alert_ack_level`), cambiar motorizado (nuevo enlace, la espera se
  reinicia) o `release`. Fudia nunca libera por tiempo.
- **No pude entregar.** El motorizado llama al cliente al número del pedido
  (fuera de Fudia), espera y reintenta. Si no logra entregar, `not_delivered`
  guarda `delivery_issue_reported_at` y la distancia al destino si su ubicación
  tiene menos de 15 minutos (`delivery_issue_distance_m`). La central lo ve en su
  panel y, si el pedido vino del chat, Orders deja el aviso al cliente
  `delivery_contact_request` en la misma transacción. «Entregado» sigue disponible.
- **Entrega fallida.** `fail` exige el reporte y 10 minutos desde él. Guarda en
  `delivery_failed_deliveries` cliente, dirección, motorizado y montos; cierra el
  pedido del local como `entregado` (ya cobró la comida al motorizado), cierra el
  despacho como `cancelled` (el envío queda con la central) e invalida el enlace.
  La central reembolsa al motorizado el 100 % de lo que pagó por la comida. La
  pérdida se reparte 50 % Fudia y 50 % central solo si la incidencia se valida:
  el motorizado reportó a 300 m o menos del destino con ubicación reciente y el
  cliente recibió el aviso por el chat. Límites: 3 incidencias cubiertas por
  central en 30 días y aporte de Fudia de hasta 100 por incidencia. Fuera de eso
  la central asume el total y `validation_notes` explica el motivo.
- **Cancelación.** Preparado y en manos de una central (`listo`/`en_camino` con
  despacho `accepted`, `picked_up` o `delivered`), Orders rechaza cancelar con
  `409 delivery_in_progress_not_cancellable`. Concierge no ofrece ni acepta
  cancelaciones por mensaje en esa etapa.

## Opción «Delivery» del restaurante

Desde 2026-11-14 el restaurante tiene una sola opción «Delivery» en su menú,
con todo su delivery. El backend publica tres lecturas con `orders.read` y el
alcance de empresa y local de la sesión; ninguna permite editar centrales:

- `GET /v1/admin/delivery/orders`: deliveries abiertos y de las últimas 24 h,
  con despacho, central, motorizado, alerta de recogida, reporte «No pude
  entregar» y entrega fallida. `dispatchStatus` vacío es delivery propio.
- `GET /v1/admin/delivery/partners`: centrales activas que cubren el distrito
  del local y si Delivery Fudia está activo para la empresa.
- `GET /v1/admin/delivery/failed-deliveries`: entregas fallidas del local de los
  últimos 90 días con sus montos.

La migración `000127_delivery_single_menu` quita `delivery_apps` y
`repartidores` del acceso a menú de los roles: ya no son opciones del menú, y sus
pestañas dependen solo del módulo activo. El catálogo de menús del rol muestra
solo «Delivery».

Plataforma sigue siendo la única que administra centrales
(`/v1/platform/delivery-partners`); en el frontend lo hace desde la misma
pestaña «Proveedores». Reintentar y usar delivery propio siguen en
`/v1/admin/orders/{id}/delivery-dispatch/*`.

`TestDeliveryFudiaCourierFlow` recorre nombre inválido, oferta ajena, doble clic
con el mismo nombre, cambio de motorizado con enlace anterior inválido,
liberación con oferta a la otra central, alertas de 3 y 8 minutos con un solo
aviso al restaurante sin liberar por tiempo, mantener, liberar con pago
registrado, recoger desde el enlace, cancelación bloqueada, ubicación, «No pude
entregar» con un solo aviso al cliente, espera mínima y montos de la entrega
fallida, y las tres lecturas del local, incluido que otro local no ve nada.

`TestDeliveryPartnerPanel` recorre enlace inválido o reemplazado, oferta ajena,
acciones fuera de orden, ubicación inválida, vencida o de otra central,
seguimiento de otro local y la conservación del envío tras entregar.

## Acceso de la central: operadores con sesión (migración 000128)

Desde 2026-11-15 el panel de la central no es público. Cada persona de la
central es un operador (`delivery_partner_operators`) con correo único (en
minúsculas) y contraseña bcrypt. El enlace del motorizado sigue siendo público:
vale para un solo reparto y vence solo.

- **Alta (Plataforma).** `POST /v1/platform/delivery-partners/{id}/operators`
  crea al operador con una contraseña temporal de 12 caracteres sin 0/O/1/l/I,
  generada con `crypto/rand`. Solo viaja en esa respuesta (`no-store`) y nunca
  se lista. `…/{operatorId}/password-reset` genera otra, quita el bloqueo y
  cierra sus sesiones. `PATCH …/{operatorId}` edita nombre, correo o estado;
  desactivar cierra sus sesiones.
- **Inicio de sesión.** `POST /v1/delivery-partner/auth/login` abre una sesión de
  12 horas en la cookie HttpOnly `fudia_partner_session` (SameSite=Lax, Secure con
  `FOODS_COOKIE_SECURE`), separada de `foods_session`. La tabla
  `delivery_partner_sessions` guarda solo el SHA-256 del token. Un correo
  desconocido compara contra un hash ficticio para no revelar si existe.
  Cinco intentos fallidos seguidos bloquean 15 minutos (`delivery_operator_locked`,
  403); tras vencer el bloqueo la cuenta empieza de nuevo. Un operador inactivo
  recibe `delivery_operator_inactive` solo con la contraseña correcta.
- **Contraseña temporal.** Con `must_change_password` el operador puede ver su
  sesión (`GET /v1/delivery-partner/session`) y cambiarla
  (`POST /v1/delivery-partner/password`: actual + nueva de 8 caracteres a 72
  bytes, distinta). El panel y sus acciones responden 403
  `password_change_required` hasta entonces. Cambiarla cierra sus otras sesiones.
- **Panel.** `GET /v1/delivery-partner/panel` y `POST …/panel/actions`,
  `…/panel/location` y `…/panel/courier` toman la central de la sesión
  (`OperatorReady`); sin sesión responden 401 y el BFF borra la cookie. Cada
  acción aplicada queda en `delivery_partner_operator_actions` (operador, acción,
  resultado y código del pedido); el registro no interrumpe la acción si falla.
- **Cerrar sesión.** `POST /v1/delivery-partner/auth/logout` revoca la sesión y
  borra la cookie.
