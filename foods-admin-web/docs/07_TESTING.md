# Pruebas

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
que el selector conserve el local activo para usuarios de empresa; la empresa
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

`inventory-settings-dialog.test.mjs` protege la estructura compartida del modal
de niveles (formulario, campos y footer), estilos responsive desde 390 px sin
CSS local, validación explícita, normalización de niveles, guardado por botón y
una sola petición pendiente. Verifica bloqueo de campos/cierres y conserva el
cierre global únicamente al comunicar la respuesta del API.

`inventory-list.test.mjs` verifica Mínimo / Reorden en una sola línea, incluidos
cero y fracciones, sin texto secundario ni CSS correctivo. Conserva nueve
columnas, paginación compartida y desplazamiento horizontal en móvil.

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
