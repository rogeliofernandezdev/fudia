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
