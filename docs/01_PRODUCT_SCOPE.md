# Alcance del producto

## Propósito

Reducir errores y tiempos desde la toma del pedido hasta el cobro, mantener el
inventario trazable y emitir comprobantes electrónicos para Perú.

## Usuarios

- Administrador o propietario.
- Encargado de local.
- Cajero.
- Mozo.
- Cocina y bar.
- Almacén y compras.
- Repartidor.

## Canales

- Atención en mesa y mostrador.
- Recojo en local.
- Delivery propio.
- Pedido recibido por WhatsApp mediante integración.

## Fase inicial

- Multiempresa y multilocal.
- Catálogo de platos, modificadores, combos, precios e impuestos.
- Mesas, reservas simples y comandas.
- POS, caja, medios de pago, división de cuenta y propina.
- Pantalla de cocina (KDS) e impresión de comandas.
- Boletas, facturas, notas y envío electrónico a SUNAT mediante proveedor.
- Recetas, insumos, compras, movimientos y stock por local.
- Pedidos por WhatsApp con confirmación humana y trazabilidad.
- Dashboard, cierre de caja y reportes de ventas, costos e inventario.

## Fuera de la primera fase

- Marketplace público de restaurantes.
- Aplicación nativa iOS/Android.
- Flota logística avanzada.
- Contabilidad general completa.
- Motor propio de homologación SUNAT.

## Suscripción SaaS

El alta comercial de una empresa se realiza desde Plataforma y exige un plan
activo. El plan define moneda, precio mensual y anual, días de prueba, límites
de locales y usuarios, versión de condiciones y módulos incluidos. La
suscripción de cada empresa conserva el precio contratado, ciclo, estado
(`trial`, `active`, `past_due` o `cancelled`), periodo vigente, fecha de
renovación y aceptación de condiciones.

Los pagos de suscripción se registran separados de los pagos de pedidos. El
modelo es independiente del proveedor para permitir integrar posteriormente una
pasarela sin acoplar el dominio. Los cambios de plan aplican los módulos del
nuevo entitlement y no permiten bajar a límites inferiores al uso activo.
