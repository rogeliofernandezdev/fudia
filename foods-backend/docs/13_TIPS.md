# Propinas

## Alcance y configuración

La migración `000095_tips` incorpora la configuración y los registros financieros
de propinas dentro del módulo billing. No requiere secretos ni servicios nuevos.
Se conservan las cookies de sesión y la autorización de empresa/local existente.

`GET/PUT /v1/admin/tip-settings?scope=organization|location` consulta o actualiza
el modo `disabled`, `suggest` o `collect`, porcentaje sugerido, sugerencia en la
precuenta, permiso de monto libre y tipos de atención. Solo el administrador
reservado puede modificarla, además del permiso `organizations.manage` de la ruta.
Un local hereda del restaurante; `inherit:true` elimina su excepción. Los canales
disponibles se entregan en `channelOptions`; el frontend no mantiene su catálogo.

Sin configuración explícita, Perú y los demás países quedan desactivados; Chile
inicia en sugerencia de 10 % para Salón. Son valores iniciales de producto, no un
motor de cumplimiento fiscal o laboral. No se aplican reglas legales inferidas a
otros países. La moneda y los decimales proceden del perfil fiscal del local.

## Cobro separado del consumo

`PaymentInput` y las líneas de pago dividido admiten `tip` opcional:

```json
{"kind":"percentage","value":10,"directToStaff":false}
```

`kind:fixed` interpreta value como dinero. El backend recalcula el porcentaje
sobre el total del consumo, excluyendo el costo de delivery; redondea según los
decimales de la moneda y rechaza montos fijos con precisión incompatible. No acepta
una propina cuando el modo es sugerencia/desactivado o el canal no está habilitado.

El campo `amount` del pago continúa siendo exclusivamente consumo. Una aceptación
de propina corresponde a la cuenta completa y se registra **una vez por pedido**,
incluso si el consumo se paga parcialmente. Solo una línea de un pago dividido
puede llevar propina. El bloqueo del pedido y la restricción única evitan duplicados;
cualquier error revierte también el pago y sus movimientos de caja.

| Caso | Efectivo esperado | Pendiente de entregar | Ventas |
| --- | --- | --- | --- |
| Sugerencia sin aceptar | No cambia | No cambia | No cambia |
| Propina cobrada en efectivo | Aumenta | Aumenta | No cambia |
| Propina cobrada por otro medio | No cambia | Aumenta | No cambia |
| Entrega directa al personal | No cambia | No cambia | No cambia |
| Entrega al trabajador desde el cajón | Disminuye | Disminuye | No cambia |
| Entrega por transferencia | No cambia | Disminuye | No cambia |

Una propina directa se registra informativamente, sin movimiento ni deuda a cargo
del restaurante. No se procesa ningún cargo con tarjeta desde este desarrollo:
se registra el cobro realizado con el medio seleccionado, igual que el POS actual.

## Entrega y trazabilidad

`GET /v1/admin/cash-shifts/{id}/tips` devuelve el reporte del turno, el disponible
acumulado autorizado y los trabajadores habilitados. El cajero solo consulta sus
turnos y las propinas recibidas en ellos; el administrador puede consultar el local.
Ser colaborador temporal para cobrar no habilita la consulta de estas finanzas.

`POST /v1/admin/cash-shifts/{id}/tip-payouts` registra destinatario, monto, medio
`cash|transfer`, observación, actor y fecha. Exige un turno abierto asignado y ser
su cajero de apertura o administrador reservado. El destinatario debe ser un
trabajador activo del alcance local; los administradores se excluyen del selector.

`requestKey` identifica el intento del cliente (no una entidad persistida). La misma
clave, actor, turno y contenido devuelve la entrega existente. Reutilizarla con
otros datos produce 409. El cliente conserva la clave al reintentar un error de red.
Los UUID de las entidades los genera PostgreSQL con `gen_random_uuid()`.

El turno se bloquea antes de modificar efectivo. Los recibos de propina pendientes
se bloquean en orden determinista y se vuelven a consultar sus asignaciones en una
segunda sentencia bajo READ COMMITTED. Así dos entregas concurrentes no consumen
el mismo saldo. Cada asignación queda en `tip_payout_allocations`. Se valida también
que una entrega en efectivo no supere el efectivo esperado del turno.

El movimiento de efectivo usa `cash_tip` o `tip_payout`; una entrega es una salida
del cajón, separada de un gasto operativo. No hay distribución automática por
porcentaje entre trabajadores: el operador elige explícitamente quién recibe.

## Reportes, cierres y documentos

`CashShiftReport.tips` incluye recibido en efectivo, otros medios, directo,
entregado en efectivo, por transferencia, pendiente, recibos y entregas detalladas.
El pendiente del reporte se refiere a las propinas recaudadas en ese turno. El
`available` del espacio de entrega incluye además turnos anteriores permitidos.

El cierre guarda una copia del reporte con las propinas. Entregar pendientes desde
otro turno no altera esa copia. Reportes históricos previos a esta migración no
inventan información de propinas.

La precuenta de 80 mm presenta una sugerencia voluntaria cuando está configurada.
El ticket distingue consumo y propina aceptada, medio, total recibido y entrega
directa. El formato del documento y el visor PDF compartido se conservan.

Las devoluciones actuales continúan revirtiendo **consumo**. No devuelven ni borran
automáticamente una propina, especialmente si ya fue entregada al trabajador. Una
devolución específica de propina y su recuperación al personal queda fuera de este
flujo; requiere un movimiento propio, no manipular los importes de ventas.

## Validación y despliegue

Las pruebas de dominio cubren porcentaje, monto fijo, redondeo, precisión monetaria
y valores inválidos. La integración con PostgreSQL comprueba herencia, pagos
parciales, efectivo/tarjeta/directo, transacciones revertidas, idempotencia,
entregas concurrentes, privacidad entre cajeros, destinatarios de otra empresa y
conservación del cierre.

Aplicar la migración y desplegar el backend antes del frontend. La migración no
modifica cobros históricos. Su reversión se niega si existen registros financieros
de propinas: nunca elimina esa trazabilidad para volver de versión.
