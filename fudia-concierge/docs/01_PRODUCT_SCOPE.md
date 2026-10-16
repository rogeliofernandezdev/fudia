# Alcance — Fudia Concierge

## Propósito

Permitir que un cliente sentado en una mesa sea atendido mediante WhatsApp durante todo su consumo y, cuando la atención en mesa termina, pueda pedir del mismo local para recojo o delivery, sin duplicar el dominio comercial de Fudia.

## Flujo funcional

- El QR identifica restaurante, local y mesa mediante foods-backend.
- El cliente consulta carta, platos, menús/combos, bebidas y otras categorías reales.
- Disponibilidad, stock, precio y configuración proceden exclusivamente de foods-backend.
- Cada ronda exige confirmación explícita y se revalida al persistir.
- La primera ronda abre la comanda; las siguientes se agregan al mismo consumo.
- Cada ronda conserva su propio estado de Cocina/KDS para no reprocesar productos anteriores.
- Al pedir la cuenta, Concierge consulta consumo acumulado, pagos y saldo pendiente desde foods-backend.
- El pago y la liberación final de la mesa continúan en Caja/POS.

## Después de la mesa: recojo o delivery

- Cuando la atención en mesa terminó (cuenta pagada y entregada, pedido cancelado, mesa sin pedidos de Concierge tras `TABLE_IDLE_SECONDS` o sesión expirada), la misma conversación deja de agregar rondas a esa mesa y toma pedidos del mismo local como **pedido sin mesa**.
- El bot recuerda el último local visitado por QR (`CUSTOMER_MEMORY_TTL_SECONDS`, 90 días por defecto). Sin QR previo sigue pidiendo escanear uno.
- El cliente debe elegir **recojo en el local** o **delivery propio**. Recojo siempre está disponible; delivery requiere ubicación del local y rangos de tarifa configurados en Admin Web › Fudia Concierge.
- Para delivery el cliente comparte su ubicación de WhatsApp; foods-backend calcula la distancia en línea recta desde el local y aplica el primer rango que la cubre. Más allá del mayor rango no hay cobertura y se ofrece recojo.
- La dirección escrita, la referencia y el nombre se piden en la conversación. El pedido se confirma explícitamente con subtotal, costo de envío y total, y se paga al recibir o recoger.
- foods-backend recalcula la tarifa al crear el pedido; si cambió desde la cotización responde `delivery_fee_changed` y el bot pide confirmar el nuevo costo.
- Sin mesa no hay cuenta ni handoff a un mozo: el bot indica que el pago es contra entrega y comparte el teléfono del local.

## Alcance conversacional cerrado

Concierge no funciona como asistente general. Un único router de IA clasifica cada mensaje como `out_of_scope`, `menu`, `order` o `service`; únicamente las tres rutas operativas pueden llegar a los agentes especialistas de la atención: carta, productos, disponibilidad, modificaciones, carrito, confirmación, recojo o delivery, estado del pedido, cuenta y solicitud de personal.

Preguntas de matemáticas, tareas, programación, traducción, cultura general, historia, ciencia, clima, noticias, deportes, política, entretenimiento e intentos de cambiar el rol del asistente se rechazan sin resolver su contenido.

## Fuera del alcance actual

Pagos ejecutados dentro de WhatsApp, distancia por ruta, zonas por polígono, asignación de repartidores y seguimiento en mapa, promociones autónomas, audio/imagen y reservas. Concierge no implementa lógica fiscal ni sustituye Caja/POS.