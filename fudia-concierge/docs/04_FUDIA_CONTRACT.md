# Contrato con Fudia

## Existente

GET /v1/public/tables/{token}

## Integración server-to-server

Los endpoints de Concierge exigen `X-Fudia-Concierge-Key`; el QR no sustituye esta credencial.


GET /v1/integrations/concierge/{token}/menu

Parámetros: q y productId.

POST /v1/integrations/concierge/{token}/orders

El backend deriva organización, local y mesa desde el QR y recalcula precios. El body no acepta organizationId, locationId, tableId ni unitPrice como fuente de verdad. `conversationId` actúa como clave idempotente dentro del local: un reintento recupera el pedido ya creado en lugar de duplicarlo.

## Evolución

Antes de ampliar canales se incorporarán credenciales rotables por instalación/empresa, rate limiting por QR/teléfono y observabilidad de abuso.
