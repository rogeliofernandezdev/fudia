# Alcance — Fudia Concierge

## Propósito

Permitir que un cliente sentado en una mesa sea atendido mediante WhatsApp durante todo su consumo, sin duplicar el dominio comercial de Fudia.

## Flujo funcional

- El QR identifica restaurante, local y mesa mediante foods-backend.
- El cliente consulta carta, platos, menús/combos, bebidas y otras categorías reales.
- Disponibilidad, stock, precio y configuración proceden exclusivamente de foods-backend.
- Cada ronda exige confirmación explícita y se revalida al persistir.
- La primera ronda abre la comanda; las siguientes se agregan al mismo consumo.
- Cada ronda conserva su propio estado de Cocina/KDS para no reprocesar productos anteriores.
- Al pedir la cuenta, Concierge consulta consumo acumulado, pagos y saldo pendiente desde foods-backend.
- El pago y la liberación final de la mesa continúan en Caja/POS.

## Fuera del alcance actual

Pagos ejecutados dentro de WhatsApp, delivery conversacional, promociones autónomas, audio/imagen y reservas. Concierge no implementa lógica fiscal ni sustituye Caja/POS.