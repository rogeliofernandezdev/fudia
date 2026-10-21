# Flujo conversacional

## Inicio

El QR pasa por `/start/{token}`: Concierge valida la mesa y abre WhatsApp con un
mensaje amable y un código corto de la mesa, sin el token técnico:

> ¡Hola! 👋 Quiero ver la carta.
> Mesa 7 · K7P2

- WhatsApp no transporta datos ocultos y al escanear aún no se conoce el número
  del cliente: el primer mensaje es lo único que une el escaneo con el chat. Por
  eso lleva la referencia de la mesa.
- El código (4 caracteres sin 0/O/1/I/L, con al menos una letra y un número) se
  guarda en Redis (`fudia:concierge:startcode:{CÓDIGO}` → token del QR) durante
  `START_CODE_TTL_SECONDS` (600 s por defecto) con `SET NX`: un código vigente nunca
  se reasigna. Se resuelve antes del LLM, en cualquier parte del mensaje y sin
  distinguir mayúsculas; luego el token se valida en foods-backend como siempre.
- Si el cliente borra el mensaje y escribe otra cosa, el bot no adivina la mesa:
  sin contexto responde «Para atenderte en tu mesa, vuelve a escanear el QR y envía
  el mensaje tal como aparece.». Un mensaje de inicio con el código vencido recibe
  «Ese código de mesa ya venció…» y no reabre un local ni sienta al cliente.
- Si Redis no puede emitir el código, `/start` usa el mensaje anterior
  `FUDIA:<token>`; ese formato se sigue aceptando para QR y enlaces antiguos.
- Al abrir la mesa, la respuesta es determinista, sin modelo: el saludo, la
  carta real de `search_menu` (query vacío) y «¿Qué deseas pedir?». La carta se
  arma en `build_carta` (`src/services/carta.py`). Si hay menús o combos
  disponibles, cada uno va primero con su nombre y precio y, debajo, todos los
  platos que incluye (`get_combo`) agrupados por su categoría en orden de
  tiempos (entradas, sopas, segundos, bebidas y postres, según el nombre de la
  categoría con `category_course`; dentro de cada una, el orden de foods-backend),
  sin precio propio
  porque el del menú los cubre (solo se indica un recargo, «(+S/ 3.00)»);
  después «Extras» con los platos que no forman parte de ningún menú, también por
  categoría. Un plato del menú no se repite en Extras. Sin menús, van primero
  «Recomendados de la casa:» y luego cada categoría en el orden de foods-backend.
  Solo se listan los productos `available` o `low`; los demás (porciones del día
  sin cargar, sin stock, fuera de horario o agotados a mano) no aparecen. Si no
  queda nada que pedir, se responde «Hoy no tenemos platos disponibles.» y se
  ofrece avisar al personal. Un combo agotado no aparece y sus platos disponibles
  van a sus categorías; si el detalle de un combo falla, se muestra como plato
  suelto. `render_carta` arma este texto y lo comparten la bienvenida y
  `search_menu` (`cartaText`). El nombre de la mesa pasa por `table_label`, que
  no repite «Mesa». Ejemplo:

  ```
  🍽️ Bienvenido a La Panchita
  📍 Mesa 7

  🍱 Menú (S/ 12.00)
  🥗 Entradas:
  - Causa Rellena
  🍛 Segundos:
  - Ají de gallina
  - Lomo Saltado

  ➕ Extras
  🐟 Marino:
  - Ceviche (S/ 25.00)

  ¿Qué deseas pedir?
  ```

  La respuesta se limita a 3500 caracteres (WhatsApp corta en 4096): si no cabe
  todo, cierra con «Hay N opciones más. Escríbeme una categoría o un plato y te
  muestro el resto.». Si la carta falla
  o está vacía, se envía solo el saludo y la mesa queda abierta igual. El modelo
  atiende a partir del primer mensaje que escribe el cliente.
- El número del cliente queda prerregistrado en foods-backend al confirmar su
  primer pedido (ver `04_FUDIA_CONTRACT.md`).

## Atención y rondas

1. Resolver la mesa.
2. Consultar la carta con search_menu por nombre, descripción o categoría; query vacío permite consultar la carta. Preguntas generales («¿qué tienes para hoy?», «quiero comer») usan query vacío. Si un query no coincide con nada, la tool devuelve `matched=false` junto con la carta completa, para que el agente nunca concluya que no hay productos; solo `menuEmpty=true` indica una carta vacía. La carta completa trae además `combos` con los grupos y opciones de cada combo, y marca con `includedInCombo` los platos que ya son opción de uno: el agente los muestra dentro del combo y no como platos sueltos. Para mostrar la carta completa el agente envía `cartaText` tal cual, el mismo texto que la bienvenida; `noneAvailable=true` indica que hoy no hay nada que pedir. Los productos agotados nunca se listan ni se ofrecen; si el cliente pregunta por uno, el agente dice que hoy está agotado y ofrece lo que sí hay.
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

## Contexto de cada turno

Cada mensaje llega al especialista precedido por un bloque
«[Estado de la atención]» (`agent_message`), en mesa y en delivery:

- Modo: mesa abierta (mesa, local y sede) o pedido sin mesa con sus datos de
  entrega y lo que falta.
- Carrito de la ronda: una línea por producto con cantidad, opciones de combo,
  modificadores, nota y subtotal, y el total.
- Si hay una confirmación pendiente, si ya se enviaron pedidos en esta atención
  y si el personal ya fue avisado.

Junto con la conversación reciente, permite entender «otro igual», «el
segundo», «sin cebolla el lomo» o «quita la causa» sin volver a preguntar. El
bloque solo llega al modelo: el historial guarda el mensaje del cliente tal cual
y las tools que exigen confirmación explícita leen sus palabras literales.
`prompts/system.md` («Entender el contexto») pide interpretar la intención, no
las palabras sueltas, y hacer una sola pregunta corta si una referencia es
ambigua.

Cada especialista (`prompts/menu_agent.md`, `order_agent.md`,
`service_agent.md`) está organizado por intenciones del cliente (ver la carta,
buscar algo, recomendar, agregar, referirse a algo ya dicho, cambiar, confirmar,
cuenta, estado, personal) y cierra con ejemplos correcto/incorrecto tomados de
conversaciones reales. Cuando una conversación real sale mal, se agrega aquí
como ejemplo y en `evals/` como caso, para que no se repita.

## Formato de los mensajes

Todas las respuestas son texto plano de WhatsApp, con la misma estructura en las
respuestas fijas del código y en las del modelo (`prompts/system.md`):

- Sin asteriscos, guiones bajos, almohadillas ni negritas.
- Varios elementos: una línea de título que termina en «:» y debajo un elemento
  por línea con «- ». Producto: «- Nombre (S/ 0.00)». Línea de pedido:
  «- 2 × Nombre». Carta con menú: «Menú (S/ 12.00)» con sus platos por
  categoría y sin precio propio y luego «Extras» con el resto, como en el ejemplo de Inicio.
- Párrafos cortos separados por una línea en blanco y una sola pregunta al final.
- Íconos con moderación, como máximo uno al inicio de un título o bloque: 🍽️
  carta, 🧾 pedido o cuenta, ✅ confirmado, 📍 mesa o ubicación, 🛵 delivery, ⏳ en
  preparación. En la carta, cada categoría lleva un ícono de comida según su
  nombre (`category_icon` en `src/services/carta.py`: 🥗 entradas, 🍛 segundos o
  fondos, 🐟 marino, 🍰 postres, 🥤 bebidas, ☕ café…, 🍴 si no reconoce el nombre),
  🍱 el menú o combo, ➕ los extras y 👨‍🍳 los recomendados. Nunca íconos en cada
  plato ni caras, manos, corazones, estrellas, fuego o celebraciones.

Como red de seguridad, `whatsapp_text` (`src/services/formatting.py`) limpia la
respuesta del modelo antes de enviarla: quita las marcas de markdown, convierte
viñetas (`*`, `•`, `·`) en «- », agrega «:» a un título que encabeza una lista y
elimina los emojis informales.

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