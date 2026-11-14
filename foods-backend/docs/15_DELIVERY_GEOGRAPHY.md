# Centrales de delivery y cobertura de Fudia

## Propiedad y selección

Solo Plataforma Fudia registra, edita, ordena, activa o desactiva centrales y
sus zonas. Las empresas no reciben endpoints de escritura de ese catálogo.
La activación de Delivery Fudia por empresa sigue siendo exclusiva de Plataforma.

Cada central se registra una sola vez y conserva `coverageCodes` con los distritos
que Fudia marca mediante casillas. Departamento y provincia sirven únicamente
para filtrar la lista: no se guardan como cobertura implícita. La misma central
puede atender distritos de distintas provincias y departamentos. El orden global
se aplica solo entre centrales activas que tienen marcado exactamente el
`districtCode` del local de origen; no se usa el destino del cliente para elegir
ese origen.

`locations.district_code` conserva el distrito del local. Alta de empresa y
alta/edición de locales admiten `districtCode`. Las interfaces peruanas piden
departamento, provincia y distrito, con selectores dependientes; el backend
publica `GET /v1/admin/geographic-areas?parentCode=` para cualquier sesión
autenticada. Ese endpoint es de solo lectura.

Sin distrito o cobertura coincidente, el pedido queda sin central y se avisa a
caja. Una lista vacía nunca significa cobertura nacional. Los locales y centrales
anteriores necesitan completar sus zonas; no se infieren de direcciones libres
ni de coordenadas. Los pedidos ya aceptados conservan su central cuando esta se
desactiva o cambia su cobertura; las ofertas pendientes se retiran y avanzan.

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
pendiente y coincidencia exacta con distritos marcados, zonas múltiples y falta de distrito.
`TestDeliveryGeographyCatalogAndPlatformOwnership` usa una sesión real de empresa:
permite leer el catálogo y rechaza administrar centrales, orden y activación de
Delivery Fudia. Verifica también distrito válido, desconocido y provincia
enviada como distrito, sin modificar el local ante el rechazo.

## Panel web de la central y seguimiento (migración 000125)

Desde 2026-11-14 la central de reparto atiende sus pedidos en un panel web
propio, sin WhatsApp ni sesión: `delivery_partners.panel_token` (32 hex,
generado por PostgreSQL, único) abre `/delivery-panel/{token}` en foods-admin-web.
Plataforma lo copia o lo reemplaza con
`POST /v1/platform/delivery-partners/{id}/panel-link`; el anterior deja de abrir.

- `GET /v1/public/delivery-panel/{token}`: ofertas vigentes de esa central,
  repartos en curso y entregados de las últimas 12 horas. Mostrar una oferta
  por primera vez inicia sus 3 minutos (`expires_at`), igual que enviarla por
  WhatsApp.
- `POST …/actions` (`accept`, `reject`, `pickup`, `deliver`): comparte con los
  botones de WhatsApp los mismos bloqueos y resultados (`Dispatcher.act`). La
  autorización compara la central del enlace con la de la oferta; la de otra
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
  centrales activas con cobertura. La primera que acepta se lo queda; la
  aceptación bloquea pedido → despacho → central, así que dos clics o dos
  centrales a la vez no pueden tomarlo dos veces.
- **Motorizado.** `POST /v1/public/delivery-panel/{token}/courier` exige el nombre
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
