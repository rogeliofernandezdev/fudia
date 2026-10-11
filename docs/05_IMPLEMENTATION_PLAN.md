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

## Fase 4 — Hospitality / Hoteles (evolución futura)

Esta fase se activa después de posicionar y consolidar FudIA como plataforma para restaurantes. La expansión a hotelería debe reutilizar la base multiempresa/multilocal y mantener separados los contextos de restaurante y hotel.

- Integración inicial con PMS existentes antes de construir un PMS completo propio.
- Propiedades, habitaciones, huéspedes y estadías.
- Room service conectado al motor actual de pedidos del restaurante.
- Cuenta de habitación / folio y cargos asociados a la estadía, sin tratarlos como método de pago.
- Solicitudes de servicio, housekeeping y estados de habitación.
- Integración de restaurante, bar y otros puntos de consumo dentro de una propiedad hotelera.
- Evolución posterior hacia un PMS ligero nativo solo si la demanda y el posicionamiento del producto lo justifican.

## Puertas de calidad

Cada fase exige contrato actualizado, migraciones reversibles, pruebas unitarias
y de integración, autorización por empresa/local, accesibilidad, verificación
visual a 390 px y escritorio, lint, typecheck y build.
