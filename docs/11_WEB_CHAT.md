# Chat web propio desde el QR

Tercera alternativa de atención: `FUDIA_WEB`, junto a `META_CLOUD_API` y
`WHATSMEOW`. Se selecciona desde **Plataforma → Países y WhatsApp → Proveedor**.
Un solo canal inicia atenciones por país. Cambiarlo conserva los QR impresos,
los pedidos y las rutas históricas de conversaciones anteriores.

## Experiencia

El mismo enlace público del QR resuelve el país y abre `/chat/{qr}` directamente,
sin lanzador flotante ni login. La referencia aprobada es una conversación clara
familiar (2026-11-09; la primera versión era oscura): fondo cálido, cabecera del
restaurante en gris claro, mensajes recibidos en blanco a la izquierda y enviados
en verde claro a la derecha, horas y escritura fija abajo. Identidad propia, sin
logotipos ni fondos copiados de WhatsApp.
Carta con imágenes reales del catálogo, búsqueda, Agregar, opciones de combos y
modificadores, carrito y confirmación explícita. No se muestran precios inventados.
El QR decide mesa, mostrador o pedido del local; nunca lo decide el texto del cliente.

Los pedidos de mesa/mostrador pueden registrarse sin teléfono. En mostrador se
solicita el nombre. El QR del local conserva el flujo de recojo/delivery, cuyos
datos de contacto siguen siendo obligatorios cuando el backend los requiere.
Para delivery existe el atajo «Ubicación» (compartir ubicación), que solicita
permiso al navegador solo al pulsarlo; la cotización la calcula el backend.
Las actualizaciones llegan mientras el chat esté abierto; no hay notificaciones
push fuera del navegador. No se han implementado adjuntos, audio ni videollamadas.

Armar el pedido no cierra el panel abierto: «Agregar» deja la carta a la vista
para sumar varios platos seguidos y quitar un plato deja «Mi pedido» abierto.
Cada toque deja una sola burbuja, la respuesta del local con el plato y lo que
lleva el pedido («Agregué 1 × Lomo saltado. Llevas 2 productos · S/ 65.00.»);
el toque en sí no se repite como mensaje del comensal. Enviar un texto, revisar
o confirmar sí cierran el panel para mostrar la respuesta.

El estado se consulta cada 1,2 s mientras el local responde, cada 4 s durante el
minuto siguiente a la última actividad y cada 12 s con el chat quieto; al volver
a la pestaña se consulta de inmediato. En Concierge esa consulta reutiliza por
30 s la mesa del QR y recuerda el estado de los mensajes ya terminados, de modo
que un chat en reposo no llama a foods-backend en cada consulta. Enviar un
mensaje, abrir la carta o las opciones siempre vuelve a validar el QR.

## Sesión, permisos e idempotencia

- BFF del mismo origen, peticiones POST y rechazo de origen externo. El origen
  del navegador se compara con el host que pidió (`X-Forwarded-Host` o `Host`),
  no con la URL interna del servidor: detrás del proxy público esa URL es
  `0.0.0.0:puerto` y rechazaría todos los chats. La cookie es `Secure` cuando
  el origen del navegador es HTTPS.
- Abrir la página no cuenta una conversación: la bienvenida es determinista y
  la conversación se registra en foods-backend con la primera interacción del
  comensal (mensaje, botón o ubicación). Si foods-backend la rechaza, el chat
  responde que no puede tomar el pedido «por este chat» y lo vuelve a consultar
  en la siguiente interacción.
- Cada escaneo lleva su propia referencia (`/chat/{qr}?s=…`, generada por
  Concierge en `/start`); recargar la página la repite. Un escaneo nuevo solo
  abre otro contexto cuando la atención de ese QR terminó (cuenta cerrada, tres
  horas sin mensajes o el chat ya pasó a delivery/recojo). Con la atención
  vigente conserva carrito, confirmación y conversación, sin otra bienvenida.
  Así el comensal vuelve a su mesa reescaneando, y recargar el chat desde casa
  no lo sienta en una mesa.
- Las sesiones nuevas se limitan por QR (`WEB_CHAT_SESSION_LIMIT` por
  `WEB_CHAT_SESSION_WINDOW_SECONDS`, 60 cada 10 minutos por defecto). Al
  superarlo, quien abre el chat por primera vez ve «Hay muchas personas abriendo
  este chat…»; las sesiones ya abiertas siguen funcionando.
- `FUDIA_CONCIERGE_API_KEY` solo en el servidor; nunca `NEXT_PUBLIC_`.
- Cookies firmadas, HttpOnly, SameSite=Lax, Secure con HTTPS. Una identidad de
  navegador y una capacidad opaca distinta por QR/canal. No se usa un teléfono ficticio.
- Redis vincula la capacidad al visitante, QR y ruta; renovar la página conserva
  el carrito. El historial conversacional y las capacidades son efímeros.
- El backend Go resuelve empresa/local y autoriza cada operación. Un texto con
  otro QR no puede cambiar el alcance de esta sesión web.
- Los botones entran en la misma cola y herramientas deterministas de Concierge.
  El journal y los locks cercados existentes siguen protegiendo los efectos.
- Reintentar conserva el identificador del mensaje. El outbox deduplica la
  entrega web. Confirmar exige además la referencia del resumen vigente;
  una pestaña desactualizada no confirma un carrito diferente.
- PostgreSQL sigue generando todos los UUID persistidos. La identidad web no
  acredita un teléfono ni se fusiona con un cliente existente por un número escrito.

## Configuración y despliegue

1. Aplicar `000123_fudia_web_channel` y `000124_web_partner_transport`.
2. En Concierge HTTP **y worker**, configurar `WEB_CHAT_URL` con el origen HTTPS
   público de foods-admin-web, sin ruta ni query. En desarrollo se admite localhost.
   Opcional, solo en el HTTP: `WEB_CHAT_SESSION_LIMIT` y
   `WEB_CHAT_SESSION_WINDOW_SECONDS` si un enlace de delivery muy difundido
   necesita más de 60 sesiones nuevas cada 10 minutos. El proxy público debe
   conservar el `Host` del dominio o enviar `X-Forwarded-Host`.
3. En el servidor de foods-admin-web, configurar `FUDIA_CONCIERGE_URL` y la misma
   `FUDIA_CONCIERGE_API_KEY` que Concierge/backend. Nunca enviar esta clave al navegador.
4. En GCP, agregar la referencia existente de esa clave a `GCP_FRONTEND_SECRETS`
   y permitir a la identidad del frontend leer ese secreto. El desplegador acepta
   esa referencia; el workflow de Concierge propaga `WEB_CHAT_URL` al HTTP y worker.
5. Seleccionar **Chat web de Fudia** y activar el canal del país. No pide número
   ni vinculación de dispositivo. El servicio verifica que el origen esté configurado.

Meta y whatsmeow conservan sus credenciales y conexiones para sus conversaciones
históricas. Para centrales de delivery, que siguen atendiendo por WhatsApp, se
conserva una ruta externa del país: nunca se les envía a un visitante del chat.
Cuando el origen del cliente es web, se usa el último transporte WhatsApp
registrado del país (se prefiere el actual si existe). Debe seguir configurado y
conectado. Sin un transporte externo no se pueden contactar centrales por WhatsApp;
el personal debe gestionar el reparto. Los botones autenticados de centrales
pueden completar una operación por su ruta histórica; los chats nuevos de
clientes siguen exigiendo el canal activo.

El chat propio evita enviar los mensajes del cliente por Meta/whatsmeow. Gemini,
Redis, backend, imágenes y alojamiento mantienen sus costos correspondientes.

## Verificación local

`fudia-concierge/tests/test_web_chat.py` comprueba precio del backend, confirmación,
reintentos, aislamiento de QR/visitante, Redis y deduplicación de salida.
`foods-admin-web/tests/web-chat-bff.test.mjs` comprueba origen, cookies y que
credenciales/identidad del navegador no se conviertan en autoridad.
`web_chat_integration_test.go` valida dos países sin número y avisos anónimos
ligados al origen real en PostgreSQL.

La vista `tests/web_chat_preview.py` es exclusivamente una fixture local de QA:
restaurante ficticio, Redis temporal, herramientas/cola reales y ninguna llamada
al modelo o pedido a un restaurante real. No se despliega ni se importa desde producción.
La validación de viewport no reemplaza una prueba con teclado y áreas seguras en
un teléfono físico.


### Resultado de revisión, 2026-11-08

- Backend: suite completa con PostgreSQL 17; arquitectura, vet y build correctos.
  Migraciones 123–124 aplicadas solo en una base temporal. Flujos de delivery,
  país y web repetidos tras aplicar la 124, correctos.
- Concierge: 312 pruebas correctas, 1 prueba de carga omitida; Redis real para
  integración y web. Ruff sin errores, mypy correcto (63 archivos), wheel y sdist generados.
- Frontend: 729 pruebas correctas; 53 comprobaciones de contrato, BFF y tokens
  repetidas tras los últimos cambios. TypeScript y build correctos. Lint sin
  errores; conserva 7 advertencias anteriores ajenas al chat.
- Infraestructura: 64 pruebas correctas.
- Navegador: 390×844, 390×480 y 1280×800. Sin desbordamiento horizontal;
  en móvil, controles comprobados de al menos 44 px y escritura siempre visible.
  Flujo probado con restaurante ficticio: agregar, revisar moneda/total, confirmar,
  recargar y conservar historial sin repetir la bienvenida ni registrar otro pedido.
- No se enviaron mensajes a números reales, no se llamó a Gemini en la fixture,
  no se desplegó y no se modificó la base habitual. Falta la prueba con teclado
  y áreas seguras en un teléfono físico.
