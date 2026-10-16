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

## Fin de la atención en mesa

El token QR es fijo por mesa: cuando el cliente se va, el siguiente comensal usa el mismo QR. Por eso, antes de cada turno en modo mesa:

1. Si la conversación tiene un pedido (`last_order_id`), se consulta `GET .../orders/{orderId}`; `active=false` (pagado y entregado, o cancelado) termina la atención.
2. Si no tiene pedido y pasó `TABLE_IDLE_SECONDS` (3 h) desde el último mensaje, también termina.
3. Si la sesión expiró, la memoria del cliente (último local por QR, 90 días) abre directamente el modo sin mesa.

Al terminar se abre un contexto nuevo (`conversationId` nuevo, carrito, confirmación, handoff e historial vacíos, `mode=delivery`). La respuesta de ese turno empieza con un aviso: la atención en mesa terminó y puede pedir para recojo o delivery. Un nuevo QR vuelve siempre al modo mesa.

## Pedido sin mesa: recojo o delivery

1. El cliente arma el carrito con la carta real, igual que en mesa.
2. Elige recojo o delivery (`set_fulfillment`). Si el local no tiene delivery, solo se ofrece recojo.
3. Para delivery comparte su ubicación de WhatsApp. El pin se procesa sin el modelo: `POST .../delivery-quote` devuelve distancia en línea recta, costo y cobertura; el bot responde con la distancia y el costo, o con el alcance máximo si está fuera de cobertura.
4. El bot pide dirección escrita, referencia y nombre (`set_delivery_details`).
5. `prepare_confirmation` exige lo que falte (`delivery_details_required` + `missing`) y muestra productos, subtotal, envío, total y tipo de entrega.
6. Tras la confirmación explícita, `confirm_order` envía `POST .../delivery-orders` con `quotedFee` y un `requestId` persistido antes de llamar.
7. El pedido entra a Cocina como canal `delivery` o `recojo` y se paga al recibir o recoger.

Cambiar el tipo de entrega, la dirección, el nombre o la ubicación invalida una confirmación preparada. `check_order_status` consulta el estado real del último pedido en ambos modos.

## Invariantes

- conversationId identifica la conversación; requestId identifica una ronda idempotente.
- requestId se persiste antes de llamar al backend y se reutiliza en cualquier reintento de esa ronda.
- Un mismo mensaje del cliente nunca ejecuta dos veces el modelo después de haber cambiado el carrito o el pedido.
- Mensajes y respuestas de un QR anterior no se aplican ni se envían en el contexto nuevo.
- Una nueva ronda no depende de que las anteriores sigan en estado confirmado.
- La cuenta y el saldo siempre se leen desde foods-backend.
- El modelo nunca inventa IDs, precios ni disponibilidad.
- La distancia y el costo de envío los calcula foods-backend; el modelo nunca los estima.
- Una conversación cuya atención en mesa terminó nunca vuelve a agregar rondas a esa mesa sin escanear el QR de nuevo.