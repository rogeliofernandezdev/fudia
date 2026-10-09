# Estado e integración

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
posterior a los límites de procesamiento y respuesta del backend.

Las transiciones de Cocina invalidan comandas, pedidos y el detalle afectado.
La respuesta confirmada se refleja en el panel correspondiente sin modal de
éxito; durante el envío se bloquea la acción y un fallo conserva el aviso de error.

El wizard de Menús y combos valida cada paso antes de continuar y todos los pasos
antes de guardar. Explica el primer error y enfoca el campo afectado. Retroceder
no borra los datos ni guarda registros parciales. X/Cancelar solicitan confirmación
cuando hay cambios sin guardar; la recarga utiliza el aviso nativo del navegador.
No se repite «Paso X de Y» junto a la navegación de pasos.

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

Las búsquedas de listados no disparan una petición por pulsación. Deben reutilizar
`useDebouncedValue` con una ventana aproximada de 300 ms y mantener el texto
visible separado del valor consultado.

TanStack Query conserva la caché por clave; las mutaciones invalidan únicamente
las familias afectadas. Los diálogos pesados que no forman parte del primer render
se cargan con `next/dynamic` cuando la acción realmente se abre.

El prefetch de datos se incorpora solo en rutas frecuentes y cuando la medición
demuestre que reduce latencia sin duplicar tráfico.

## Cantidades en Menús y combos

El selector usa `availableQuantity` del listado de Productos, no cupos
predeterminados del catálogo. Selección manual y plantilla Menú del día comparten
`createComboOption`: «Reservar» parte del saldo real del local (porciones aún
disponibles o stock físico), conserva cero cuando está agotado y deja vacío con
«Sin límite» cuando el API devuelve null para Sin control. Cada opción se vende
en unidades completas; un saldo físico fraccionario limita las reservas a su
parte entera, sin modificar el inventario.

La validación compartida del wizard y el campo rechazan negativos, fracciones
y cantidades superiores al saldo. El usuario puede reducir la reserva.
La caché incluye empresa/local; se consulta al abrir el wizard y se refresca
cada diez segundos mientras permanece abierto. Un refresco no pisa las
cantidades elegidas. Editar un menú conserva sus reservas persistidas; una
cantidad que exceda el saldo actualizado muestra validación, no un reemplazo
silencioso. Guardar conserva el contrato existente `quota` y no realiza un
movimiento de stock.

## Estado de productos

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

«Nuevo pedido» abre un formulario breve de datos, con Delivery inicialmente seleccionado.
Los tipos seleccionables (Delivery, Recojo y Mostrador) toman sus etiquetas de
`channelOptions` del API. Salón conserva su toma de comanda por mesa; WhatsApp
conserva el ingreso por Concierge y no se ofrece como origen manual.
Delivery exige cliente, teléfono válido y dirección; referencia, notas y costo
de envío completan la entrega. «Continuar» valida estos datos y abre `ComandaView`,
la misma toma de pedidos utilizada por Salón: carta, cantidades, elecciones,
notas y resumen. Delivery muestra cliente/dirección en lugar de mesa e incluye
el envío en el total. Recojo y Mostrador tampoco solicitan mesa. «Editar datos»
vuelve al formulario conservando productos, elecciones y notas. Avanzar o volver
no realiza peticiones de creación ni genera pedidos parciales. El catálogo
operativo es paginado y presenta la disponibilidad real del local.

La acción explícita «Registrar y enviar a cocina» hace POST al endpoint existente
`/v1/admin/orders` con `sendToKitchen=true`; no guarda al pulsar Enter desde un
campo. El servidor deriva empresa/local, recalcula precios y valida disponibilidad.
La petición bloquea campos, cierre y repetición; el aviso de fallo cierra el modal
mediante el cierre directo de respuesta, sin abrir una confirmación de descarte.
X/Cancelar confirma el descarte si hubo cambios. El éxito cierra el formulario,
selecciona el canal creado, limpia filtros e invalida Pedidos, Cocina, POS,
Dashboard y disponibilidad. No se registran pagos desde este formulario: el cobro
sigue en POS. Delivery continúa con Listo → En camino → Entregado.

## Responsable de mesa y autoría del cobro

Al guardar la comanda de una mesa, la API asigna al usuario autenticado como mozo.
Salón y Pedidos muestran «Mozo» y, cuando existen pagos, «Cobrado por», con los
nombres reales del servidor. Sus detalles reutilizan una franja de datos de lectura;
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

Productos obtiene del backend Cocina, Barra y Entrega directa. Salón y Pedidos
muestran destino y estado por producto solo como lectura, sin botones por fila.
El pie presenta un único «Confirmar entrega» para todo el pedido cuando todos
los productos pendientes están Listos. Usa `PATCH orders/{id}/status` con
`status=entregado`: el backend confirma todas las líneas listas en una transacción,
sin modificar las ya entregadas. No incorpora una alternativa de entrega individual.
Cocina/Barra preparan de forma independiente; Entrega directa nunca ingresa a KDS.

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

Salón y Pedidos reutilizan `canCancelOrder`: una Entrega directa lista aún sin
servir puede cancelarse si no hay pagos ni preparación iniciada o entrega de
otros productos. El estado agregado Listo no significa que se haya preparado.
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
