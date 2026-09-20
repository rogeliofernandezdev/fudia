# Plan de implementación

## Fase 0 — Fundaciones

- ADR, modelo de tenencia, autenticación, permisos y auditoría.
- Esqueleto de los cuatro proyectos y CI.
- OpenAPI inicial y design tokens compartidos.
- Entornos, secretos, logs, métricas y trazas.

## Fase 1 — Venta esencial

- Empresas, locales, usuarios y cajas.
- Menú, productos, modificadores y precios.
- Mesas, POS, comandas y KDS.
- Cobro, cierre de caja e impresión.
- Boleta/factura electrónica mediante proveedor homologado.

## Fase 2 — Control del negocio

- Recetas, insumos, inventario y mermas.
- Proveedores, compras y recepciones.
- Dashboard y reportes de ventas, caja, costos y tiempos.
- Alertas de stock y auditoría ampliada.

## Fase 3 — Omnicanal

- Pedidos por WhatsApp con handoff humano.
- Recojo, delivery y seguimiento.
- Clientes, promociones y fidelización básica.
- Operación offline acotada y reconciliación.

## Puertas de calidad

Cada fase exige contrato actualizado, migraciones reversibles, pruebas unitarias
y de integración, autorización por empresa/local, accesibilidad, verificación
visual a 390 px y escritorio, lint, typecheck y build.
