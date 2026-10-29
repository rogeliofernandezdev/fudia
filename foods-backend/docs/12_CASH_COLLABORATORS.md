# Colaboradores temporales para cobrar en POS

## Diagnóstico

La asignación de equipo ya persistía en `cash_shift_users`, pero POS y las rutas
de cobro exigían `cash.manage`. El mesero no tenía acceso a POS y el selector
permitía incorporar administradores o al propio usuario. Los pagos bloqueaban
el pedido, pero no coordinaban la vigencia de la asignación con retiro/cierre.

## Implementación

El mozo colaborador solo cobra sus propias atenciones de Salón. Se usa
`orders.waiter_id`, que el flujo de comanda vincula automáticamente con el
usuario autenticado; no hay una asignación manual nueva de mesas. Una mesa
de otro mozo, sin responsable o un pedido de Recojo/Delivery no concede cobro
temporal. `cash.manage` conserva la autorización del cajero para cobrar otras
atenciones. La consulta de POS temporal filtra también filas, total y sumas
por el mozo vinculado al pedido.

Cobro individual y dividido comprueban esa vinculación después de bloquear el
pedido y antes de escribir pagos o movimientos, dentro de la misma transacción.
Un rechazo devuelve `403 cash_collection_order_forbidden`, sin cobros parciales.
La interfaz comparte `canCollectOrder` entre Mesa, Pedidos y POS. El API conserva
la autoridad definitiva aunque se cambie el UUID desde el navegador.

- Billing valida permiso administrativo o asignación activa, sin modificar roles.
  La capacidad temporal se limita a cobros y lectura operativa de POS.
- Identity proyecta `cashCollection` en el contexto mediante un puerto inyectado
  desde composición. Los permisos y accesos permanentes permanecen intactos.
- POS usa un endpoint mínimo de turno sin saldos ni movimientos administrativos.
  Devoluciones conservan `cash.manage`; Caja conserva sus permisos anteriores.
- El selector y la escritura comparten la validación de colaborador. El cliente
  también excluye al actor, asignados y turnos cerrados y revalida la selección.
- Cobrar bloquea turno/asignación en modo compartido; retirar/cerrar usan bloqueo
  exclusivo del turno. El bloqueo del pedido sigue evitando sobrecobros.
- `assigned_by`, `assigned_at`, `unassigned_at`, `payments.created_by` y `shift_id`
  conservan la trazabilidad existente. El abridor sigue como responsable.
- Solo el usuario que abrió el turno puede cerrarlo, con permiso `cash.manage`.
  Otro cajero o administrador recibe `403 cash_shift_owner_required`, sin
  guardar conteos, informes ni modificar equipo. La interfaz oculta el cierre
  a quienes no son el abridor, comparando UUID y no nombres.

## Elección de caja por el administrador

El administrador reservado de empresa puede cobrar sin ser colaborador ni abrir
su propio turno. `GET /pos/access` devuelve `canChooseShift`, las cajas con turno
abierto del local activo y su abridor, sin saldos. Si solo hay una, `shift` la
selecciona automáticamente. Con varias, el formulario exige una elección; con
ninguna, impide cobrar. `assignedShift` conserva el turno realmente asignado para
las devoluciones, cuya autorización no se amplía.

Cobro individual y dividido aceptan `shiftId` opcional únicamente para el
administrador. El backend valida el rol reservado, empresa, local y estado del
turno dentro de la transacción, y lo bloquea en modo compartido frente al cierre.
Un cajero o mozo que envía `shiftId` recibe 403; conservan su turno autorizado.
El pago guarda al administrador en `created_by`, y el movimiento usa el turno
elegido. No se modifica `opened_by`, no se crea una asignación y solo el abridor
conserva el derecho de cierre. El efectivo se entrega físicamente a la caja elegida.

## Archivos y responsabilidad

`billing/application/{cash,payments,collection}.go` y adaptadores PostgreSQL:
reglas, autorización limitada, transacciones y consultas con alcance.
`identity/application/administration.go`, dominio de contexto y composición:
capacidad de sesión sin dependencia de implementaciones de otros módulos.
`transport/httpapi/routes.go` y transporte de Billing: autorización por ruta.
`api/openapi.yaml`: contrato de contexto y acceso mínimo de POS.
Frontend: equipo de Caja, POS, proveedores de sesión y navegación temporal.
Pruebas de aplicación, integración y frontend acompañan estos cambios.

## Alcance y despliegue

La lectura financiera de Caja se limita a los turnos abiertos por el usuario;
solo el administrador reservado de empresa (o de plataforma) puede consultar
todos los del local. `cash.read`, `cash.manage`, `cash.expected.read` o administrar
cajas no amplían ese alcance. Listado, búsqueda y total se filtran en SQL antes
de paginar. Detalle, equipo e informe (incluido el snapshot del cierre) validan
propietario en backend y devuelven 404 para turnos ajenos. Una asignación temporal
no concede estos datos; el turno financiero actual devuelve null si solo es
colaborador. POS conserva su acceso mínimo y la capacidad de cobrar.

El catálogo de cajas conserva ocupación y responsable, con `detailsVisible=false`
para cajas ajenas. Sus importes, notas, movimientos y equipo no se exponen, y
el frontend omite detalle y acciones de operación sobre esa proyección. El saldo
del resumen solo agrega los turnos visibles. No modifica historiales ni requiere
una migración adicional. La autorización del cierre sigue reservada al abridor.

No crea tablas ni modifica autenticación o roles de los colaboradores.
La administración de cajas usa el permiso separado `cash.registers.manage`;
la migración 94 lo incorpora únicamente a administradores reservados existentes.
Requiere desplegar backend y frontend. La navegación y el turno se refrescan cada 30 segundos y
al recuperar foco; esta demora visual no concede cobros tras una revocación,
porque el backend vuelve a validar bajo bloqueo. Los permisos de lectura que
el rol ya poseía siguen vigentes. Un cobro iniciado con los bloqueos ya adquiridos
se confirma antes del retiro/cierre, conservando el orden transaccional.

## Verificación realizada

- PostgreSQL 16 desechable: 20 grupos de regresión de Caja, pagos y aislamiento;
  prueba final de colaboradores con sus seis escenarios aprobada, incluido cierre
  concurrente y cobro dividido. Contenedores eliminados al terminar.
- Go: pruebas generales, arquitectura, `go vet ./...` y build del API aprobados.
- Frontend: lint sin errores (seis advertencias anteriores), TypeScript y build
  de producción aprobados. Pruebas de POS, equipo, permisos y navegación aprobadas.
  La suite general verificó 567 casos; su única expectativa antigua exigía la
  consulta financiera anterior de turno. Se actualizó al endpoint mínimo y se
  verificaron los 44 casos del archivo de contratos, sin fallos restantes.
- No se utilizaron datos de producción ni se desplegó durante la implementación.
