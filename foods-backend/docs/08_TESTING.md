# Pruebas

`cash_collaborators_integration_test.go` recorre rutas con cookie real para
asignación de mozo, exclusión de administradores/actor/inactivos/otros locales,
duplicados y otro turno, capacidad de sesión sin `cash.manage`, cobros individual
y dividido, límites de lectura temporal y rechazo de operaciones administrativas.
Comprueba atribución, saldo con dos autores simultáneos, doble cobro completo,
revocación/historial y cierre. Una barrera real de filas pone un cierre delante
de un cobro ya autorizado: el cobro espera y se rechaza después del cierre sin
persistir ni provocar interbloqueo. Se ejecuta en PostgreSQL 16 temporal.
`billing/application/collection_test.go` cubre autorización administrativa,
asignación temporal y fallos de permisos/base sin conceder permisos permanentes.

Medios de pago: `payment_methods_total_limit_integration_test.go` comprueba que
un inactivo consume plaza, que los filtros no alteran la capacidad y que tres
altas concurrentes con dos plazas disponibles terminan en cuatro registros y
un rechazo 409. Las pruebas de país verifican cuatro medios para Perú y solo
Efectivo/Tarjeta para otros países. El ciclo administrativo inicia con tres filas
de prueba para permitir el cuarto registro; desactivar no libera plazas.

`company_admin_locations_integration_test.go` verifica con PostgreSQL temporal
que el administrador de empresa ve los tres locales con una sola asignación,
incluye un local recién creado y cambia mediante la cookie real sin perder
permisos, perfil ni opciones de Caja. Comprueba autorización de reserva, login
con el local original inactivo, rechazo de otra empresa/local inactivo y
revocación inmediata. Un rol personalizado con el mismo nombre o `users.manage`
permanece limitado a su asignación; un administrador inactivo no concede acceso.

`inventory/application/location_settings_test.go` comprueba niveles enteros para
unidades contables, fracciones medibles, normalización y rechazo si falta la
metadata de la unidad. `inventory_settings_units_integration_test.go` comprueba
el catálogo en el listado real, 400 sin escritura para botellas y aceptación de
fracciones en kg; requiere PostgreSQL efímero.

El dashboard verifica ventas por categoría con dos decimales y solo pedidos
totalmente pagados. La regresión de cantidades fraccionarias comprueba el
redondeo después de sumar, en hoy/7/30 días, sin modificar cantidades ni precios
persistidos. Ejecutar con PostgreSQL temporal para no omitir la integración.

`payment_ticket_integration_test.go` comprueba identidad real del restaurante,
perfil del local, aislamiento empresa/local y consulta sin escrituras. La prueba
de pagos verifica identidad y nombres remotos después del pago dividido confirmado.

`cash_report_integration_test.go` verifica productos sin duplicación por cobros
parciales/divididos, cantidades/precios, movimientos, autores, medios de pago,
fechas UTC, instantánea persistida, conteo y cierre repetido. Simula fallo al
persistir el informe para comprobar rollback del cierre y del conteo. Cambiar
nombres después de cerrar no modifica el documento emitido. Cubre aislamiento
empresa/local, permisos reales, sesión anónima y cierre ciego antes/después del
cierre. La migración 79 y la suite se ejecutan en PostgreSQL temporal.

`product_availability_history_test.go` verifica motivo obligatorio y longitud,
snapshot anterior/nuevo, autor persistido, hora UTC, no-op sin duplicar historial,
rollback ante fallo de auditoría, cadena completa con cambios concurrentes,
paginación de días anteriores, legado sin datos inventados, aislamiento por
empresa/local y permisos de lectura/escritura para Admin y Operación.
La migración 78 de índice se comprueba arriba/abajo en PostgreSQL temporal.

Las pruebas de alta de producto verifican cantidad inicial positiva, 15 porciones
en el local/día correcto, listados reales, ajustes y Agotado/Reactivar posteriores,
edición comercial sin sobrescribir cantidades, permisos y aislamiento. PostgreSQL
temporal comprueba rollback completo si fallan disponibilidad o auditoría.
Todas crean el producto con una categoría activa (`seedProductCategory`); las
unitarias de `menu/application` y `menu/domain` verifican que falte, esté vacía o
mal formada (`category_required`), que una categoría inactiva o ajena se rechace
dentro de la transacción (`invalid_category`) y que la edición permita conservar la
categoría actual del producto.

El onboarding (`platform_mvp_integration_test.go`) verifica la precarga de
`seed_operational_defaults`: categoría «General» (`both`), «Mesa 1» y «Mesa 2» en
«Principal» para 2 personas, libres y con QR, modalidad `dine_in`, sin productos,
pedidos ni turnos; un reintento directo de `SeedOperationalDefaults` no duplica
filas ni sobrescribe la modalidad, y una zona fuera de las plantillas se crea en
el mismo lote junto con su mesa. `platform_onboarding_regression_test.go` fuerza
un fallo en la auditoría final y comprueba que no queda empresa, categoría, mesas
ni puesta en marcha. Las unitarias de `platformadmin` cubren los valores e
invariantes de la precarga y que el paso corre dentro de la transacción con
rollback ante fallo.
Las unitarias también comprueban que empresa y roles son una llamada cada uno,
que el rol de administrador asignado es el devuelto por `SeedRoles` y que un
fallo dentro del batch de `CreateTenant` se atribuye a su paso
(`create_location` → `location_code_already_registered`). La integración
reintenta `SeedRoles` sin duplicar roles, valida el orden de sus ids y que el
local quede enlazado al perfil fiscal configurado.

Probar invariantes, casos de uso, contrato HTTP y persistencia. Puertas: go test,
go vet, validación OpenAPI y build.

`user_capacity_integration_test.go` verifica rechazo de altas/reactivaciones al
alcanzar `max_users`, conservación del estado tras rechazo, edición y
desactivación permitidas, cupo liberado, idempotencia, aislamiento, plan ilimitado
y suscripción ausente. Dos reactivaciones o alta/reactivación concurrentes
comparten el último cupo sin superar el límite. Ejecutar en PostgreSQL temporal.

`inventory_units_integration_test.go` verifica catálogo inicial para nuevas
empresas, Bolsa/Paquete, alta y duplicados, aislamiento, FK y validación de
unidades de artículos/entradas, auditoría y permisos de Compras/Inventario.
La migración 76 conserva códigos históricos y cantidades al subir/bajar en
PostgreSQL temporal. Las unidades nuevas no se comparten con otros tenants.

`table_batch_test.go` protege una sola consulta para cualquier tamaño del lote.
La integración en PostgreSQL temporal verifica orden, QR únicos, auditoría
atómica, zonas y nombres inválidos, aislamiento por local, concurrencia y rollback
si falla la auditoría. Las pruebas del servidor verifican que el deadline de
procesamiento vence antes del límite de respuesta y permite enviar un error JSON.

`product_update_integration_test.go` comprueba actualizaciones exitosas con
Cocina, Barra y Sin preparación, datos comerciales persistidos y auditoría.
Cubre la conservación de destino/tipo omitidos, rechazo de destinos inválidos
y aislamiento entre empresas. Se ejecuta con PostgreSQL temporal.

Las pruebas de catálogo de roles cubren alcance efectivo por plan, dependencias,
contadores visibles, permisos compartidos, acceso total de Plataforma, rechazo
de asignaciones fuera del plan y conservación de permisos ocultos al editar.
La integración verifica el alta Emprende con Menús y combos y la migración 74
(up/down, módulos ausentes/inactivos/activos y suscripción cancelada) en
PostgreSQL temporal, sin modificar datos del negocio.

La migración 75 verifica Inventario y Compras en Emprende: altas nuevas, módulos
ausentes/inactivos/activos, suscripciones trial/active/past_due/cancelled,
idempotencia, aislamiento de otros planes, precios/límites/contratos intactos y
rollback con habilitaciones previas y ediciones posteriores. Roles y catálogos
presentan ambos módulos sin conceder acciones nuevas a los otros perfiles.

La migración 90 verifica la matriz acumulativa de planes, precios/límites
intactos, paso de Emprende con abastecimiento al plan no vendible, Emprende sin
abastecimiento, suscripción cancelada intacta, Compras/Costos/Concierge en
Impulso y Escala, y rollback sin sobrescribir ediciones posteriores. El catálogo
de roles de Emprende expone Concierge y oculta Inventario, Compras, Kardex y
Costos.

`concierge_usage_integration_test.go` cubre el límite mensual de Concierge:
conteo idempotente por `conversationId`, modo `measure` sin bloqueo ni aviso
pasado el tope, `warn` con aviso al porcentaje y al 100 % sin bloquear,
`enforce` con 409 explícito sin cortar la conversación en curso, rechazos
contados aparte, excepción propia/ilimitada/del plan y cambio del tope del plan
aplicados sin redeploy, tokens Lite/Flash idempotentes por turno, reporte de una
conversación no registrada sin bloqueo e historial. Ocho aperturas concurrentes
con tope 3 abren exactamente 3. Una sesión real de administrador de empresa
recibe 403 en todas las rutas de Plataforma (incluidos planes) sin aplicar
escrituras, y contexto, suscripción, planes disponibles y Concierge settings no
exponen topes, modo ni uso. La migración 91 se prueba down/up (topes base,
`emprende-abastecimiento` = Emprende, `legacy` ilimitado, política única en
`measure`/80 %). `concierge/domain/usage_test.go` prueba tope efectivo, bloqueo
y avisos. Ejecutar en PostgreSQL temporal.

## Combinaciones y conversiones de compra

`inventory_combinations_integration_test.go` prueba siembra para nuevas
empresas, filtro por unidad, catálogo y artículos aislados, altas de tipos y
pares, conflictos de código, varias conversiones por artículo, una preferencia,
recepción de cinco paquetes de diez botellas (50), venta por botella (49),
bolsas de 0.5 kg, stock cero al crear, factores históricos intactos y rollback
completo ante combinación inválida o fallo de auditoría. Migración 77 se prueba
down/up sobre PostgreSQL temporal, conservando UUID, factor y saldo. Las rutas
de catálogo y conversión rechazan escritura de usuarios de solo lectura.

## Cuenta de mesa y estaciones

La suite incluye atención completa con Cocina/Barra/Sin preparación, exclusión de directos del KDS, entrega por producto, consumo adicional solo en cuenta Abierta, idempotencia y entrega completa antes de cerrar/cobrar. Se bloquea cierre/cobro con productos pendientes, incluidos agregados obsoletos y cuentas históricas cerradas prematuramente, sin persistir pagos. Después del cierre se rechaza consumo adicional; pago parcial mantiene mesa ocupada y pago completo la libera automáticamente. Se prueba tenant/local, mozo asignado y cierre/cobro concurrentes. Migración 73 se valida en PostgreSQL efímero, nunca con datos del negocio.

`table_account_safety_integration_test.go` prueba las rutas de preparación con
sesión y rol Cocinero reales: pedidos y rondas de Barra/Sin preparación no pueden
mutarse mediante la compatibilidad histórica de Cocina, aunque sus instantáneas
ya estén entregadas. Conserva pedidos/rondas verdaderamente históricos.
La cancelación verifica sin preparación sin servir, mesa reutilizable, reversión
de stock sin duplicados y bloqueo por preparación, entrega, pago, finalización,
transporte, otro mozo u otro local; incluye cuentas mixtas y estados agregados
desactualizados.

`product_list_integration_test.go` comprueba 15 porciones, descuento de 3 y
reversión, saldo separado por local y día en zona horaria distinta de UTC,
inventario fraccionario con unidad, Sin control null, agotado manual, inactividad,
horario y ausencia de cupo del día. Conserva la descripción para editar.

## Delivery y recojo de Concierge

`pickup_channels_integration_test.go` verifica que el alias histórico
`mostrador` y `recojo` comparten filtros y conteos de Pedidos/Cocina, lectura de
POS y altas canónicas, sin reescribir los pedidos antiguos ni mezclar empresas
o locales. El dashboard prueba agregación conjunta de cobros de ambos códigos.
`orders/domain/channels_test.go` protege la única opción «Recojo en local» y la
compatibilidad. La integración requiere PostgreSQL efímero.

`delivery/domain/fees_test.go` prueba la distancia en línea recta redondeada a
0.01 km, la elección del primer rango que cubre la distancia, la ausencia de
cobertura sin rangos o más allá del mayor límite y la validación de rangos
(límites, decimales, duplicados y máximo de 10).
`concierge_delivery_integration_test.go` cubre: sin ubicación ni rangos solo hay
recojo; opciones y cotización dentro y fuera de cobertura; tarifa aceptada
desactualizada rechazada con `delivery_fee_changed`; pedido delivery sin mesa con
precios del catálogo, envío y distancia guardados; reintento idempotente con el
mismo pedido aunque cambien las tarifas; pedido de recojo sin dirección ni envío;
estado visible solo para la conversación dueña y `active=false` al cancelar. Se
ejecuta con `DATABASE_URL` de una base efímera (CI), nunca con datos del negocio.

`identity/application` verifica que, con el acceso leído al autenticar, la
petición autoriza sin otra consulta, rechaza permisos no otorgados, no reutiliza
el acceso para otro local y deja pasar al administrador de plataforma.
`orders/application` comprueba que la preparación precarga todos los productos en
una consulta, salta los modificadores de productos que no los tienen, sigue
rechazando modificadores ajenos y productos no disponibles, y usa las consultas
individuales para productos fuera de la precarga. `inventory/application`
comprueba un único bloqueo ordenado para todos los productos del pedido y el
error `product_unavailable` cuando falta alguno. La integración existente cubre
la inserción en lote con combos, modificadores, recetas y Concierge.

`inventory_presentation_content_integration_test.go` verifica el catálogo con
`countable`, que las combinaciones expongan `unitCountable` (también al crear
una nueva), que un «Paquete = 0,001 botellas» se rechace con
`invalid_presentation_content` sin dejar el artículo a medias, que «Paquete = 6
botellas» se acepte, que agregar después una presentación fraccionaria también
se rechace y que una bolsa de 0,5 kg siga permitida. Las unitarias de
`inventory/domain` y `inventory/application` cubren la regla y que la
presentación rechazada nunca se guarda.
