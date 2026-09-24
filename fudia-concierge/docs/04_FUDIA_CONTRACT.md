# Contrato con Fudia

## Existente

GET /v1/public/tables/{token}

## Extensión inicial

GET /v1/public/concierge/{token}/menu

Parámetros: q y productId.

POST /v1/public/concierge/{token}/orders

El backend deriva organización, local y mesa desde el QR y recalcula precios. El body no acepta organizationId, locationId, tableId ni unitPrice como fuente de verdad.

## Evolución

Antes de ampliar canales públicos se incorporarán sesión de capacidad de corta duración y rate limiting por QR/teléfono.
