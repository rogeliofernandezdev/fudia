# Mapa de módulos

## Backend

| Módulo | Responsabilidad |
| --- | --- |
| Identity | sesión, usuarios, roles, permisos y dispositivos |
| Organizations | empresas, locales, áreas, mesas, cajas y series |
| Menu | categorías, productos, recetas, combos, modificadores y precios |
| Service | reservas, turnos de mesa, comandas y estado del servicio |
| Orders | pedidos de salón, mostrador, recojo, delivery y WhatsApp |
| Kitchen | estaciones, tickets, cursos, tiempos y estados KDS |
| Billing | cuentas, descuentos, propina, pagos, caja y cierres |
| EInvoicing | boletas, facturas, notas, XML/CDR y estado SUNAT |
| Inventory | insumos, lotes, movimientos, conteos, mermas y stock |
| Purchasing | proveedores, órdenes de compra y recepciones |
| Customers | identidad, datos tributarios, direcciones y preferencias |
| Delivery | despacho, repartidor, estado y evidencia de entrega |
| Messaging | WhatsApp, notificaciones, plantillas y conversaciones |
| Reporting | agregados operativos, ventas, costos y auditoría |

## Web operativa

- Inicio de turno y selección de local/caja.
- Mapa de mesas y apertura de servicio.
- POS visual con categorías, búsqueda y modificadores.
- Comanda y seguimiento por estación.
- KDS para cocina/bar.
- Cobro, división, comprobante e impresión.
- Cola de pedidos de mostrador, recojo, delivery y WhatsApp.
- Cierre de caja y sincronización.

## Web administrativa

- Dashboard ejecutivo y operativo.
- Menú, recetas y precios.
- Inventario, compras, proveedores y mermas.
- Ventas, comprobantes y cierres.
- Clientes y canales.
- Locales, mesas, cajas, estaciones, usuarios y permisos.
- Onboarding protegido de plataforma: empresa, perfil fiscal, primer local y administrador en una sola transacción.
- Integraciones y reportes.


## Fudia Concierge

- Recepción de mensajes de WhatsApp y validación del webhook de Meta.
- Resolución del contexto iniciado por QR.
- Interpretación conversacional mediante LLM y herramientas deterministas.
- Carrito efímero y confirmación explícita.
- Consulta server-to-server del menú operativo en foods-backend.
- Registro idempotente del pedido real en foods-backend.
- Entrega del pedido confirmado al flujo existente de Pedidos/KDS.

## Límites

Los módulos no acceden a tablas de otros módulos mediante repositorios ajenos.
La coordinación síncrona ocurre mediante servicios de aplicación explícitos;
las integraciones externas se encapsulan detrás de puertos. Fudia Concierge no
accede directamente a PostgreSQL de Fudia ni replica el dominio de pedidos.
