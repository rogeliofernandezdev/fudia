# Delivery Fudia: centrales de reparto por WhatsApp

Fudia trabaja con centrales de delivery registradas por Plataforma en un orden
global. Delivery Fudia se activa por empresa, y todos sus locales lo usan.
El backend filtra ese orden por el departamento, provincia y distrito del local
de origen. Fudia administra exclusivamente las centrales y sus coberturas;
ninguna empresa las registra. Sin distrito o cobertura el pedido avisa a caja.
foods-backend decide a quién se ofrece cada pedido y en qué estado está
(`foods-backend/docs/04_DATA_MODEL.md`, migración 000113). Concierge solo envía
los mensajes y aplica los botones que pulsa la central.

## Flujo

1. El pedido de delivery (por llamada o por WhatsApp) se confirma en caja.
   foods-backend lo ofrece a la 1.ª central del orden global.
2. `PartnerMessageRelay` (en cada worker, cada `DELIVERY_PARTNER_POLL_SECONDS`,
   5 s por defecto) reclama `GET /v1/integrations/concierge/delivery-messages/pending`.
   Esa misma consulta avanza la cascada en foods-backend.
3. La oferta llega al WhatsApp de la central con «Aceptar» y «Rechazar».
   Concierge confirma `/dispatched` y desde ese momento corren 3 minutos.
4. «Rechazar», o 3 minutos sin respuesta, pasa el pedido a la siguiente central.
   Si ninguna acepta, la campana del local avisa a caja, que puede reintentar o
   usar su delivery propio.
5. Al aceptar, la central recibe «listo para recoger» con el botón «Recoger»
   cuando cocina termina. Si cocina ya había terminado, lo recibe de inmediato.
6. «Recoger» pasa el pedido a «En camino». Si el cliente pidió por WhatsApp,
   recibe el aviso `order_on_the_way` («🛵 Tu pedido … va en camino»).
7. Si el pedido se cancela o el local pasa a delivery propio después de que una
   central aceptó, esa central recibe un aviso de cancelación sin botones.

La central paga al restaurante solo el costo del pedido (`subtotal`) al
recogerlo. Al entregarlo cobra al cliente el pedido más su delivery (`total`),
y se queda con `deliveryFee`. Fudia no gestiona ese cobro.

## Botones y respuestas

Cada botón lleva el payload `fdx:<accept|reject|pickup>:<actionToken>`. El
webhook lo reconoce, tanto en el botón de una plantilla (`button.payload`) como
en un botón interactivo (`button_reply.id`). Lo encola como `InboundMessage`
con `partner_action`, sin pasar por el modelo ni por la sesión de un comensal.

`ConciergeService` llama `POST /v1/integrations/concierge/delivery-actions` con
el número que pulsó. foods-backend solo acepta el WhatsApp de la central
ofertada, y responde con el resultado. Concierge redacta la respuesta:

- **accepted / already_accepted:** «✅ El pedido … es tuyo.» Un reintento recibe
  el mismo texto.
- **rejected:** «Entendido, no tomarás el pedido …».
- **taken:** otra central lo tomó o ya no se le ofrece.
- **expired:** pasaron los 3 minutos.
- **cancelled:** el pedido ya no requiere delivery.
- **not_ready:** cocina aún no termina.
- **picked_up:** «🛵 Registramos que recogiste el pedido …».
- **not_found:** el token no existe o el número no es el de la central.
- **payment_conflict:** el local registró pagos antes de aceptar. El pedido queda
  para reparto propio; se indica no recoger ni cobrar al cliente y coordinar con
  el local. Los botones de una asignación retirada o vuelta anterior son `taken`.

Esta respuesta se envía como texto libre: la central acaba de escribir, así que
está dentro de la ventana de 24 h. La cola deduplica el botón por `message_id`.
Si el backend falla, la cola reintenta, y foods-backend no duplica el efecto.

## Envío

| Caso | Con plantilla configurada | Sin plantilla |
| --- | --- | --- |
| Oferta | Plantilla con 2 botones de respuesta rápida | Mensaje interactivo con Aceptar/Rechazar |
| Listo para recoger | Plantilla con 1 botón | Mensaje interactivo con Recoger |
| Cancelado | Plantilla sin botones | Texto |

Una central normalmente no le escribió al bot en las últimas 24 h, así que en
producción **se necesitan las plantillas**. El modo sin plantillas sirve para
probar con un número que acaba de escribir al bot.

Si Meta rechaza un envío con certeza, el mensaje no se confirma y vuelve en
60 s (hasta 5 intentos). Si el envío es incierto (Meta pudo haberlo recibido),
se confirma igual: así no se repite una oferta que corre contra el reloj.

## Plantillas para crear en Meta Business Manager

Crear en WhatsApp Manager › Plantillas de mensajes, categoría **Utilidad** e
idioma **Español** (o el que se configure en `WHATSAPP_TEMPLATE_LANGUAGE`).
Configurar sus nombres en Concierge:

- `WHATSAPP_DELIVERY_OFFER_TEMPLATE`
- `WHATSAPP_DELIVERY_READY_TEMPLATE`
- `WHATSAPP_DELIVERY_CANCELLED_TEMPLATE`

En GCP son variables del repositorio que el workflow pasa al despliegue.

Los parámetros `{{n}}` llegan siempre en una sola línea. Los botones deben
ser de **respuesta rápida** (no de URL ni de llamada) y respetar el orden
indicado.

### 1. Oferta (`fudia_delivery_oferta`)

Cuerpo:

```text
🛵 Nuevo delivery {{1}}
Recoge en: {{2}}
Entrega en: {{3}}
Cliente: {{4}}
Pedido: {{5}}
Pagas al local: {{6}}
Cobras al cliente: {{7}}
Responde en 3 minutos.
```

Botones de respuesta rápida, en este orden: `Aceptar`, `Rechazar`.

Ejemplos para la revisión de Meta:

1. `PED-0142`
2. `La Rinconada · Miraflores (Av. Larco 123)`
3. `Av. Perú 100 · Ref.: Puerta azul · 2.4 km · https://maps.google.com/?q=-12.121900,-77.029700`
4. `Ana · 999123456`
5. `2 × Lomo saltado (Sin ají), 1 × Chicha morada`
6. `S/ 60.00`
7. `S/ 65.00 (incluye tu delivery S/ 5.00)`

### 2. Listo para recoger (`fudia_delivery_listo`)

Cuerpo:

```text
✅ El pedido {{1}} está listo para recoger en {{2}}.
Paga al local: {{3}}.
Pulsa Recoger cuando lo tengas.
```

Botón de respuesta rápida: `Recoger`.

Ejemplos: `PED-0142`, `La Rinconada · Miraflores (Av. Larco 123)`, `S/ 60.00`.

### 3. Cancelado (`fudia_delivery_cancelado`)

Cuerpo:

```text
El pedido {{1}} de {{2}} ya no requiere tu delivery.
No lo recojas.
```

Sin botones. Ejemplos: `PED-0142`, `La Rinconada · Miraflores (Av. Larco 123)`.

Los nombres de ejemplo pueden cambiar. Lo que importa es que coincidan con
las variables, y que el orden de los parámetros y los botones sea el indicado.

## Despliegue

Desplegar Concierge **antes o junto** con foods-backend. Una versión anterior de
Concierge ignora el aviso `order_on_the_way` y no consulta la bandeja de
centrales: los pedidos quedarían ofrecidos sin mensaje hasta vencer.

## Pruebas

`tests/test_delivery_partners.py` cubre:

- Payloads de los botones.
- Parámetros de plantilla en una línea, con lo que la central paga al local y lo que cobra al cliente.
- Texto completo de la oferta.
- Respuestas idempotentes.
- Envío con plantillas y sin ellas.
- Reintento ante fallo y confirmación ante envío incierto.
- El webhook separa los botones de la central de las respuestas rápidas del comensal.
- El servicio aplica el botón sin modelo ni sesión, y el reintento no vuelve a llamar al backend.
