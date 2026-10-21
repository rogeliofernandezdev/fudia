# Contrato con Fudia

Los endpoints de integración exigen X-Fudia-Concierge-Key; el QR no sustituye esta credencial.

## Carta

GET /v1/integrations/concierge/{token}/menu

q busca por palabras en nombre, descripción o categoría, sin distinguir mayúsculas ni tildes y con plurales simples («ceviches», «aji»); basta una palabra y los productos con más coincidencias van primero. Las palabras de relleno («¿qué tienes para hoy?», «quiero comer») se ignoran: una búsqueda hecha solo de ellas devuelve la carta completa. productId revalida una selección exacta.
Cada producto trae `recommended`: los que el local marcó «Recomendar en la carta digital» llegan primero y
el especialista de carta los presenta como «Recomendados de la casa»; nunca se recomienda otro producto.
Cada producto trae también `remaining`: porciones o unidades que quedan hoy en el local, o `null` si no controla cantidad.
Al agregar al carrito (producto suelto, con modificadores o combo y cada opción elegida), Concierge suma lo que ya está en el carrito y, si se pasa, responde `insufficient_stock` con `available` e `inCart` sin agregar nada; el agente dice cuántos quedan y pregunta si agrega esa cantidad.
Es una validación temprana: al confirmar, foods-backend vuelve a validar con bloqueo y responde `insufficient_stock` o `insufficient_portions` si otra mesa pidió antes.

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

## Prerregistro del cliente

Al crear un pedido de mesa (`/orders`) o de recojo/delivery, foods-backend usa el
`customerPhone` del cuerpo para prerregistrar al comensal en `customers` de la
empresa del QR, dentro de la misma transacción del pedido: reutiliza el cliente si
el número ya existe (también guardado sin código de país) y si no lo crea con canal
preferido WhatsApp. El pedido nuevo queda enlazado (`orders.customer_id`); una ronda
que se suma a una comanda abierta registra al cliente sin cambiar el cliente de esa
comanda. Concierge no hace llamadas adicionales para esto.
