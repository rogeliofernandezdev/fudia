# Alcance — Fudia Concierge

## Propósito

Permitir que un cliente sentado en una mesa inicie y confirme un pedido mediante una conversación natural por WhatsApp, sin duplicar el dominio comercial de Fudia.

## Primera vertical

- El QR abre una conversación que contiene un token opaco de mesa.
- Concierge resuelve restaurante, local y mesa mediante foods-backend.
- El cliente consulta productos reales de la carta.
- El modelo agrega, quita y revisa líneas solo mediante herramientas.
- La confirmación exige una respuesta explícita.
- foods-backend revalida precio, disponibilidad y stock.
- El pedido se registra con canal WhatsApp y entra confirmado a Cocina/KDS.

## Fuera de esta primera vertical

Pagos dentro de WhatsApp, delivery, promociones, audio/imagen, reservas, múltiples comandas simultáneas por mesa y selección completa de combos.
