# Pruebas

`comanda-catalog-layout.test.mjs` protege categorías del API, acceso persistente a
Menús y combos, selección única, recuperación de categorías, precios y bloqueo
de productos no disponibles. Verifica el fallback de fotos, controles de cantidad,
desplazamiento nativo sin barra visible y acceso a la lista completa. Verifica
acentos de categoría estables por ID (índigo, celeste, violeta y petróleo),
marca de selección independiente del color y contraste AA en cada familia,
cápsulas de cantidad compartidas y fotos de 64 px con fallback del sistema.
Tras build, `comanda-preview.mjs` renderiza la
comanda real con datos ilustrativos y comprueba 390 px, escritorio, tablet,
altura reducida, categorías extensas, menús, selección, skeleton, error y vacío.
La última categoría sigue accesible mediante scroll horizontal y la lista completa
incluye todas las categorías y menús; su acceso no se desplaza con la tira.
La barra inferior abre la revisión, sin guardar ni enviar. Las pruebas del borrador
comprueban que el control circular para plegar la hoja conserva productos y
notas, vuelve a la carta sin guardar ni cerrar la comanda y presenta un icono
chevron de 24 px, fondo índigo tenue sin borde y un área táctil de 44 px. También
comprueban disminuir/quitar sin afectar combos ni notas de otras líneas.
Protegen «Guardar comanda» con `sendToKitchen=false`, «Enviar a cocina» con `true`,
bloqueo de clics repetidos, progreso en la acción elegida y aviso «sin enviar a cocina».
El botón secundario tiene fondo `primary-200`, borde `primary-600` y texto
`primary-700`; el hover usa `primary-300`/`primary-800` para conservar contraste AA.
Comprueba los iconos de portapapeles/envío, radios completos en las acciones
móviles y cabecera blanca sobre índigo sin quedar cubierta por el scrim abierto.
Incluye resumen móvil abierto, notas, opciones de combos, nombres largos y
ausencia de imagen; comprueba cantidades de 44 px, foco sin recortar y pie visible.
La foto del catálogo acompaña al borrador sin formar parte del payload del pedido.
No registra pedidos ni consume el API. La emulación
comprueba geometría; tacto, teclado y chrome del navegador requieren teléfono real.

`modal-standard.test.mjs` protege la cabecera solo con icono, título y X,
contexto fuera del header, estado opcional real y acciones contiguas a la
derecha con 10 px. Audita que los Dialog de contenido no dupliquen la cabecera
ni el footer del formulario. Confirmaciones y respuestas quedan exceptuadas.
`shared-ui-preview.mjs` comprueba esa geometría en escritorio, 390 px y altura
reducida, incluyendo el pie visible al desplazar el cuerpo.

`sales-detail.test.mjs` protege el orden mesa (o cliente), canal, atendido por y
fecha del pedido, con dos filas alineadas en escritorio; en móvil mesa/canal
comparten fila y encargado/fecha ocupan filas completas con etiqueta junto al valor;
el skeleton reproduce los cuatro datos y otros canales no inventan un mozo.
Tras el build, `node tests/helpers/sale-detail-preview.mjs .next/sale-detail-preview`
renderiza el modal real con datos ilustrativos, sin API ni escrituras.
Comprueba escritorio, 390 px, altura reducida, nombres largos, Delivery y carga:
sin recortes ni desbordes, header sin información adicional y resumen fijo visible.
Ventas también comprueba pagos reales, devoluciones, pagos vacíos y múltiples
pagos; listas sin tarjetas, resumen fuera del cuerpo y ticket visible al desplazarlo.
El test de estructura exige `ModalWorkspace`, `ModalItemList`, cantidades circulares,
separador de pagos y estado único en el resumen, sin un pie de acciones vacío.
Con un tercer argumento `pos`, el mismo preview revisa el detalle de cobro:
resumen inferior fijo, pagos parciales y acciones ocultas en solo lectura.
Incluye anchos de 600, 610, 620 y 630 px para evitar recortes al pasar entre
backdrop con márgenes y pantalla completa. En móvil valida mesa/canal juntos,
encargado/fecha a ancho completo y el resumen dentro de la pantalla.
Verifica además el canal alineado al extremo derecho, valores de encargado y
fecha en la misma columna y jerarquía real de tamaño y color entre etiquetas
y datos: etiquetas en seminegrita, valores regulares y títulos de sección en
negrita, tanto en móvil como en escritorio. La etiqueta contextual muestra
«Mesa», «Cliente» o «Venta» según el dato real, nunca la combinación ambigua.
`pos-payment-flow.test.mjs` protege esta organización y conserva los permisos
de devolución, turno requerido, cierre y separación del formulario de cobro.

`modal-typography.test.mjs` protege los roles compartidos de etiquetas 600,
valores 400 y secciones 700, la adopción en los detalles y asistentes, y la
ausencia de pesos invertidos en estilos locales. Los resúmenes monetarios y
estados conservan su énfasis. Protege también la ausencia de «Cliente / Mesa»
en modales; POS y Ventas prueban mesa, cliente y venta directa por separado.

`login-layout.test.mjs` protege los tres mensajes informativos del acceso en
una sola fila, icono/texto alineados y textos completos con tipografía auxiliar.
Protege además la misma estructura centrada en móvil y escritorio, cabecera
de marca destacada sin panel lateral ni títulos añadidos, campos neutros, geometría táctil, foco
visible, scroll en viewport bajo y texto «Ingresar», conservando carga,
validación y contexto posterior. Se revisa su geometría en escritorio y móvil
desde 390 px, incluidos errores y la sesión activa, en navegador aislado.
Comprueba que Recuperar contraseña muestra solo «Próximamente», sin consultas ni
envíos, conservando foco y cierre compartido. Su hover no añade fondo, sombra ni
subrayado; el icono de mostrar contraseña tampoco añade fondo ni sombra al pasar
el cursor. Ambos mantienen foco visible con teclado. No añade recuperación por correo.

`payment-ticket.test.mjs` verifica ancho real de 80 mm, PDF/impresión con la misma
composición, negro/blanco, nombres y tildes, cantidades, pagos divididos y saldo
parcial, continuación de tickets largos, escape de HTML y ausencia de datos
fiscales inventados. `pos-payment-flow.test.mjs` comprueba que la vista previa se
abre después del API y del aviso, nunca en error ni antes de enviar. Se revisa el
PDF renderizado y el diálogo de impresión aislado; el cierre A4 no se modifica.

`cash-report.test.mjs` comprueba el resumen completo, cantidades/precios/autores,
egresos y formato del snapshot. Genera un PDF real con logo y 80 productos para
verificar varias páginas, y prueba clic explícito, doble envío, espera del API,
caché por empresa/local/turno, apertura después del aviso global, reintento de
lectura sin repetir cierre y liberación del Blob. La revisión visual con Poppler
comprueba cabeceras, tildes, tablas y pie en páginas iniciales, intermedias y
finales; extracción de texto comprueba las 80 líneas y el último producto.
El cierre protege Enter en los inputs sin bloquear saltos de línea en la
observación ni activación del botón enfocado. Se verifica altura acotada,
scroll visible y pie persistente también en resoluciones bajas desde 390 px.

`availability-history.test.mjs` prueba motivo nuevo obligatorio, longitud y
trim, validación sin API, Enter sin guardado, una petición pendiente, bloqueo de
campos/cierres, cupo separado de estado, envío después del motivo y caché por
empresa/local/producto. Comprueba lectura sin escritura, carga/error/reintento,
vacío, paginación, hora local, históricos sin inventar valores y layout compartido
desde 390 px con scroll controlado de historial.

`combo-quantity.test.mjs` renderiza el wizard real para comprobar la precarga de
«Reservar» en la selección manual y la plantilla con `availableQuantity` del API,
incluidos agotado y Sin control. Cubre límites compartidos, stock fraccionario
en opciones de unidades completas, reserva reducida, edición sin reemplazo,
caché por empresa/local, refresco sin sobrescribir valores y transporte de
`quota`. Protege la eliminación de `defaultDailyQuota` en Menús y combos.

`purchase-response.test.mjs` utiliza las mutaciones reales de TanStack para
probar respuestas demoradas, clics repetidos, bloqueo compartido entre acciones,
ausencia de reintento automático y refresco pendiente tras éxito/error. El aviso
solo aparece después del API y no deja el detalle ni la confirmación abiertos.
`ux-refinement.test.mjs` verifica el cierre compartido de hijo a padre antes del
aviso, limpieza de registros desmontados y adopción de `onResponseClose` en todos
los modales de origen. Conserva foco, scroll y cierre explícito sin Escape/fondo.

`product-dialog.test.mjs` verifica cantidad inicial obligatoria y positiva solo
en altas con porciones, campo a ancho completo, único POST y ausencia de cantidad
en Sin control/PATCH. Conserva guardado explícito, bloqueo y reintento manual.

`table-batch.test.mjs` prueba una sola escritura, errores de resultado incierto
sin reintento automático, conservación del borrador, refresco de mesas y bloqueo
durante envío. También prueba el BFF con respuestas confirmadas/cortadas y la
propagación de correlación sin exponer cookies en logs.

`account-identity.test.mjs` protege los roles reales bajo el usuario, sus estados
remotos y el aislamiento de caché por usuario/empresa/local. Contexto verifica
que el selector conserve el local activo para empresa y plataforma; la empresa
se muestra por separado en la barra superior.

`plan-navigation.test.mjs` verifica Menús y combos, Inventario y Compras en Emprende, ocultación de
módulos no incluidos incluso con rol de acceso total, separación de acceso y
lectura, catálogo de Plataforma y recarga de roles/permisos al cambiar el plan.

Probar validaciones, permisos, filtros, errores y vacíos. Validar 390 px, escritorio
y accesibilidad. Puertas: lint, typecheck, tests y build.

`inventory-units.test.mjs` comprueba opciones del API sin lista fija, caché por
empresa, selección, skeleton/error/vacío, alta solo con permiso y selección de
la respuesta confirmada. Verifica envío explícito sin duplicar el POST, bloqueo
y conservación del formulario, sin anidar formularios ni guardar con Enter.

`purchase-item-dialog.test.mjs` usa el formulario y la validación reales para
comprobar «Crear y agregar» en productos e insumos. Sin unidad el botón permite
validar y muestra el dato faltante; categoría, precio y conversión inválidos no
envían. Cubre selección remota de unidad, corrección y reintento, una sola
petición ante clics rápidos, bloqueo durante envío y Enter sin creación implícita.

`purchase-categories.test.mjs` cubre el catálogo vacío con alta accesible,
`menu.manage`, categorías compatibles, selección del UUID devuelto, aislamiento
de caché por empresa, skeleton/reintento y el modal compartido de Catálogo y
Compras. No se duplica el formulario ni se genera un UUID en el cliente.

`pagination.test.mjs` protege la primitiva global en variantes completa, simple
y compacta: rango/total, límites de navegación, etiquetas e iconos, página activa
y colores compartidos. Impide volver a introducir paginadores antiguos, estilos
genéricos de pies de gestión o parches `!important` que sobrescriban la primitiva.

`tests/ux-refinement.test.mjs` verifica validación y conservación del borrador de
menús, cupos/recargos/fechas, catálogo paginado, transición directa de Cocina,
contraste del botón exitoso, geometría primaria única y uso de `Dialog` en todas
las ventanas. Incluye Tab/Shift+Tab, restauración de foco, ventanas superpuestas
y cierre exclusivamente explícito (autocierre solo en éxito).

El contrato de servicio separa preparación y cuenta. Probar destino remoto del producto, append sobre la misma cuenta Abierta sin líneas anteriores, botones por mozo, entrega completa antes de cierre/cobro, estados operativos sin modal exitoso y ajuste responsive desde 390 px. Los fixtures cobrables incluyen `billClosedAt`, estado Entregado y todos los productos entregados; Listo no equivale a Entregado. Rechazar cierre/cobro con Cocina, Barra o Entrega directa pendientes, incluso con estado agregado obsoleto; después del cierre no permitir consumo adicional ni con pago parcial.

Salón y Pedidos renderizan los controles reales para comprobar una única acción
«Confirmar entrega» en el pie, sin botones por producto. Un pedido mixto de Cocina,
Barra y Entrega directa debe tener todos sus productos pendientes Listos; un estado
agregado Listo no permite saltar productos en preparación. Probar mozo asignado,
lectura sin permisos, bloqueo durante envío, ronda adicional con entregas previas
y transición a «Cerrar cuenta» después de servir todo. La entrega envía una sola
petición de estado, refresca las vistas afectadas y conserva el modal de error,
sin modal exitoso.

`order-delivery.test.mjs` verifica que ambos detalles (Salón/Pedidos) compartan
la cancelación de Entrega directa sin servir, oculten la acción ante preparación,
entrega, pago o finalización, respeten al mozo asignado y la bloqueen mientras
se guarda. Las comandas históricas Listas no ofrecen cancelación.

`product-list.test.mjs` verifica siete columnas y paginación compartida, nombre
sin descripción, saldo cero, inventario con unidad y Sin control con «—».
Comprueba caché por local, refresco del saldo y skeleton sin línea secundaria.
Incluye acceso al menú sin gestión: no hay botones de creación/edición/estado
ni columna de acciones en Productos y Categorías, tampoco en vacíos o skeleton.
Con gestión conserva las acciones; al perderla retira los formularios abiertos.

`assigned-permissions.test.mjs` prueba la separación menú/acciones en la
sesión para Administrador, Cocinero, Mozo y roles personalizados, sin privilegios
por nombre. Cubre permisos independientes de Mesas y Zonas, vacíos y borradores
sin autorización. `combo-quantity.test.mjs` añade lectura sin altas/edición/estado
y retiro de wizard/confirmación al perder gestión, conservando consulta.

`inventory-settings-dialog.test.mjs` protege la estructura compartida del modal
de niveles (formulario, campos y footer), estilos responsive desde 390 px sin
CSS local, validación explícita, normalización de niveles, guardado por botón y
una sola petición pendiente. Verifica bloqueo de campos/cierres y conserva el
cierre global únicamente al comunicar la respuesta del API.

`inventory-list.test.mjs` verifica Mínimo / Reorden en una sola línea, incluidos
cero y fracciones, sin texto secundario ni CSS correctivo. Conserva nueve
columnas, paginación compartida y desplazamiento horizontal en móvil.

`location-plan-limit.test.mjs` verifica «Nuevo local» deshabilitado al alcanzar
la capacidad remota del plan, primer local, límite ilimitado, guardia del handler,
lectura/paginación/inactivos, carga/error/reintento y edición sin bloquear.
Protege el refresco tras mutaciones y la consulta compartida aislada por empresa
y usuario, sin límites hardcodeados ni ampliación de permisos.

`user-plan-limit.test.mjs` verifica el cupo de usuarios activos, alta y
reactivación bloqueadas, edición/desactivación permitidas y roles independientes.
Incluye primer usuario, cupo ilimitado, búsquedas/paginación, carga/error/reintento,
permisos, refresco y cambio de cupo con confirmación abierta. Usuarios y Locales
ejercitan la misma capacidad compartida.

`purchase-presentations.test.mjs` verifica opciones remotas sin catálogo fijo,
preferencia por empresa/unidad sin adivinar contenido, una sola presentación
sin controles redundantes y + junto al selector. Cubre skeleton/error/vacío,
lectura sin alta, etiquetas «Unidad de inventario» / «Presentación de compra»,
nombre remoto y contenido 1 para compras individuales, creación con Nombre
únicamente, sin duplicar Paquete u otra unidad homónima. Verifica preferencia
entre opciones visibles, reutilización desde + sin POST, conservación de
conversiones anteriores y reutilización de nombres
existentes, códigos automáticos sin colisiones, selección confirmada, guardado
explícito y protección frente a doble POST. Incluye contenidos fraccionarios,
validación de conversiones históricas y transporte persistente
sin stock al crear. `purchase-item-dialog.test.mjs` comprueba la integración
del selector compartido y conservación del borrador si falla el API.

`shared-ui-layout.test.mjs` protege la eliminación de pestañas antiguas,
selección suave y contador contrastado, navegación por teclado y foco itinerante,
márgenes comunes de modal, área táctil de cierre y geometría móvil sin recorte.
Protege la cabecera índigo `primary-700`, título/cierre blancos, padding vertical
de 8 px y altura mínima de 64 px. Los helpers comprueban contraste AA y permiten
que los encabezados largos crezcan sin recortar texto.
La cabecera llega al borde superior y ocupa todo el ancho del modal, sin franja
degradada ni elementos decorativos anteriores.

La comprobación visual aislada `tests/helpers/shared-ui-preview.mjs` renderiza las
primitivas React reales con sus CSS y Manrope del build, usando datos ilustrativos.
No consulta ni modifica datos de restaurantes. Con Playwright disponible y, si
hace falta, `FOODS_PLAYWRIGHT_PATH`/`FOODS_BROWSER_PATH` apuntando a sus instalaciones
locales, ejecutar `node tests/helpers/shared-ui-preview.mjs .next/shared-ui-preview`.
Guarda capturas y geometría para escritorio, tablet, 390 px y viewport bajo;
comprueba desborde, cierre táctil, márgenes y pie visible después del scroll.
El navegador y el servidor temporal se cierran al terminar. No sustituye la prueba
con teléfono físico de teclado, áreas seguras y comportamiento de toque.

`modal-redesign.test.mjs` renderiza el detalle real de Pedidos/Salón mediante
`tests/helpers/order-detail-fixture.mjs`, con TanStack Query y un contexto de
sesión ilustrativo. Protege `ModalHeader` y `ModalWorkspace` compartidos, estados
de preparación/cuenta separados, resumen fuera del cuerpo desplazable y acciones
según mozo, entrega completa y permiso de cobro. Incluye carga, error recuperable,
productos vacíos y cuenta pagada; no consulta el API ni permite escrituras reales.
También protege `ModalItemList`: cantidades circulares con tokens azules existentes,
producto/importe alineados sin encabezados de tabla ni borde exterior,
notas fuera del listado sobre fondo suave y resumen inferior con separador fino. «Pendiente» usa
ámbar tintado; «Cuenta abierta» usa celeste tenue con icono y texto oscuros, sin borde, sin distinguir
artificialmente productos que comparten el mismo estado.
Subtotal, Pagado y Saldo pendiente se verifican en la misma fila a 390 px.
El fondo completo del resumen usa `cloud-50`; el estado abierto destaca
con fondo `sky-100` y texto `sky-800`, distinto del índigo de Confirmado. Pagada conserva verde y Finalizada usa gris neutro; los cuatro
estados de cuenta tienen tonos distintos. No se repite el conteo agregado de unidades en el encabezado.
Ejecutar desde `foods-admin-web`: `node --test tests/modal-redesign.test.mjs`.
`tests/operational-status-standard.test.mjs` protege los pares exactos de tokens,
su contraste mínimo 4.5:1 y su correspondencia con la
[matriz obligatoria](02_DESIGN_SYSTEM.md#colores-de-operación-pedido-preparación-y-cuenta).

Después de `npm run build`, ejecutar
`node tests/helpers/order-modal-preview.mjs .next/order-modal-preview` para
generar capturas y `geometry.json` del componente real con datos ilustrativos,
Manrope del build y CSS fuente de los mismos componentes. Para comprobar también
interacciones con los estilos compilados, definir `FOODS_PREVIEW_COMPILED_CSS=1`:
ambos helpers cargarán conjuntamente los CSS del build. Es una comprobación aislada
de composición, no una captura de una ruta autenticada ni garantía del orden de navegación.
Usa Playwright y las mismas variables opcionales de ruta descritas arriba.
Comprueba escritorio 1440×900, ventanas 1192×712 y 967×835, móvil 390×844,
viewport bajo 390×480, tablet táctil 768×1024, Delivery en escritorio/móvil,
skeleton y error. Mide el eje central de título/X, la primera línea de
etiqueta/nombre del mozo y el resumen inferior al mismo ancho del cuerpo. Exige que
el mozo preceda a la fecha en el DOM y comprueba los siete estados del pedido,
con contraste de texto de las píldoras de al menos 4.5:1. Verifica ausencia de desborde,
panel dentro del viewport, cuenta y acciones visibles, cierre táctil de 44 px y
scroll de 20 productos conservando la posición del resumen. El formulario global
se revisa con `shared-ui-preview.mjs` en tamaños `md` y `lg`, incluido pie visible
tras recorrer campos largos. Ambos helpers cierran navegador y servidor temporal
al terminar. Estas comprobaciones aisladas no prueban la aplicación con sesión
autenticada ni sustituyen una revisión en teléfono físico.
