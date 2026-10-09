# Pruebas

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
Cocina, Barra y Entrega directa, datos comerciales persistidos y auditoría.
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

La suite incluye atención completa con Cocina/Barra/Entrega directa, exclusión de directos del KDS, entrega por producto, consumo adicional solo en cuenta Abierta, idempotencia y entrega completa antes de cerrar/cobrar. Se bloquea cierre/cobro con productos pendientes, incluidos agregados obsoletos y cuentas históricas cerradas prematuramente, sin persistir pagos. Después del cierre se rechaza consumo adicional; pago parcial mantiene mesa ocupada y pago completo la libera automáticamente. Se prueba tenant/local, mozo asignado y cierre/cobro concurrentes. Migración 73 se valida en PostgreSQL efímero, nunca con datos del negocio.

`table_account_safety_integration_test.go` prueba las rutas de preparación con
sesión y rol Cocinero reales: pedidos y rondas de Barra/Entrega directa no pueden
mutarse mediante la compatibilidad histórica de Cocina, aunque sus instantáneas
ya estén entregadas. Conserva pedidos/rondas verdaderamente históricos.
La cancelación verifica entrega directa sin servir, mesa reutilizable, reversión
de stock sin duplicados y bloqueo por preparación, entrega, pago, finalización,
transporte, otro mozo u otro local; incluye cuentas mixtas y estados agregados
desactualizados.

`product_list_integration_test.go` comprueba 15 porciones, descuento de 3 y
reversión, saldo separado por local y día en zona horaria distinta de UTC,
inventario fraccionario con unidad, Sin control null, agotado manual, inactividad,
horario y ausencia de cupo del día. Conserva la descripción para editar.
