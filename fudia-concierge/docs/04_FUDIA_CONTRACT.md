# Contrato con Fudia

Los endpoints de integración exigen X-Fudia-Concierge-Key; el QR no sustituye esta credencial.

## Carta

GET /v1/integrations/concierge/{token}/menu

q busca por nombre, descripción o categoría. productId revalida una selección exacta.
Cada producto trae `recommended`: los que el local marcó «Recomendar en la carta digital» llegan primero y
el especialista de carta los presenta como «Recomendados de la casa»; nunca se recomienda otro producto.

## Rondas

POST /v1/integrations/concierge/{token}/orders

foods-backend deriva organización, local y mesa desde el QR y recalcula precios y disponibilidad. conversationId permanece estable durante la conversación. requestId identifica idempotentemente cada ronda; repetirlo recupera la misma operación y una ronda posterior usa otro requestId.

La primera ronda abre la comanda. Las posteriores se agregan al mismo consumo y crean tickets KDS independientes.

## Estado del pedido

GET /v1/integrations/concierge/{token}/orders/{orderId}?conversationId=...

Devuelve código, canal, estado, totales, estado de pago y `active`. Solo responde para pedidos creados por esa conversación (404 en otro caso). `active=false` significa que la atención terminó: cuenta de mesa pagada y entregada, o delivery/recojo entregado o cancelado.

## Delivery y recojo

GET /v1/integrations/concierge/{token}/delivery-options

Recojo siempre disponible; `deliveryAvailable` exige ubicación del local y al menos un rango. Incluye dirección y teléfono del local, alcance máximo y rangos `{maxDistanceKm, fee}`.

POST /v1/integrations/concierge/{token}/delivery-quote

Cuerpo `{latitude, longitude}` (la ubicación nunca viaja en la URL). Devuelve `covered`, `distanceKm` (línea recta, 0.01 km), `fee`, `maxDistanceKm`. `409 delivery_unavailable` si el local no tiene delivery.

POST /v1/integrations/concierge/{token}/delivery-orders

Cuerpo `{fulfillment: delivery|pickup, customerName, customerPhone, conversationId, requestId, items, address?, reference?, latitude?, longitude?, quotedFee?}`. foods-backend recalcula precios, distancia y tarifa: `409 delivery_out_of_range` o `409 delivery_fee_changed` si la tarifa aceptada ya no es la vigente. `requestId` es idempotente: repetirlo devuelve el mismo pedido aunque las tarifas hayan cambiado. Crea un pedido `delivery` o `recojo` sin mesa, confirmado y enviado a cocina.

## Cuenta

POST /v1/integrations/concierge/{token}/bill

Devuelve el consumo acumulado real, moneda, productos, total, pagos registrados, saldo pendiente y estado de pago. Registra la solicitud de cuenta, pero no registra un pago. requestId identifica idempotentemente cada solicitud: repetirlo devuelve la cuenta vigente sin registrar otra solicitud; si se omite, cada llamada se registra.