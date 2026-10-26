# Pruebas

## Visor PDF compartido

`pdf-viewer.test.mjs` verifica versión alineada del visor oficial completo,
motor/worker locales, protocolo de transferencia, impresión de PDF.js, fallos,
límites de canvas y liberación de listeners/URLs. Los tests de ticket y caja
conservan el formato y contenido de sus generadores.
El harness `node tests/helpers/pdf-viewer-preview.mjs` permite comprobar el
componente real sin sesión, pagos ni escrituras de negocio: ticket, caja,
80 páginas y archivo dañado. Ver [matriz y límites](09_PDF_VIEWER.md).
La impresión Android/iPhone requiere pruebas en dispositivos físicos con sus
servicios e impresoras; cambiar el viewport no valida ese soporte.

## Subida directa de imágenes

image-upload.test.mjs verifica validación antes de autorización, bytes solo a
Storage sin cookies, file al final, rechazo de host ajeno y de archivos >5 MiB,
confirmación antes de POST/PATCH y guardado PROCESSING sin esperar READY. También
reutilización tras fallo del producto y exclusión de metadatos/URLs temporales
de los DTO de escritura. ProductDialog conserva su pie, permisos y validación.
Validar campo de imagen a 390 px/escritorio, PROCESSING/FAILED, selección inválida
y reintento. CORS/IAM/eventos y límite Storage necesitan prueba real en GCP;
los mocks locales no certifican esa configuración.

`uploads-route.test.mjs` protege GET/HEAD de fotos en el mismo origen, bytes y MIME
permitidos, UUID/nombre válido, timeouts/errores y ausencia de cookies, redirects
o contenido HTML del upstream. Usa el handler real con fetch aislado; no acredita
conectividad GCP ni sustituye subir/ver una foto tras cambiar revisión.

`feedback-success.test.mjs` protege el círculo verde con check blanco, entrada
circular desde escala grande hasta normal con rebote leve, en una capa verde
independiente del check, botón secundario neutro, separación de la X,
fundido con movimiento reducido y conservación de error/info,
autocierre y callback. `tests/helpers/feedback-preview.mjs` renderiza el componente
real con datos ilustrativos, comprueba contraste, centro del icono y geometría
desde 390 px, y muestrea la escala durante la animación sin ejecutar el API.

`button-actions.test.mjs` protege las intenciones compartidas: salir sin guardar
neutro, anular un registro rojo tenue y Continuar primario con flecha al final.
Comprueba que la variante no depende de la etiqueta y conserva envío de formularios,
accesibilidad y bloqueo durante una petición.

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
móviles y cabecera blanca sobre marino sin quedar cubierta por el scrim abierto.
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
`product-dialog.test.mjs` y `shared-ui-layout.test.mjs` protegen el uso de
`ModalForm`/`ModalBody` en Producto y la X blanca sin atenuación al guardar.
Tras build, `product-modal-preview.mjs` renderiza el modal real (solo sustituye
el interior del editor de texto) con datos ilustrativos: normal/guardando,
escritorio, 390 px y altura reducida. Comprueba los márgenes 24/16 px, separación
de 10 px, pie fijo sin recortes, acciones móviles iguales y cierre bloqueado
sin cambios de color/opacidad. No consulta el API ni guarda productos.

`sales-detail.test.mjs` protege el orden mesa (o cliente), canal, atendido por y
fecha del pedido, en una franja de una fila en escritorio (`grid-auto-flow: column`);
cada sección abre con su título subrayado (2 px `primary-200`) y no repite la
línea superior de los productos; en móvil los cuatro
datos usan filas compactas etiqueta/valor con la retícula compartida `modal-record-data`;
el skeleton reproduce los cuatro datos y otros canales no inventan un mozo.
Tras el build, `node tests/helpers/sale-detail-preview.mjs .next/sale-detail-preview`
renderiza el modal real con datos ilustrativos, sin API ni escrituras.
Comprueba escritorio, 390 px, altura reducida, nombres largos, Delivery y carga:
sin recortes ni desbordes, header sin información adicional y resumen fijo visible.
Ventas también comprueba pagos reales, devoluciones, pagos vacíos y múltiples
pagos; listas sin tarjetas, resumen fuera del cuerpo y ticket visible al desplazarlo.
El test de estructura exige `ModalWorkspace`, `ModalItemList`, cantidades circulares,
separador de pagos y estado único en el resumen, sin un pie de acciones vacío.
Protege «Ver resumen de pago» como `secondary` con icono `eye`, sin mutaciones.
`shared-ui-layout.test.mjs` protege el radio móvil compartido de los botones
de texto y conserva el cuadro de la X. El preview comprueba el radio real,
área táctil y ausencia de recortes en la acción inferior.
Con un tercer argumento `pos`, el mismo preview revisa el detalle de cobro:
resumen inferior fijo, pagos parciales y acciones ocultas en solo lectura.
Incluye anchos de 600, 610, 620 y 630 px para evitar recortes al pasar entre
backdrop con márgenes y pantalla completa. En móvil valida las cuatro filas
alineadas, columna común de valores, ajuste de nombres largos y resumen dentro de la pantalla.
Verifica además que las etiquetas se alineen a la primera línea de su valor,
que el contexto breve no exceda 140 px y la jerarquía real de tamaño y color entre etiquetas
y datos: etiquetas en seminegrita, valores regulares y títulos de sección en
negrita, tanto en móvil como en escritorio. La etiqueta contextual muestra
«Mesa», «Cliente» o «Venta» según el dato real, nunca la combinación ambigua.
`pos-payment-flow.test.mjs` protege esta organización y conserva los permisos
de devolución, turno requerido, cierre y separación del formulario de cobro.

`modal-typography.test.mjs` protege los roles compartidos de etiquetas 12 px / 600,
valores 14 px / 400 y secciones 16 px / 700, el título de sección compartido
`.crud-modal .modal-section-title` (mayúsculas, `ink-800`) y que pedido, venta
y cobro no redefinan la tipografía de sus `h3`, la adopción en los detalles y asistentes, y la
ausencia de pesos invertidos en estilos locales.
La excepción `modal-inline-metadata` comprueba etiqueta/valor a 14 px y mismo
interlineado, manteniendo 600/400 y fecha al extremo derecho sin recortes.
Los resúmenes monetarios y estados conservan su énfasis. Protege también la ausencia de «Cliente / Mesa»
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
desde 390 px con scroll controlado de historial. Comprueba cinco columnas,
separación entre cambio y cantidad, cupo anterior/nuevo, vendidas, ceros reales,
nulos sin valores inventados y fecha/hora sin cortes. El helper
`tests/helpers/availability-history-preview.mjs <directorio>` renderiza el modal
real con datos ilustrativos y estados normal, antiguo, carga, error, vacío y
contenido largo, a 390, 768 y 1440 px.

El mismo archivo de pruebas protege el listado de Disponibilidad: cinco
columnas agrupadas, iconos semánticos, Guardar solo después de editar, mínimo
de cupo con error asociado al campo y separación entre cupo y override manual.
Comprueba Sin control sin cantidades ficticias, inventario con unidad y «Sin
dato», restricciones de acciones derivadas, bloqueo durante envío y
consulta de historial sin permiso de escritura.

Después de `npm run build`, ejecutar desde `foods-admin-web`
`node tests/helpers/availability-page-preview.mjs .next/availability-page-preview`.
El helper renderiza el componente real y las primitivas compartidas con Plus Jakarta Sans
y CSS compilado del build. Sustituye las consultas por datos ilustrativos
aislados; no consulta ni modifica datos de restaurantes. Usa Playwright y las
variables opcionales `FOODS_PLAYWRIGHT_PATH`/`FOODS_BROWSER_PATH` para las rutas
locales. Comprueba 28 combinaciones: 390, 768, 1440 y 1546 px con estados
normal, editado, nombres/cantidades largos, solo lectura, carga, error y vacío.
Verifica ausencia de desborde en página y contenido, cinco columnas, respuesta
estrecha, área de controles de al menos 44 px, texto táctil del selector,
panel sin sombra y tono ámbar de Agotar en la cascada compilada. Guarda capturas
de normal en los cuatro anchos y de los siete estados a 390 px. Cierra navegador
y servidor temporal al terminar. Es verificación aislada de composición;
no sustituye una ruta autenticada ni la revisión de teclado, toque y áreas
seguras en un teléfono físico, que no se ha realizado en esta iteración.

`combo-quantity.test.mjs` renderiza la hoja real (`ComboEditor`) para comprobar la precarga de
«Reservar» en la selección manual y la plantilla con `availableQuantity` del API,
incluidos agotado y Sin control. Cubre límites compartidos, stock fraccionario
en opciones de unidades completas, reserva reducida, edición sin reemplazo,
caché por empresa/local, refresco sin sobrescribir valores y transporte de
`quota`. Protege la eliminación de `defaultDailyQuota` en Menús y combos. También
cubre Reservar visible en cada línea, el error junto al campo al guardar, la coma
decimal, la plantilla como primer paso, la confirmación al quitar una parte con
opciones y «Cuándo se vende» (solo Siempre y Elegir días; «Elegir días» sin días
bloqueado), las fechas convertidas con la zona del local (Lima UTC-5, Madrid con
cambio de horario) y «Fuera de fecha» en el listado. El buscador «Agregar opción» se prueba con
`buildOptionGroups`: secciones por categoría con la de la parte primero, «Sin
categoría» al final, sin repetir lo agregado y sin secciones vacías.

`combo-sale-window.test.mjs` cubre la selección de fecha en cualquiera de los dos
calendarios: completar el otro extremo, reemplazar un rango vencido, conservar
rangos de varios días, limpiar límites y volver a Siempre. Verifica 00:00 y
23:59:59.999 en Lima, el último instante incluido y la exclusión del día siguiente,
días de 23/25 horas en Madrid y el payload ISO real de guardado. La prueba de
interacción de `combo-quantity.test.mjs` exige calendarios `date` sin nuevos
controles y refresco de `combo-detail` tras guardar.

`combo-detail-modal.test.mjs` fija el detalle de Menús y combos a la anatomía
compartida (`size-md`, cabecera sin subtítulo, `modal-body`, `modal-record-data`,
`ModalItemComponents`, `ModalError`, estado en el pie), impide que el módulo
redefina cuerpo, tamaño o pesos y que vuelva a mostrar reglas de elección. Fija
también la hoja de menús: `crud-modal size-lg`, un solo `modal-body`, sin pasos ni
«Continuar», resumen del pie con el precio en `primary-700` y selección en tinte.

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
También exige categoría: el selector no ofrece «Sin categoría», el esquema
rechaza vacío o espacios con «Selecciona una categoría.», el campo muestra el
error con `aria-invalid`, no se guarda hasta elegir una, la categoría inactiva
solo se lista para el producto que ya la tiene y una única activa se preselecciona.

`table-batch.test.mjs` prueba una sola escritura, errores de resultado incierto
sin reintento automático, conservación del borrador, refresco de mesas y bloqueo
durante envío. También prueba el BFF con respuestas confirmadas/cortadas y la
propagación de correlación sin exponer cookies en logs. Verifica que
`tables_not_saved` conserve su mensaje específico y que el aviso 5xx muestre el
código de seguimiento.

`account-identity.test.mjs` protege los roles reales bajo el usuario, sus estados
remotos y el aislamiento de caché por usuario/empresa/local. Contexto verifica
que el selector conserve el local activo para empresa y plataforma; la empresa
se muestra por separado en la barra superior.

`plan-navigation.test.mjs` verifica Menús y combos y Concierge en Emprende, abastecimiento, recetas, reservas y gastos desde Impulso como suma acumulativa, ocultación de
módulos no incluidos incluso con rol de acceso total, separación de acceso y
lectura, catálogo de Plataforma y recarga de roles/permisos al cambiar el plan.

`concierge-usage-limit.test.mjs` protege el tope mensual de Fudia Concierge, solo
de Plataforma: el plan valida y envía `maxConciergeConversations` como número o
`null` (vacío = ilimitado) y lo muestra con los demás límites; el plan que ven las
empresas (`SubscriptionPlan`, `getCurrentOrganizationSubscription`) no gana el
campo y ningún archivo fuera de `src/modules/platform` (incluido Admin › Fudia
Concierge) referencia `concierge-usage` ni el tope. Ejecuta la API real con `fetch`
simulado (rutas `concierge-usage`, máximo 100 empresas, PATCH de configuración y de
excepción), los formatos regionales («0 / N», «N / Ilimitado»), validaciones, el
mapeo central de avisos (ámbar cerca del tope, rojo tope alcanzado, «Solo
medición» en modo medir), skeleton/error/vacío de cada vista, el modal de detalle
de solo lectura y la ausencia de topes comerciales fijos en el frontend.

Probar validaciones, permisos, filtros, errores y vacíos. Validar 390 px, escritorio
y accesibilidad. Puertas: lint, typecheck, tests y build.

Para Categorías, `category-filters.test.mjs` y `product-list.test.mjs` comprueban
búsqueda sin acentos, uso, estado, combinación, reinicio de página y filtrado del
catálogo completo. `tests/helpers/category-table-preview.mjs <directorio>` renderiza
la tabla real con datos ilustrativos a 390, 768 y 1440 px: tres filtros en lugar
del título redundante, controles táctiles de 44 px en móvil y sin desbordamiento de página.

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

El contrato de servicio separa preparación y cuenta. Probar destino remoto del producto, append sobre la misma cuenta Abierta sin líneas anteriores, botones por mozo, entrega completa antes de cierre/cobro, estados operativos sin modal exitoso y ajuste responsive desde 390 px. Los fixtures cobrables incluyen `billClosedAt`, estado Entregado y todos los productos entregados; Por entregar no equivale a Entregado. Rechazar cierre/cobro con Cocina, Barra o Sin preparación pendientes, incluso con estado agregado obsoleto; después del cierre no permitir consumo adicional ni con pago parcial.

Salón y Pedidos renderizan los controles reales para comprobar una única acción
«Confirmar entrega» en el pie, sin botones por producto. Un pedido mixto de Cocina,
Barra y Sin preparación debe tener todos sus productos pendientes por entregar; un estado
agregado Por entregar no permite saltar productos en preparación. Probar mozo asignado,
lectura sin permisos, bloqueo durante envío, ronda adicional con entregas previas
y transición a «Cerrar cuenta» después de servir todo. La entrega envía una sola
petición de estado, refresca las vistas afectadas y conserva el modal de error,
sin modal exitoso.

`order-delivery.test.mjs` verifica que ambos detalles (Salón/Pedidos) compartan
la cancelación de Sin preparación sin servir, oculten la acción ante preparación,
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
y retiro de la hoja/confirmación al perder gestión, conservando consulta.

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
la regla de unidades contables (contenido mínimo 1, pasos de 1, error visible,
esquema con el mismo mensaje del backend y kilos con fracciones),
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
selección blanca, pista contrastada y contadores índigo con número blanco,
acciones alineadas en `TabsBar`, iconos obligatorios de 16 px sin ocultar en móvil,
navegación por teclado y foco itinerante,
márgenes comunes de modal, área táctil de cierre y geometría móvil sin recorte.
Protege la cabecera índigo `primary-700`, título/cierre blancos, padding vertical
de 8 px y altura mínima de 64 px. Los helpers comprueban contraste AA y permiten
que los encabezados largos crezcan sin recortar texto.
La cabecera llega al borde superior y ocupa todo el ancho del modal, sin franja
degradada ni elementos decorativos anteriores.

La comprobación visual aislada `tests/helpers/shared-ui-preview.mjs` renderiza las
primitivas React reales con sus CSS y Plus Jakarta Sans del build, usando datos ilustrativos.
No consulta ni modifica datos de restaurantes. Con Playwright disponible y, si
hace falta, `FOODS_PLAYWRIGHT_PATH`/`FOODS_BROWSER_PATH` apuntando a sus instalaciones
locales, ejecutar `node tests/helpers/shared-ui-preview.mjs .next/shared-ui-preview`.
Guarda capturas y geometría para escritorio, tablet, 390 px y viewport bajo;
comprueba desborde, cierre táctil, márgenes y pie visible después del scroll.
Para revisar solo pestañas: `node tests/helpers/shared-ui-preview.mjs .next/tabs-preview tabs`.
Incluye escritorio, tablet y móvil de 360/390 px, cinco y seis pestañas, etiquetas largas,
contadores de cero y seis cifras, una acción a ancho completo o dos en columnas
iguales en móvil, grupos de dos sin fragmentos vecinos (incluido el último par),
iconos visibles de 16 px, contraste AA y foco
en la última opción tras el desplazamiento. Con `FOODS_PREVIEW_COMPILED_CSS=1`
comprueba también los estilos compilados del build.
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
Plus Jakarta Sans del build y CSS fuente de los mismos componentes. Para comprobar también
interacciones con los estilos compilados, definir `FOODS_PREVIEW_COMPILED_CSS=1`:
ambos helpers cargarán conjuntamente los CSS del build. Es una comprobación aislada
de composición, no una captura de una ruta autenticada ni garantía del orden de navegación.
Usa Playwright y las mismas variables opcionales de ruta descritas arriba.
Comprueba escritorio 1440×900, ventanas 1192×712 y 967×835, móvil 390×844,
viewport bajo 390×480, tablet táctil 768×1024, Delivery en escritorio/móvil,
skeleton y error. Mide el eje central de título/X, la primera línea de
etiqueta/nombre del mozo y el resumen inferior al mismo ancho del cuerpo. Exige que
el mozo preceda a la fecha en el DOM y comprueba los siete estados del pedido,
con contraste de texto de las píldoras de al menos 4.5:1.
Incluye Mozo sin asignar en estado Nuevo: fecha alineada al borde derecho y
Cancelar + Enviar comanda apilados, ambos a ancho completo. Comprueba que dos
botones que comparten una fila se repartan el ancho, y que una acción sola
no deje media columna vacía. `shared-ui-layout.test.mjs` protege el dueño
compartido `modal-action-layout`, sus subgrupos y la fila principal.
Verifica ausencia de desborde, panel dentro del viewport, cuenta y acciones visibles, cierre táctil de 44 px y
scroll de 20 productos conservando la posición del resumen. El formulario global
se revisa con `shared-ui-preview.mjs` en tamaños `md` y `lg`, incluido pie visible
tras recorrer campos largos. Ambos helpers cierran navegador y servidor temporal
al terminar. Estas comprobaciones aisladas no prueban la aplicación con sesión
autenticada ni sustituyen una revisión en teléfono físico.

## Dispositivo móvil de referencia

`mobile-reference-device.test.mjs` protege los defectos encontrados al recorrer
el catálogo de `page-routes.ts` y sus modales en el viewport de referencia (427 × 933,
también 390 px): pie fijo abajo en Receta y en los `form` heredados de pantalla
completa (Caja), hojas a ancho completo sin `calc(100vw - N)` (Nuevo pedido,
Nuevo cliente, gastos, usuario, roles, proveedor y canal de WhatsApp), 44 px
táctiles en QR, editor de texto, mapa, cierre del menú lateral y retorno de
`PageHeader`, márgenes de 16 px y Continuar compartido en Plataforma, retorno
y `EmptyState` en Próximamente, texto sin recortar en Mi perfil y estados
compartidos de carga, error con Reintentar y vacío en Reservas. En Compras la
variante de 760 px vuelve a aplicarse (orden, artículo y recepción a pantalla
completa, «Proveedor» en una columna, cuerpos a 16 px). Una guarda general
falla si una regla base posterior anula un `@media` del mismo archivo; `.sidebar`
de `shell.css` es la única excepción registrada y queda pendiente.

La revisión manual se hace con sesión real: un iframe del mismo origen de
427 × 933 dentro de una pestaña (comparte la cookie), animaciones desactivadas
y barras de desplazamiento superpuestas como en Android. Para carga y error se
intercepta `fetch` del iframe en las lecturas `GET /api/admin|platform/*` antes
de hidratar, sin tocar mutaciones. Mide ancho del documento, elementos fuera del
viewport, controles < 44 px (un `label` de 44 px que envuelve el input cuenta
como área táctil), textos recortados, padding de `.content` y, en modales,
rectángulo, márgenes del cuerpo y distancia del pie al borde inferior.
No sustituye una prueba en el teléfono físico.

## Rendimiento

`performance-budget.test.mjs` protege las decisiones de carga: barrels públicos
con solo re-exportaciones, `sideEffects` limitado a CSS, `optimizePackageImports`
para `@/design-system`, `@/providers`, `@/shared/management` y cada módulo,
`loading.tsx` del grupo admin con `RouteSkeleton` (solo opacidad, sin retraso y
con `prefers-reduced-motion`), `staleTimes.dynamic`, carga diferida de `qrcode`,
del buscador de combos y del editor de producto, y tamaño máximo de las imágenes
de interfaz. `source-contract.test.mjs` fija qué hojas del design system registra
el root layout.

Medición (Next 16 ya no imprime «First Load JS» en `next build`):

```bash
npm run build
npx next start --hostname localhost --port 5181
node scripts/measure-first-load.mjs http://localhost:5181
```

El script suma, en gzip, los chunks que descarga el HTML de cada ruta (incluidos
los que referencia el payload RSC) y su CSS; excluye polyfills `nomodule`. Las
rutas admin se sirven sin sesión, porque el shell valida en el cliente.

Línea base y resultado de la optimización del 18 oct 2026 (JS / CSS de primera
carga en kB gzip):

| Ruta | JS antes | JS después | CSS antes | CSS después |
| --- | ---: | ---: | ---: | ---: |
| `/login` | 281,3 | 200,7 | 32,1 | 27,0 |
| `/dashboard` | 289,6 | 189,1 | 52,8 | 36,5 |
| `/pos` | 336,6 | 189,7 | 67,0 | 35,6 |
| `/dining-room` | 336,6 | 199,6 | 67,0 | 39,5 |
| `/kitchen` | 336,6 | 181,4 | 67,0 | 34,4 |
| `/orders` | 336,6 | 226,1 | 67,0 | 42,2 |
| `/cash` | 336,6 | 233,2 | 67,0 | 36,4 |
| `/tables` | 336,6 | 226,2 | 67,0 | 34,0 |
| `/reservations` | 336,6 | 222,2 | 67,0 | 32,8 |
| `/combos` | 306,2 | 191,9 | 55,2 | 35,6 |
| `/products` | 306,2 | 228,6 | 55,2 | 32,5 |
| `/recipes` | 306,2 | 186,4 | 55,2 | 34,5 |
| `/sales` | 286,0 | 190,2 | 50,8 | 34,4 |
| `/purchases` | 297,8 | 234,4 | 54,3 | 35,8 |
| `/inventory` | 297,8 | 185,6 | 54,3 | 34,7 |
| `/customers` | 285,6 | 225,3 | 50,4 | 34,1 |
| `/settings/organization` | 283,6 | 257,6 | 49,7 | 40,2 |
| `/settings/users` | 276,2 | 223,4 | 48,8 | 34,8 |
| `/platform/onboarding` | 253,7 | 244,4 | 35,6 | 35,0 |
| `/table/[qr]` | 276,0 | 149,5 | 19,6 | 25,3 |

Hallazgos que explicaban la línea base: los barrels arrastraban react-select +
emotion (~53 kB) y el envoltorio de Mapbox (~10 kB) a todas las rutas, incluidos
login y la carta QR pública; cada ruta de Operaciones cargaba el módulo completo
(~50 kB), y `qrcode` viajaba con todas ellas. El logo (889 kB, mostrado a 36 px y
precargado en cada visita) y la imagen de login (1,1 MB a 120 px) se descargaban
en su resolución original; ahora pesan 16 kB y 66 kB. La carta QR sube 5,7 kB de
CSS por registrar las hojas compartidas en el root layout, a cambio de 126 kB
menos de JS.

En navegador (build de producción, sesión real): el toque en el menú lateral
cambia URL y muestra el skeleton en 16–80 ms; una pantalla nueva aparece hacia
los ~340 ms (React mantiene un fallback de Suspense ~300 ms; el payload RSC
tarda 10–60 ms) y una visitada en los últimos 30 s abre en 50–70 ms. Se revisaron
Dashboard, Punto de venta, Salón, Pedidos y Menús y combos (incluido el buscador
diferido «Agregar opción») en escritorio y, con el iframe de 427 × 933 y 390 px,
en móvil. Los sondeos de 10 s de Salón, Cocina y del saldo del catálogo se
mantienen: son comportamiento aprobado y tienen pruebas propias.

`order-timeline.test.mjs` verifica el seguimiento del pedido: hitos de salón y delivery, solo horas reales del backend (alta, cierre de cuenta, cierre de la atención), pago sin hora inventada y corte en pedidos cancelados.
`modal-redesign.test.mjs` verifica que Productos, Detalles y Seguimiento compartan
la sangría interior (16 px en escritorio, 12 px en móvil), sin cambiar las pestañas
redondeadas, la superficie o las acciones del modal.

`sales-detail.test.mjs` verifica además el agrupado identidad/atención y
pestañas/panel (pastillas compartidas y superficie común, sin tarjeta independiente),
radio original del contenedor (14 px), pista y selección con `radius-pill` compartido,
sangría interior de los paneles,
filas alineadas de «Detalles», reloj junto a la fecha, «Mesa» sin duplicar su prefijo,
conservación simultánea de cliente y mozo, etiquetas de ancho natural en escritorio,
tres pestañas visibles en móvil y cantidad con el mismo fondo oscuro y
texto blanco que el icono de pagos. La comprobación en navegador mantiene los
productos, pagos y detalles accesibles, resumen fijo y acción secundaria a
ancho completo en móvil de 390 px.
Para la regresión del pie de Ventas, verificar además anchos de 375, 412, 480 y
600 px y una altura de 400 px: el modal se acota al overlay y reserva filas para
resumen y acciones. Comprobar el límite inferior del botón, no solo su presencia en el DOM.

### Paleta compartida

Después del build, `node tests/helpers/palette-preview.mjs` sirve en
`http://localhost:5196` las rampas reales y los componentes `Status` y `Button`.
Es una muestra aislada sin API ni operaciones del restaurante. Revisar escritorio
y 390 px: verde con menta, rojo con rosado y ámbar con crema; etiquetas e iconos
conservan el significado de cada estado. `design-tokens.test.mjs` comprueba el
contraste de texto, acciones destructivas, hover y la superficie verde de Cocina.
