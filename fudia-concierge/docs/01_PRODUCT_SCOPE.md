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

Saludos, agradecimientos y comentarios personales o de ánimo («estoy triste», «tengo hambre») no se rechazan: van al especialista de carta, que responde con calidez en una o dos líneas, sin consejos ni otros temas, y ofrece ayuda con el pedido. La carta se lista cuando el cliente la pide, pregunta qué hay o busca/pide algo que no está disponible: en este último caso se avisa brevemente y se muestra de nuevo la carta vigente completa, no solo algunas alternativas.

## Fuera del alcance actual

Pagos ejecutados dentro de WhatsApp, distancia por ruta, zonas por polígono, asignación de repartidores y seguimiento en mapa, promociones autónomas, audio/imagen y reservas. Concierge no implementa lógica fiscal ni sustituye Caja/POS.

## Venta por mostrador

- Una mesa marcada «Venta por mostrador» en Admin Web usa el mismo QR y el mismo
  bot. foods-backend decide el flujo por el tipo del QR; Concierge solo lo refleja
  (`mode="counter"`).
- El bot identifica el local y al cliente por su número: saluda por su nombre si
  ya es cliente y, si es nuevo, le pide el nombre con el que caja lo llamará.
- El cliente elige productos y cantidades con la carta real, confirma y cada
  confirmación crea un pedido independiente (código `PED-…`) que va a preparación
  y a la bandeja de caja. No hay cuenta abierta, mozo, pago previo ni delivery;
  se recoge en el mostrador y se paga en caja.
- Cuando Cocina/Barra termina el pedido, el cliente recibe por WhatsApp «Tu
  pedido PED-… está listo» con el mostrador del local (también en recojo).
- Cada número tiene su propia sesión; varios clientes pueden pedir a la vez con
  el mismo QR.

## Consentimiento promocional

Después del primer pedido confirmado, Concierge pregunta una sola vez por
empresa, al final de la confirmación y sin interrumpir la compra: «¿Te gustaría
recibir ofertas, promociones y beneficios exclusivos de [restaurante] por
WhatsApp?», con las opciones «Sí, quiero recibirlos» y «No, gracias». La
respuesta se reconoce sin el modelo en el siguiente mensaje; cualquier otro
mensaje continúa el pedido. «Ya no quiero promociones» retira el consentimiento
en cualquier momento. foods-backend guarda pregunta, respuesta y retiro con fecha
y el texto literal del cliente; quien rechaza o retira queda fuera de campañas.
Se pregunta solo en la venta por mostrador; la mesa y los pedidos sin mesa
conservan su flujo. El retiro funciona en cualquier modo.
