# Estado e integración

## Cambio de local

El selector consulta `locations/available`; el backend incluye todos los locales
activos de la empresa para el rol reservado Administrador de empresa y mantiene
los otros roles limitados a sus asignaciones. El plan no decide permisos del
usuario. La caché usa `available-locations` con empresa y usuario, se revalida
al recuperar el foco si tiene más de un minuto y se invalida al guardar/desactivar locales o modificar
usuarios/roles. Cambiar de local conserva el flujo de sesión y limpia los datos
remotos antes de cargar el contexto autorizado del destino.

## Fotografías de producto en GCP

Preview blob inmediato, validación JPEG/PNG/WebP, 5 MiB y 4000×4000 px antes de
autorizar. JSON por BFF; bytes directamente a Storage con política POST firmada.
Confirmar upload y guardar producto con imageId; no esperar conversión.
imageKey/imageStatus/imageExpiresAt/URLs son datos de lectura, nunca se reenvían
como referencia persistente. Guardado fallido reutiliza la subida confirmada.

PROCESSING usa preview local por empresa/local mientras exista, u original
temporal al reabrir. Detalle consulta cada 2 s mientras PROCESSING; READY usa
medium WebP hasta 600 px. Listados/comanda usan thumbnail hasta 200 px; no se
genera ni descarga versión 1200 px. FAILED conserva fallback y ofrece otra foto.
Estados/error y reintento usan componentes existentes; sin recolorear la paleta.
URLs se renuevan al recuperar foco y antes de expirar; el navegador no hace una
petición para firmar cada foto. No almacenar URLs firmadas ni previews en localStorage.
La sesión HttpOnly/token opaco y el control de permisos permanecen intactos.
Contrato y recursos en ../../foods-infrastructure/docs/10_PRODUCT_IMAGES.md.

## Fotografías en despliegues sin proxy externo

Las URLs persistidas `/uploads/products/{uuid}.{png|jpg|jpeg|webp}` se conservan.
Un Route Handler GET/HEAD usa `FOODS_API_URL` de runtime para transmitir bytes del
backend; no devuelve redirects, HTML ni cookies. Rechaza nombres fuera del patrón,
limita el timeout y no fija una foto mutable en caché. Cloud Run monta Cloud
Storage en el backend; el frontend no obtiene credenciales de bucket ni SQL.
La ruta es infraestructura BFF y no modifica el contrato de subida ni el catálogo.

## Ticket de pago de 80 mm

POS abre la vista previa únicamente en el callback `onClose` del aviso de éxito,
después de la respuesta de `POST payments`. Consulta nuevamente el detalle y
verifica que el pago confirmado esté presente; imprimir, descargar y reintentar
son de solo lectura y nunca repiten cobros. POS y Ventas reutilizan el mismo
diálogo para copias. La caché incluye empresa/local/pedido, sin catálogos locales.
`receiptContext` viene del backend y describe identidad actual y moneda del local,
no una instantánea fiscal histórica. No se recalculan impuestos desde preferencias.
El PDF tiene ancho 80 mm y altura adaptable; los documentos largos continúan sin
omitir líneas. El visor muestra el Blob real y libera su URL al cerrar. Imprimir
usa un documento aislado con la misma composición y abre el diálogo del navegador;
no requiere un puente ESC/POS ni configura la impresora automáticamente.

## Cierre de caja con informe

`useCashReport` consulta el informe por empresa/local/turno y presenta skeleton,
error con reintento y vacíos explícitos. El cierre carga un resumen fresco antes
del arqueo, sin reconsultar al recuperar foco ni borrar el conteo. `Cerrar turno`
exige clic, bloquea doble envío y espera el API; no tiene reintento
automático de escritura. El aviso exitoso se muestra tras confirmar y su callback
global `onClose` abre la vista previa después de retirar el aviso, sin apilarla
con el formulario anterior. Se invalidan Caja, informe y Dashboard.

La generación PDF se carga bajo demanda y utiliza moneda/zona/nombres del
snapshot, no la configuración actual. Un fallo de lectura/logo/PDF permite
reintentar solo el informe: nunca repite el cierre. El Blob se libera al cerrar
la vista. Turnos conserva un acceso de lectura al detalle y PDF; los cierres
anteriores sin snapshot se identifican como reconstruidos.

TanStack Query administra estado remoto por organización/local. Presentación no
contiene rutas HTTP ni DTO crudos. Toda dependencia muestra skeleton, error y vacío.

Los modales de respuesta se muestran desde `onSuccess`/`onError` o después de
una petición esperada con `await`, nunca al enviar ni durante validación local.
`FeedbackProvider` cierra primero los diálogos de origen registrados en `Dialog`.
El éxito y el error remoto sustituyen al formulario; las validaciones de cliente
se muestran en línea sin cerrar ni enviar. No se convierte un rechazo remoto en
éxito aunque una vista anterior muestre un cambio.

Revisión de Compras comparte un bloqueo síncrono entre cambio de estado,
aprobación y cancelación. Evita otra escritura antes del siguiente render y
durante el refresco de órdenes, detalle y dashboard. No reintenta automáticamente
la transición; refresca el estado incluso tras un rechazo para no repetir una
acción desde datos obsoletos. El aviso depende del resultado de la escritura,
no del éxito de la consulta posterior.

El registro por lote de mesas bloquea campos y acciones durante el envío y no
reintenta automáticamente el POST. El éxito se muestra únicamente tras recibir
la confirmación del API. Si se pierde la respuesta o falla el servicio, conserva
el borrador y avisa «No pudimos confirmar el registro», refrescando las mesas para
comprobar si se guardaron antes de repetir. No presenta como rechazado un resultado
incierto. El BFF conserva el identificador de correlación y usa un límite de 40 s,
posterior a los límites de procesamiento y respuesta del backend. Si el backend
responde `tables_not_saved` (PostgreSQL rechazó el lote y no guardó nada), se
muestra ese error específico como «No se pudieron registrar», sin tratarlo como
incierto. Todo error 5xx del lote agrega «Código de seguimiento» con el
`correlationId`, que coincide con el evento `api_error` del log del backend.

Las transiciones de Cocina invalidan comandas, pedidos y el detalle afectado.
La respuesta confirmada se refleja en el panel correspondiente sin modal de
éxito; durante el envío se bloquea la acción y un fallo conserva el aviso de error.

La hoja de Menús y combos valida todo al guardar (`validateComboDraft`, que reutiliza
las reglas por sección de `validateComboStep`). Muestra el primer error junto a su
campo (bajo la línea si es de Reservar o Costo extra), desplaza la hoja hasta el campo
y lo enfoca; cualquier cambio posterior limpia el aviso. No guarda
registros parciales. X/Cancelar solicitan confirmación cuando hay cambios sin
guardar; la recarga utiliza el aviso nativo del navegador.

## Autocompletes remotos

Unidad de inventario se carga desde `purchase-units`, sin opciones locales ni fallback
inventado. La caché de `inventory-units` incluye empresa. Presenta skeleton,
error con Reintentar y vacío explícito. Con `purchases.manage`, el botón «Nueva
unidad» abre un diálogo compartido; Guardar persiste código/nombre, selecciona
la respuesta confirmada y refresca el catálogo sin perder el borrador del artículo.
El diálogo se monta fuera del formulario padre y no guarda con Enter. Paquete o
Bolsa como unidad base no reemplazan las presentaciones de ingreso y sus factores.

Los autocompletes de catálogos grandes no cargan el catálogo completo. La apertura inicial usa una página pequeña de hasta 10 opciones. Las búsquedas no consultan con menos de 3 caracteres; desde 3 caracteres delegan el filtro al backend y recorren la paginación necesaria para reunir todas las coincidencias. Las opciones ya seleccionadas en formularios multirregistro se excluyen en cliente sin alterar el resultado remoto.

## Performance de consultas en cliente

Equipo del turno reutiliza el selector buscable `react-select` y los estilos de
lookup compartidos. Filtra por nombre, sin distinguir tildes ni mayúsculas,
sobre los colaboradores del local ya cargados desde `cash-users/options`:
escribir no dispara peticiones. Esta lista de equipo no es un catálogo masivo.
La caché incluye empresa, local y usuario; asignar/quitar invalida las opciones.
El API mantiene la autorización y las exclusiones definitivas al asignar.
Errores de opciones o equipo bloquean la asignación y ofrecen Reintentar.

Locales y Mi perfil comparten la lectura de suscripción, con caché por empresa
y usuario y refresco al recuperar foco. «Nuevo local» consulta `maxLocations`
y el uso activo devueltos por el API; no deduce capacidad por nombre de plan,
filas de una página o total de locales inactivos. La carga, el error y la recarga
que sigue a un cambio (alta, edición o desactivación invalidan la suscripción)
bloquean el alta; el refresco periódico (cada 60 s) ya no deshabilita los botones un
instante (`isInvalidated` de la consulta). El error ofrece reintento. Crear, editar y desactivar invalidan el uso.
El backend conserva la validación transaccional definitiva del límite.

Usuarios reutiliza `useSubscriptionCapacity`, igual que Locales: máximo y uso
activo proceden de la suscripción, no de la tabla filtrada. El cupo bloquea
«Nuevo usuario» y reactivar inactivos; editar, desactivar y crear roles permanecen
independientes. Alta, edición y cambios de estado refrescan el cupo. La confirmación
compartida admite `confirmDisabled` separado de `pending`: si cambia la capacidad
mientras está abierta, deshabilita Activar y explica el motivo sin impedir Cancelar.

Las búsquedas de listados no disparan una petición por pulsación. Deben reutilizar
`useDebouncedValue` con una ventana aproximada de 300 ms y mantener el texto
visible separado del valor consultado.

TanStack Query conserva la caché por clave; las mutaciones invalidan únicamente
las familias afectadas. Los diálogos pesados que no forman parte del primer render
se cargan con `next/dynamic` cuando la acción realmente se abre.

El prefetch de datos se incorpora solo en rutas frecuentes y cuando la medición
demuestre que reduce latencia sin duplicar tráfico.

Refrescos y cargas revisados (2026-11-04):

- Comanda: los menús elegidos se piden juntos con `GET order-combo-details?ids=`
  (hasta 20 por petición; si uno falla, falla el lote como antes `Promise.all`).
  Antes era una petición por menú cada 10 s.
- Catálogo de la comanda: con una sola página cargada se refresca cada 10 s como
  antes; con varias, TanStack las volvería a pedir todas, así que solo se refresca la
  disponibilidad de lo cargado (`order-catalog-availability`, lotes de 100) y la carta
  completa al recuperar foco.
- Cocina refresca cada 10 s mientras la pantalla está visible; una pestaña oculta o
  minimizada deja de consultar y se actualiza al volver.
- Al recuperar foco, la sesión y Mi perfil se revalidan siempre (decisión de
  seguridad); avisos y locales disponibles solo si están vencidos.
- `LocationMap` (Mapbox y su CSS), `TimezoneSelect` y el reparto de propinas de Caja
  (`react-select`) se cargan con `next/dynamic` al mostrarse, no con la página. En el
  build de producción, la carga inicial (sin comprimir) bajó: Caja 471 → 381 KB de
  JS; Clientes 221 → 169 KB de CSS; Organización 454 → 370 KB de JS y 224 → 171 KB
  de CSS. Ninguna de esas rutas trae ya `react-select` ni el CSS de Mapbox al abrir.
- Se mantienen a propósito: el perfil del restaurante consulta cada 60 s para
  renovar URLs firmadas antes de que expiren, y Productos cada 10 s con el listado visible.

Revisión de rendimiento (2026-11-20):

- Pantallas en vivo por SSE: ver «Actualizaciones en vivo». Con el canal
  conectado, Cocina, Salón, comanda, Pedidos, POS, Caja, Ventas, Reservas,
  Dashboard, Productos, Menús, campana, avisos y pagos de Fudia Pay se refrescan al
  confirmarse un cambio del local y su consulta periódica baja a 60 s; los
  intervalos de 10, 15 y 30 s de este documento son los de respaldo sin canal.
- `regional-format` reutiliza un `Intl.NumberFormat`/`DateTimeFormat` por locale y
  opciones: crearlos costaba más que usarlos en cada celda de dinero y fecha.
- El proxy `/api/admin/[...path]` transmite el cuerpo del API tal como llega (también
  binarios) en vez de esperarlo completo, y cancela la llamada al API si el navegador
  abandona la petición.
- Los listados con búsqueda o paginación (Pedidos, Ventas, POS, Clientes,
  Productos y la carta de la comanda) pasan el `signal` de TanStack: cambiar de
  filtro cancela la petición anterior en vez de dejarla correr.
- El selector de opciones del editor de menús descarga las páginas restantes de la
  carta a la vez (conocido el total) y no la vuelve a pedir al recuperar foco; las
  opciones elegidas siguen refrescándose solas.
- El contexto de sesión memoiza su valor: los consumidores de `useSession()` no se
  vuelven a renderizar cuando el proveedor se renderiza sin cambios.
- El PDF de cierre de caja y los QR de mesas usan `logo-mark.png` (16 KB) en lugar
  del original de 1254 px (889 KB) que se decodificaba y comprimía en cada PDF.
- La suscripción (plan y cupos) se revisa cada 60 s y al recuperar foco si el dato
  tiene más de un minuto, en lugar de cada 30 s y en cada foco.

## Actualizaciones en vivo

`LiveUpdates` (montado en el layout de administración) abre
`EventSource("/api/admin/events")` para el local de la sesión; la ruta BFF
`app/api/admin/events` reenvía el flujo del API sin plazo ni almacenamiento y lo
corta al cerrar la pestaña o cambiar de local. Cada evento `change` trae los temas
confirmados (`orders`, `kitchen`, `payments`, `cash`, `notices`, `pay`,
`availability`, `catalog`, `delivery`, `salon`, `reservations`) y solo invalida las
consultas que los declararon.

Una consulta en vivo no fija `refetchInterval`: declara
`meta:{live:[temas],poll:intervalo}` (`poll` puede ser una función del estado, p. ej.
`false` al completar el pedido; `live` también, como la carta de la comanda, que con
varias páginas solo se recarga por cambios de productos). El `refetchInterval` por
defecto del `QueryClient` (`liveRefetchInterval`) usa `poll` sin canal y
`max(poll, 60 s)` con canal, como respaldo por si se pierde un aviso. Al conectar o
perder el canal se refrescan todas las consultas en vivo, con `cancelRefetch:false`
para reutilizar una lectura en curso. Si el API responde 503 (no está escuchando a
la base) o la conexión cae, las pantallas siguen con su consulta normal y el canal
se reintenta con espera creciente (5 → 60 s). La sesión, el perfil, el panel de
reparto y el chat web conservan su propia consulta: no dependen del local de la
sesión de administración.

## Fechas de venta en Menús y combos

`availableFrom`/`availableUntil` viajan al API como instantes ISO, pero el formulario
las escribe y muestra en la **hora del local activo**: `zonedInputToIso` e
`isoToZonedInput` (`domain/sale-window.ts`) usan `location.timezone` con `Intl`, no
el reloj del navegador. Así un menú vence a las 23:59 del local aunque se administre
desde otro país. Los calendarios piden únicamente el día: `changeSaleDate` completa
el otro límite si falta y ajusta un rango anterior que ya no contiene el día
elegido. `fullDaySchedule` prepara 00:00 y 23:59:59.999 locales; la conversión a
ISO conserva segundos y milisegundos, incluido el último minuto del día. Los
límites se convierten por separado para respetar días de 23 o 25 horas por cambios
de horario. La edición normaliza solo el borrador; no escribe al abrir. Guardar
invalida también `combo-detail` para retirar fechas y estados anteriores.
El listado (`GET /combos`) devuelve `outsideDates`, calculado en el
servidor, para mostrar «Fuera de fecha» en un menú activo que no se vende por fecha.

## Cantidades en Menús y combos

El selector usa `availableQuantity` del listado de Productos: `availableUnits` = 0
excluye la opción y, si el producto controla cantidad, es el máximo del tope.
«Lo disponible» depende del control del producto: porciones del día menos las
vendidas (platos por porciones), stock físico del local en Inventario (bebidas y
mercadería por inventario, p. ej. gaseosa con 24 unidades → «de 24») o sin número
(sin control: tope vacío, sin máximo ni sufijo).
Cada opción tiene un **«Tope»** opcional con sufijo «de N» (lo disponible hoy;
sin sufijo si el producto no controla cantidad, contrato `quota`): el máximo que ese
menú vende de esa opción por día operativo del local. El tope **no separa
porciones**: carta, comanda y menú consumen el mismo stock, y la opción se agota
cuando se acaba el stock aunque le quede tope. El contador se reinicia cada día,
también en menús que se venden «Siempre».

Selección manual y plantilla Menú del día comparten `createComboOption`, que
llena el tope con lo disponible hoy (parte entera); el usuario puede reducirlo o
vaciarlo («Sin tope»: vende hasta agotar). Sin control de cantidad queda vacío. La ayuda del encabezado lo explica.
La validación compartida acepta vacío (consume todo lo disponible) o un entero
de 1 a lo disponible: con 10 registrados, 20 se rechaza junto al campo
(«Escribe un entero de 1 a 10 o déjalo vacío.») y el backend lo vuelve a
validar al guardar (2026-11-03, pedido del usuario). La caché incluye
empresa/local; la carta completa se consulta al abrir la hoja y, mientras permanece
abierta, cada diez segundos se refrescan solo las opciones elegidas
(`GET products?productIds=`, lotes de 100, clave `combo-picker-chosen`), con la misma
cantidad que el listado. Antes se descargaba toda la carta cada diez segundos. Editar un menú conserva los topes persistidos.
Guardar no realiza un movimiento de stock.

## Estado de productos

Disponibilidad solicita un motivo nuevo antes de guardar un cupo, Agotar o
Reactivar. No usa la nota anterior como motivo ni envía al abrir el diálogo.
Reutiliza RHF/Zod, formulario CRUD, validación en línea, guardado explícito por
botón y bloqueo síncrono de duplicados. Éxito/error solo aparecen tras la
respuesta y cierran el diálogo mediante `onResponseClose` compartido.
La acción Historial con reloj abre historial remoto por producto, empresa y local, con
paginación compartida, fecha/hora regional, valores anteriores/nuevos, motivo
y autor. Conserva los eventos antiguos sin detalle con un texto explícito.
El historial separa «Cambio» de «Cantidad»: cupo anterior → nuevo y unidades
vendidas al cambiar, exclusivamente desde el evento. Los valores nulos son «Sin
registrar», nunca cero ni el saldo actual del producto. Un registro moderno sin
control de cantidad puede indicar «No aplica». Los registros de inventario no
inventan existencias históricas: el contrato no incluye snapshots de stock físico.
Lectura no exige `menu.manage`. Guardar invalida historial, disponibilidad,
productos, catálogo operativo, combos y dashboard.

El listado de Productos recibe `availableQuantity` e `inventoryUnit` del backend,
sin calcular existencias en el cliente ni cargar Disponibilidad por cada fila.
Su caché incluye el local; actualiza al recuperar foco y cada 10 segundos mientras
el listado está visible. Pedidos, Salón y ajustes de Disponibilidad invalidan
también Productos. La descripción sigue disponible para editar, pero no se muestra
en la tabla.

El alta con `quantityControl=portions` envía `initialPortionQuantity` positiva
en el mismo POST de producto. La API crea catálogo, disponibilidad del día/local
y auditoría en una transacción; no se encadenan dos peticiones ni se muestra éxito
parcial. El campo no se envía para Sin control ni en PATCH comercial. Un fallo
cierra el formulario al mostrar el error y no reintenta automáticamente. El éxito refresca productos,
disponibilidad, catálogo de comandas y dashboard. Los ajustes y el override Agotado
se mantienen exclusivamente en Disponibilidad, sin cantidades predeterminadas.

La tabla de Carta y productos permite activar un producto inactivo mediante
`PATCH /v1/admin/products/{id}/status`, enviando únicamente `active`. La operación
conserva la ficha comercial y el historial, exige `menu.manage` y refresca los
listados del catálogo, disponibilidad, comandas y selectores de abastecimiento.

## Agregar artículo en Compras

«Agregar artículo» abre en la búsqueda (`purchase-item-picker`, sobre
`purchase-inventory-items?q=`). Con dos o más caracteres consulta también
`purchase-linkable-products?q=` (`purchase-linkable-products`, hasta 20): productos
activos de la carta que aún usan cupo del día o ningún control, sin combo ni receta
activa. Elegir uno abre «Controlar con inventario» y envía
`POST purchase-inventory-items` con `productId`, unidad, presentaciones y stock
mínimo: el backend reutiliza el mismo producto (nombre, precio, categoría, imagen e
historial), lo pasa a Inventario físico y deja el stock en 0. «Crear artículo
nuevo» envía `newProduct` si «Se vende a clientes» está activo (categoría y precio)
o `newIngredient` si no; el interruptor nace activo sin el módulo `recetas` y
apagado con él. Al guardar se refrescan `purchase-inventory`,
`purchase-linkable-products`, `inventory`, `products` y `product-availability`.
Cada vista remota mantiene skeleton, error con Reintentar y vacío propio.

Los totales de la orden replican al backend: cada subtotal es
`round(cantidad × costo, 2)` (cantidad con hasta 3 decimales y costo con 4) y el
total suma los subtotales redondeados (`purchase-totals.ts`). Las presentaciones
distintas de la unidad de inventario exigen cantidades enteras al pedir, recibir
y devolver, con el mismo mensaje en las tres pantallas.

## Estado de categorías

En Compras, «Nuevo producto vendible» exige una categoría activa compatible con
`retail` o `both`, obtenida de `purchase-item-categories`. Si no existen, el campo
explica cómo resolverlo y ofrece «Nueva categoría» solo con `menu.manage`.
Reutiliza `CategoryDialog` del catálogo y `POST /categories`, sin ampliar permisos
ni inventar categorías. Tras guardar, selecciona el UUID confirmado y actualiza
las cachés de categorías; el borrador del artículo y la orden se conservan.

Las categorías inactivas ofrecen la acción Activar mediante
`PATCH /v1/admin/categories/{id}/status`. Solo cambia `active`, conserva nombre,
orden, uso y asociaciones, y exige `menu.manage`. La activación refresca
`categories`, `purchase-item-categories`, `order-categories` y `order-catalog`.
La desactivación sigue bloqueada cuando hay productos activos asociados.

## Alta manual desde Pedidos

«Nuevo pedido» abre un formulario breve de datos, con Mostrador inicialmente
seleccionado. Los tipos seleccionables (Mostrador, Recojo en local y Delivery)
toman valores y etiquetas de `manualChannelOptions` del API. Salón conserva su toma de comanda por mesa; WhatsApp
conserva el ingreso por Concierge y no se ofrece como origen manual.
Delivery exige cliente, teléfono válido y dirección; referencia, notas y costo
de envío completan la entrega. «Continuar» valida estos datos y abre `ComandaView`,
la misma toma de pedidos utilizada por Salón: carta, cantidades, elecciones,
notas y resumen. Delivery muestra cliente/dirección en lugar de mesa e incluye
el envío en el total. Recojo en local no solicita mesa, dirección ni envío. «Editar datos»
vuelve al formulario conservando productos, elecciones y notas. Avanzar o volver
no realiza peticiones de creación ni genera pedidos parciales. El catálogo
operativo es paginado y presenta la disponibilidad real del local.

`recojo` unifica pedidos presenciales y encargos para retirar. La **venta por
mostrador** (2026-11-02) se registra con `channel=mostrador`: un pedido sin mesa
ni mozo, sin pedir un mostrador físico, con nombre, teléfono y notas opcionales,
que va a preparación y se cobra en Caja en cualquier momento (preparación y pago
son independientes: `canChargeAccount` lo permite si `origin==="counter"`). El
API lo lee como `channel=recojo` (misma pestaña de Pedidos y Cocina, mismos
filtros y conteos) y expone `origin=counter`; la presentación lo identifica con
`originLabel` de `order-meta.ts` («Mostrador», icono `store`). Los pedidos
históricos guardados como `mostrador` también se muestran con ese origen. El
origen WhatsApp y los vínculos de Concierge se conservan sin inferirlos del
retiro. Los pedidos de mostrador creados por Concierge desde el QR de mostrador
siguen el mismo contrato.

`Table.serviceType` (`table`|`counter`) llega en `GET /tables` y se envía en el
alta por lote y en la edición. La bandeja del POS (`POSOrderSummary`) agrega
`origin` e `itemsText` (productos resumidos por el backend); el detalle
(`Order`, `KitchenTicket`) agrega `origin`. `Customer` expone
`marketingConsentStatus` y `marketingConsentUpdatedAt` de solo lectura.

La acción explícita «Registrar y enviar a cocina» hace POST al endpoint existente
`/v1/admin/orders` con `sendToKitchen=true`; no guarda al pulsar Enter desde un
campo. El servidor deriva empresa/local, recalcula precios y valida disponibilidad.
La petición bloquea campos, cierre y repetición; el aviso de fallo cierra el modal
mediante el cierre directo de respuesta, sin abrir una confirmación de descarte.
X/Cancelar confirma el descarte si hubo cambios. El éxito cierra el formulario,
selecciona el canal creado, limpia filtros e invalida Pedidos, Cocina, POS,
Dashboard y disponibilidad. No se registran pagos desde este formulario: el cobro
sigue en POS. Delivery continúa con Por entregar → En camino → Entregado.
Fuera del salón primero se cobra y luego se entrega (2026-11-14): con saldo
pendiente, el detalle muestra solo «Cobrar saldo» y no «Entregar»
(`nextOrderAction` + `awaitingPayment`); al quedar pagado, «Entregar» pasa a ser
la acción principal. El backend lo valida con `payment_required_before_delivery`.
En Delivery Fudia (`partnerHandlesDelivery`: central buscando o asignada) el
detalle no muestra «En camino» ni «Entregar»: los marca el repartidor y caja
solo cobra (backend `delivery_handled_by_partner`). El delivery propio conserva
ambas acciones.
El administrador de plataforma no ve «Cobrar» (`canCollectOrder`); el backend
responde `platform_admin_cannot_collect`.

## Responsable de mesa y autoría del cobro

Al guardar la comanda de una mesa, la API asigna al usuario autenticado como mozo.
Salón y Pedidos muestran «Mozo» y, cuando existen pagos, «Cobrado por», con los
nombres reales del servidor. Sus detalles reutilizan una franja de datos de lectura;
`modal-inline-metadata` presenta etiqueta y nombre a 14 px, con fecha al extremo
derecho y wrap sin truncar cuando falta espacio. No modifica la autoría ni el dato remoto.
Ventas y POS identifican además al autor de cada pago. Nunca se toma al cajero
del turno como autor de un cobro que realizó otro integrante.

Las acciones de atención combinan `orders.manage` con `waiterId`: otra persona
consulta el detalle sin editar, cancelar, enviar a cocina o confirmar la entrega.
No se requiere un paso adicional de asignación. Cobrar sigue reservado a
`cash.manage` y al turno vigente, independientemente de la responsabilidad del mozo.
La API vuelve a validar la propiedad en cada mutación. No se transfiere la mesa
por abrir su detalle ni al cobrarla. Cada nueva ocupación obtiene su propio responsable.

## Cuenta y entrega en Salón

La mesa permanece Ocupada desde el primer pedido hasta finalizar la atención.
El estado de cuenta (Abierta / Por cobrar / Pagada) se muestra separado del
estado de preparación, sin fusionar etiquetas.

Productos obtiene del backend Cocina, Barra y Sin preparación. Salón y Pedidos
muestran destino y estado por producto solo como lectura, sin botones por fila.
El pie presenta un único «Confirmar entrega» para todo el pedido cuando todos
los productos pendientes están por entregar. Usa `PATCH orders/{id}/status` con
`status=entregado`: el backend confirma todas las líneas listas en una transacción,
sin modificar las ya entregadas. No incorpora una alternativa de entrega individual.
Cocina/Barra preparan de forma independiente; Sin preparación nunca ingresa a KDS.

«Agregar productos» abre `ComandaView` en modo append sobre la misma mesa,
vacío de líneas anteriores; utiliza `requestKey` estable durante el reintento.
Solo la cuenta Abierta admite consumo adicional. «Cerrar cuenta» se habilita
después de entregar todos los productos, incluidos los de cada ronda adicional;
marca Por cobrar sin liberar la mesa. Una cuenta cerrada no admite más consumo
ni reapertura, aunque tenga saldo pendiente o pagos parciales.

Solo el mozo asignado con `orders.manage` puede agregar, cerrar o entregar;
otros mozos consultan el detalle. Cajero o mozo con `cash.manage` y turno asignado
registran pagos únicamente de cuentas cerradas con todos los productos entregados.
El backend valida cada producto bajo bloqueo del pedido, incluso si el estado
agregado o el cierre histórico dicen otra cosa. La liberación automática requiere
cuenta cerrada, pago completo y todos los productos entregados.

La confirmación global de entrega y el cierre actualizan directamente la vista,
bloquean sus botones durante la petición y conservan el modal de error.
La entrega global mantiene el bloqueo hasta finalizar el refresco de los datos,
para no permitir una segunda confirmación sobre el estado anterior.
Invalidan detalle, Pedidos, Salón, POS, Cocina/Barra, Ventas y Dashboard.
Los detalles abiertos se refrescan cada 10 segundos hasta `completedAt`.

Salón y Pedidos reutilizan `canCancelOrder`: un producto Sin preparación listo aún sin
servir puede cancelarse si no hay pagos ni preparación iniciada o entrega de
otros productos. El estado agregado Por entregar no significa que se haya preparado.
Pedidos históricos sin detalle de servicio conservan cancelación solo desde
Nuevo/Confirmado. Backend valida de nuevo bajo bloqueo y revierte cantidades
una sola vez; la autorización del mozo y `orders.manage` no cambia.


## Combinaciones de unidad y presentación de compra

Compras consulta `purchase-combinations?unit=...`, aislado por empresa y unidad.
No contiene una lista fija de envases ni infiere cuántas unidades contiene cada
uno. «Unidad de inventario» expresa el saldo; «Presentación de compra» expresa cómo
se compra. Recibir cinco paquetes de diez botellas suma 50 botellas; vender una
descuenta una, no un paquete.

Seleccionar unidad sugiere la combinación predeterminada del catálogo remoto.
El contenido se exige por artículo (hasta tres decimales positivos). El formulario
comparte `PurchasePresentationsField` para altas y artículos existentes: muestra
una presentación y su contenido, sin «Otra presentación» ni controles de
«Predeterminada». Las combinaciones comunes se inicializan en backend al crear
la empresa. El botón + junto al selector abre «Nueva presentación» y solicita
únicamente Nombre; reutiliza tipos existentes por nombre o genera su código
técnico internamente. No cambia la preferencia global de la unidad. Selecciona
la respuesta confirmada, dejando el contenido pendiente de completar.
Requiere permisos de Compras y conserva conversiones históricas, stock y
documentos anteriores. El guardado es explícito, bloquea duplicados y conserva
el borrador si falla; el catálogo presenta skeleton, error y reintento.
La opción individual conserva el código `unit` y factor 1, pero muestra
`unitName` del catálogo remoto, no la etiqueta técnica «Unidad base».
Cuando otro tipo tiene el mismo nombre que la unidad de inventario, el selector
de registro conserva únicamente la opción individual y elige la preferencia
entre opciones visibles. La comparación ignora espacios, mayúsculas y tildes,
sin un catálogo fijo de envases. El + reutiliza esa opción si se ingresa su
nombre, sin crear otra combinación. Una conversión previa seleccionada con
contenido distinto de 1 permanece identificada por su contenido; no se alteran
documentos, existencias ni conversiones persistidas.

## Acceso directo al cobro

`/pos?orderId=...` representa una intención de cobrar desde Salón o Pedidos.
La pantalla se llama «Cobros» (la ruta y el módulo `pos` conservan su nombre
técnico). Su listado consulta `pos/orders?paymentStatus=unpaid` por defecto
(Sin pagos y Pago parcial); Ventas consulta `paymentStatus=paid` y es quien
devuelve pagos de pedidos saldados mediante la API pública
`@/modules/operations/pos` (`getPOSCollectionAccess`, `refundPayment`,
`RefundDialog`).
El POS inicia la misma consulta de saldo usada por la acción Cobrar de su tabla
y abre el formulario de pago, no el detalle de lectura. No depende de encontrar
el pedido en la página visible ni del filtro activo. Cambiar el `orderId` remonta
la composición para no conservar el pedido anterior.
Mientras consulta pedido y turno muestra skeleton; un fallo conserva reintento
explícito. El formulario requiere `cash.manage`, turno y pedido aún cobrable.
El saldo se consulta nuevamente al abrir, incluso si estaba en caché; recuperar
el foco no remonta el formulario ni borra el monto que el usuario está escribiendo.
Registrar utiliza `POST /v1/admin/payments`; backend vuelve a validar permisos,
empresa/local, estado, turno y saldo en transacción. El éxito cierra el formulario
e invalida POS, Caja, Salón, Pedidos, Ventas y Dashboard. No registra pagos al
abrir el enlace y no incorpora una acción de cobro al modal de consulta.

## Punto de entrega en el detalle de pedido

`Order` incluye `deliveryLatitude`, `deliveryLongitude` y `deliveryDistanceKm`
(números o `null`). Concierge los guarda con el pin de WhatsApp y el alta
manual de delivery con el punto marcado en el mapa (`deliveryLatitude` y
`deliveryLongitude` en `POST /v1/admin/orders`; la distancia la calcula el
servidor). Recojo y Salón los devuelven en `null`. Las ediciones no los cambian. `deliveryPoint` (dominio de
`operations/orders`) es el único que interpreta esos campos y construye la URL
`https://www.google.com/maps/search/?api=1&query=lat,lng`: devuelve `null` si el
canal no es `delivery` o falta alguna coordenada válida, y descarta distancias
no numéricas. La presentación solo formatea la distancia con
`formatRegionalNumber`. Usa la misma consulta del detalle (`getOrder`), sin
peticiones adicionales ni geocodificación en el cliente.

## Panel de reparto y enlace del motorizado

El panel de la central (`/delivery-panel`) exige la sesión de un operador: el
proxy `/api/delivery-partner/[...path]` reenvía solo la cookie HttpOnly
`fudia_partner_session` a `/v1/delivery-partner/*`, devuelve la que fije el
backend y la borra ante un 401 o al cerrar sesión, aunque el backend no responda. La página redirige a `/delivery-panel/login` sin
cookie; en el cliente, un 401 o `password_change_required` (`needsLogin`,
`needsPasswordChange`) también vuelven al acceso. `/delivery-panel/{token}`
(enlaces antiguos) redirige al acceso. El enlace del motorizado
`delivery-courier/{courierToken}` sigue público por `/api/public/[...path]`.
TanStack Query consulta cada 5 s (30 s tras un error), también en segundo plano:
`["delivery-partner-panel"]`, `["delivery-partner-session"]` (nombre del
operador, 60 s; el acceso la deja en caché con la respuesta del login o del
cambio de contraseña, sin pedirla otra vez, y tras un error se reintenta cada
30 s) y `["delivery-courier",token]`. Los contadores (oferta, espera de recogida y
entrega fallida) avanzan cada segundo con `useTickingClock` dentro de la tarjeta o
fila que los muestra, solo mientras tienen un contador activo: el panel completo
ya no se vuelve a renderizar cada segundo.
Cada acción invalida su consulta y muestra el resultado (`outcome`) con
`outcomeFeedback`; el navegador no calcula estados ni montos.

La central filtra y prioriza esa misma respuesta con `domain/dispatch.ts`, sin
peticiones ni contratos nuevos. «Por asignar» incluye solo pedidos aceptados sin
nombre. La carga agrupa nombres normalizados de repartos activos; no representa
disponibilidad ni un padrón de trabajadores. Asignar desde esa vista conserva
el detalle abierto para copiar el nuevo enlace aunque salga del filtro. Ante
un error de actualización se conservan los datos y se bloquean las escrituras.

Las vistas revelan los pedidos en cliente de 10 en 10 después de aplicar búsqueda
y filtros. Un `IntersectionObserver` agrega el siguiente lote al alcanzar el
final visible, respetando el scroll interno de escritorio y el de página en
móvil. El botón «Cargar más pedidos» ofrece continuidad manual y por teclado.
Los contadores y la carga del responsable usan la respuesta completa. Los sondeos
conservan cantidad revelada y selección; cambiar filtros reinicia en 10. No se
simula una petición al revelar datos ya recibidos. El API actual no ofrece cursor:
conserva activos y entregados de las últimas 12 horas (`PanelHistoryHours`). Para
escalar la consulta e historial amplio se requiere ampliar OpenAPI y backend con
cursor, filtros y agregados independientes de la ventana visible.

- Central: `POST …/actions` con `accept`, `reject`, `release` (solo por recoger),
  `keep` (mantener ante la alerta de recogida) y `fail` (entrega fallida tras el
  reporte del motorizado). `POST …/courier` `{token,courierName}` registra o
  cambia al motorizado y devuelve `courierToken`; el enlace se arma con
  `courierLink(origin,courierToken)` como `/delivery-courier/{token}`. El mismo nombre
  conserva el enlace; otro nombre lo reemplaza y el backend invalida el anterior.
- Alertas: `pickupAlert` (`warning` 3 min, `urgent` 5 min, `critical` 8 min),
  `pickupWaitSeconds` y `pickupAlertAcknowledged` vienen del backend;
  `needsPickupDecision` muestra Mantener / Cambiar motorizado / Liberar. Los
  relojes (oferta, espera y `failAvailableInSeconds`) solo descuentan el tiempo
  transcurrido desde la última consulta. Nada se libera ni se registra por tiempo.
- `failedDeliveries` lista las entregas fallidas registradas con sus montos y
  notas de validación tal como las devuelve el backend.
- Motorizado: `GET` devuelve `{partnerName,courierName,order}`; `POST …/actions`
  con `pickup`, `deliver` y `not_delivered`; `POST …/location` con la ubicación
  (cuerpo, nunca la URL) solo con el reparto en curso. Un 404 muestra el mensaje
  del backend (enlace reasignado, liberado, entregado o cancelado).
- «Recogí el pedido» requiere `paymentConfirmed` y cocina terminada. Sin
  confirmación, incluso si falta el campo, permanece deshabilitado con el motivo
  visible; `payment_pending` explica que debe esperar a caja. El backend valida
  nuevamente bajo bloqueo.
- Pedidos: `canCancelOrder` oculta «Cancelar» en un delivery con central asignada
  (`deliveryDispatchStatus` `accepted`, `picked_up` o `delivered`) que está listo
  o en camino; el backend responde `delivery_in_progress_not_cancellable`. Sin
  despacho o con delivery propio la regla no cambia.
- Campana: `delivery_pickup_delayed` (8 min sin recogida) abre Pedidos, tono
  `peach` e ícono `clock`.

## Tarifas de envío

`DeliveryFeesPanel` (módulo `delivery`, pestaña Delivery › Tarifas de envío) usa la
clave `["delivery-fees"]` con `GET concierge-settings/delivery-fees` y guarda
con `PUT concierge-settings/delivery-fees` (el proxy `/api/admin` reenvía `PUT`).
El borrador de filas es estado local; `deliveryFeeErrors` valida límites
(0.01–50 km, costo 0–999, dos decimales, sin límites repetidos) y
`deliveryFeesPayload` envía `{tiers:[{maxDistanceKm,fee}]}` numérico y ordenado.
El backend vuelve a validar y es la autoridad. Al guardar se reemplaza la caché
con la respuesta, sin otra consulta. Las tarifas aplican al local activo.

## Delivery Fudia

El catálogo remoto `["geographic-areas",parentCode]` pertenece a Locales y
Primer local. El distrito conserva su función de dirección. Centrales solo
registra nombre, teléfono y estado: no envía `coverageCodes`. Los antiguos
campos de cobertura del API son históricos y no restringen el reparto.
Todas las centrales activas reciben ofertas simultáneas; la primera aceptación
gana bajo bloqueo del backend. El orden solo organiza el listado. Las empresas
no crean, seleccionan ni modifican centrales. Desplegar el backend sin filtro
de distritos antes de este frontend.

`src/modules/delivery` publica tipos, API y componentes. Todo pasa por el proxy
`/api/admin` o `/api/platform`:

- **Delivery (`/delivery?tab=`):** `deliveryTabFrom` valida la pestaña contra
  las visibles y cae en `providers`. No tiene una lista duplicada de pedidos.
  `?tab=orders` redirige en servidor a Pedidos; cambiar entre las demás pestañas
  actualiza la URL con `history.replaceState`.
  - Proveedores: clave `["delivery-providers"]` con `GET delivery/partners`
    (`{enabled,districtName,items:[{name,whatsappPhone}]}`), solo lectura. Si la
    sesión es de Plataforma se muestra `DeliveryPartnersAdmin`.
  - Entregas fallidas: clave `["delivery-failed"]` con `GET delivery/failed-deliveries`.
- **Centrales (Plataforma, en Delivery › Proveedores):**
  - Clave `["platform-delivery-partners"]`, con `GET|POST /platform/delivery-partners`
    y `PATCH /platform/delivery-partners/{id}`.
  - Desactivar reenvía nombre y WhatsApp con `active:false`.
  - Subir o bajar envía el orden completo a `PUT /platform/delivery-partners/order`
    y reemplaza la caché con la respuesta. Ante un error, recarga la lista.
- **Empresa (Plataforma):** clave `["platform-organization-delivery",organizationId]`
  (`GET|PUT /platform/organizations/{id}/delivery`). El proxy de plataforma
  reenvía `PUT`.
- **Local:** solo usa `["delivery-link",locationId]`
  (`GET|POST locations/{id}/delivery-link`). Generar o regenerar el enlace
  reemplaza la caché con la respuesta.
- **Pedido › Reparto:** `["delivery-dispatch",order.id,order.status]` consulta
  `GET orders/{id}/delivery-dispatch`, cada 10 s mientras el pedido no esté
  cerrado. Incluye central, teléfono, motorizado, fechas, alerta e incidencia
  sin descargar la lista del local. Solo `404 delivery_dispatch_not_found`
  representa ausencia de despacho; el resto conserva el error explícito.
  - Reintentar y Usar delivery propio están en `OrderDispatchActions`, dentro
    de la fila de Pedidos, y llaman los POST existentes `retry|own`. El detalle
    solo presenta seguimiento. La fila usa `deliveryDispatchStatus` del listado
    sin pedir el despacho por cada registro; el listado se refresca cada 10 s.
    Tras la operación se invalidan despacho, detalle, lista,
    POS, ventas, dashboard y avisos. Esto actualiza también el total y saldo al
    devolver el costo de envío al restaurante.
  - Se bloquean dobles acciones durante la mutación y acciones en pedidos
    cerrados, con entrega fallida o sin `orders.manage`. El backend conserva
    la autorización y todas las transiciones, incluidos conflictos por pagos.

El avance de Delivery Fudia pertenece al motorizado: Caja no ofrece «En camino»
ni «Entregar», tampoco en «Sin central». Primero debe reintentar centrales o
confirmar el cambio a delivery propio; el backend rechaza el avance manual
mientras el despacho siga en Delivery Fudia.

## Tope mensual de Fudia Concierge (solo Plataforma)

`platform-api` es el único consumidor de `/v1/platform/concierge-usage*` y del
campo `maxConciergeConversations` (`PlatformSubscriptionPlan`); el plan de empresa
y `getCurrentOrganizationSubscription` no lo reciben. Claves de caché:
`["platform-concierge-settings"]` (modo y porcentaje global),
`["platform-concierge-usage",ids]` (uso del mes de la página visible de Empresas,
consultado después de cargarla, máximo 100 ids; `summary` cuenta toda la cartera) y
`["platform-concierge-usage-detail",id]` (detalle, tokens, historial). Guardar la
configuración reemplaza su caché e invalida uso y detalles; guardar la excepción
de tope (`PATCH {limitSource,maxConversations}`, número solo con tope propio)
reemplaza el detalle e invalida el listado; guardar un plan invalida ambos. Los
topes vienen siempre del API y los números/meses usan `regional-format`.


## Caja de cobro del administrador

POS usa las capacidades del API `pos/access`, no infiere la elección a partir de
`cash.manage`. Para administrador, una caja abierta se selecciona automáticamente;
varias muestran un selector en el formulario de cobro y ninguna impide registrar.
La elección se limita a empresa/local/usuario de la consulta, se descarta al cerrar
el formulario y se revalida si cambian las opciones. Solo el cobro administrativo
envía `shiftId`. Cajeros y colaboradores conservan el turno asignado. El cobro
identifica al actor y no cambia al abridor. Devoluciones usan `assignedShift`,
sin heredar la elección administrativa de caja.

## Campana de avisos del personal

`modules/notifications` es un slice independiente: `NotificationBell` hace su
propia consulta (`useNotices`) y el shell solo lo monta en la barra superior,
antes del selector de local, y le pasa `canOpen` para saber qué pantallas puede
abrir el usuario. La consulta `GET /api/admin/notices` (BFF → `/v1/admin/notices`)
usa la clave `["staff-notices", empresa, local, usuario]`, se refresca cada 15 s,
al recuperar foco si la respuesta tiene más de 10 s y siempre al abrir la campana, y solo se activa con `tables.read` o
`cash.manage`; sin esos permisos la campana no se muestra. El backend decide qué
avisos ve cada rol y devuelve los textos redactados: el panel no arma mensajes
ni filtra por rol.

El backend solo devuelve avisos pendientes: cuando el pedido se atiende (comanda
lista entregada, cuenta pagada, pedido nuevo cuya comanda ya está lista o ya fue
entregado, o pedido cancelado) el aviso desaparece de la lista y del contador en
la siguiente consulta (las mutaciones de Salón, Pedidos, Cocina y POS invalidan
`["staff-notices"]` para actualizarla al instante). Abrir un aviso no lo quita:
si no se atendió, sigue visible como leído.

El aviso `pay_detected` trae `payNotificationId`. Con Caja permitida, la campana
no navega: monta `PayNotificationOpener` (de `modules/payment-devices`), que lee
el pago con `GET /api/admin/pay-notifications/{id}` (clave
`["pay-notification", id]`, sin caché) y abre «Asociar pago a pedido»; al asociar
invalida `["staff-notices"]`. Sin id, el aviso lleva a `/cash?tab=pagos`.

«Cuenta anterior sin cerrar» (`stale_account`, icono `clock`, tono `peach`) llega
cuando alguien escanea el QR de una mesa cuya cuenta lleva más de 3 h sin pedidos;
abre la mesa en Salón (o el cobro en POS) y sale de la lista al cobrarla,
cancelarla o agregar productos.

Abrir un aviso lo marca como leído al instante (actualización optimista con
reversión si falla) y navega a la primera pantalla permitida para su tipo:
pedidos y llamados de mesa → Salón; ventas por mostrador, delivery y recojo →
Pedidos (su detalle tiene «Enviar comanda», porque el pedido del bot espera a caja)
o Cobros; cuenta pedida → Cobros o Salón; comanda lista → Salón si
tiene mesa y Pedidos si no. Si el aviso trae `orderId`, Salón y Pedidos abren
directamente el detalle de ese pedido (`?orderId=`), listo para actuar (por
ejemplo, el mozo pulsa «Confirmar entrega»); en Cobros solo la cuenta
pedida abre el cobro del pedido. `useOrderDeepLink` consume el parámetro al
abrir el detalle y devuelve la URL a la pantalla, así el mismo aviso puede
reabrirlo. Las acciones del detalle siguen sujetas a los permisos de siempre.
Los avisos que aparecen entre dos lecturas del feed (`freshNotices`) se
anuncian por `aria-live` y, con la campana cerrada, se muestran como aviso
emergente (portal a `body`, porque la topbar atraparía el `position:fixed`).
Tocarlo hace lo mismo que elegirlo en la lista: lo marca como leído y abre su detalle.
«Marcar como leídos» aplica a todos los visibles del
usuario. La lista tiene skeleton, error con Reintentar y estado vacío; un aviso
nuevo se anuncia por `aria-live` sin robar el foco.
