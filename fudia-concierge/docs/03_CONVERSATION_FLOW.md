# Flujo conversacional

## Inicio

El QR inicia WhatsApp con un token opaco de mesa. El token se resuelve antes del LLM.

## Atención y rondas

1. Resolver la mesa.
2. Consultar la carta con search_menu por nombre, descripción o categoría; query vacío permite consultar la carta.
3. Agregar únicamente IDs devueltos por Fudia.
4. Revalidar producto, combo o modificadores.
5. Revisar el carrito temporal de la ronda.
6. prepare_confirmation fija una clave idempotente para esa ronda.
7. El cliente confirma explícitamente.
8. confirm_order envía la ronda a foods-backend.
9. El carrito se vacía, pero la conversación y la comanda siguen abiertas.
10. Cada ronda aparece de forma independiente en KDS.

## Cuenta

1. El cliente pide la cuenta o pregunta cuánto debe.
2. Si hay productos sin confirmar, primero se resuelven.
3. request_bill consulta la comanda real de la mesa.
4. foods-backend devuelve detalle acumulado, total, pagos y saldo pendiente.
5. Concierge muestra ese resultado sin reconstruirlo desde el chat.
6. El cobro continúa en Caja/POS.

## Invariantes

- conversationId identifica la conversación; requestId identifica una ronda idempotente.
- Una nueva ronda no depende de que las anteriores sigan en estado confirmado.
- La cuenta y el saldo siempre se leen desde foods-backend.
- El modelo nunca inventa IDs, precios ni disponibilidad.