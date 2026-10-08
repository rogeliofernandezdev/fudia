# Pruebas

Probar invariantes, casos de uso, contrato HTTP y persistencia. Puertas: go test,
go vet, validación OpenAPI y build.

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
