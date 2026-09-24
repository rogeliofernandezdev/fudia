# Contrato con Fudia

Los endpoints de integración exigen X-Fudia-Concierge-Key; el QR no sustituye esta credencial.

## Carta

GET /v1/integrations/concierge/{token}/menu

q busca por nombre, descripción o categoría. productId revalida una selección exacta.

## Rondas

POST /v1/integrations/concierge/{token}/orders

foods-backend deriva organización, local y mesa desde el QR y recalcula precios y disponibilidad. conversationId permanece estable durante la conversación. requestId identifica idempotentemente cada ronda; repetirlo recupera la misma operación y una ronda posterior usa otro requestId.

La primera ronda abre la comanda. Las posteriores se agregan al mismo consumo y crean tickets KDS independientes.

## Cuenta

POST /v1/integrations/concierge/{token}/bill

Devuelve el consumo acumulado real, moneda, productos, total, pagos registrados, saldo pendiente y estado de pago. Registra la solicitud de cuenta, pero no registra un pago.